package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Device is a registered agent. The API key is stored only as a bcrypt hash.
type Device struct {
	ID           int64
	DeviceID     string
	APIKeyHash   string
	Hostname     string
	OS           string
	Arch         string
	AgentVersion string
	CreatedAt    time.Time
	LastSeenAt   *time.Time
}

// CreateDevice registers a new device with its hashed API key.
func (s *Store) CreateDevice(ctx context.Context, deviceID, apiKeyHash string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO devices (device_id, api_key_hash) VALUES ($1, $2)`,
		deviceID, apiKeyHash)
	return err
}

// GetDevice looks up a device by its stable id. Returns ErrNotFound if unknown.
func (s *Store) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	var d Device
	err := s.pool.QueryRow(ctx, `
		SELECT id, device_id, api_key_hash,
		       COALESCE(hostname, ''), COALESCE(os, ''), COALESCE(arch, ''),
		       COALESCE(agent_version, ''), created_at, last_seen_at
		FROM devices WHERE device_id = $1`, deviceID).
		Scan(&d.ID, &d.DeviceID, &d.APIKeyHash, &d.Hostname, &d.OS, &d.Arch, &d.AgentVersion, &d.CreatedAt, &d.LastSeenAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// UpdateDeviceSeen refreshes the metadata and last-seen time after a heartbeat.
func (s *Store) UpdateDeviceSeen(ctx context.Context, deviceID, hostname, osName, arch, agentVersion string, ts time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE devices
		SET hostname = $2, os = $3, arch = $4, agent_version = $5, last_seen_at = $6
		WHERE device_id = $1`,
		deviceID, hostname, osName, arch, agentVersion, ts)
	return err
}
