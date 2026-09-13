package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/izzudin96/nadi-server/internal/auth"
	"github.com/izzudin96/nadi-server/internal/store"
)

// onlineThreshold is how long since a heartbeat a device is still "online".
// Placeholder for the "missed N heartbeats" open question (PLAN §8).
const onlineThreshold = 2 * time.Minute

const defaultSeriesWindow = time.Hour

// targetSeriesPoints is how many points a downsampled series aims for. The
// series stride is derived from the requested window so every range returns a
// bounded, chart-friendly number of points.
const targetSeriesPoints = 600

type deviceView struct {
	DeviceID     string     `json:"device_id"`
	Hostname     string     `json:"hostname"`
	OS           string     `json:"os"`
	Arch         string     `json:"arch"`
	AgentVersion string     `json:"agent_version"`
	LastSeenAt   *time.Time `json:"last_seen_at"`
	Online       bool       `json:"online"`
}

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := s.store.ListDevices(r.Context())
	if err != nil {
		s.logger.Error("listing devices", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	views := make([]deviceView, 0, len(devices))
	for _, d := range devices {
		views = append(views, deviceView{
			DeviceID:     d.DeviceID,
			Hostname:     d.Hostname,
			OS:           d.OS,
			Arch:         d.Arch,
			AgentVersion: d.AgentVersion,
			LastSeenAt:   d.LastSeenAt,
			Online:       d.LastSeenAt != nil && now.Sub(*d.LastSeenAt) < onlineThreshold,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": views})
}

// deviceIDRe constrains dashboard-created device ids. Existing agents may use
// looser ids; the heartbeat path stays permissive for backward compatibility.
var deviceIDRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

type createDeviceRequest struct {
	DeviceID string `json:"device_id"`
}

type deviceKeyView struct {
	DeviceID string `json:"device_id"`
	APIKey   string `json:"api_key"`
}

// handleCreateDevice registers a device and returns its API key once. The key
// is never retrievable again (only its hash is stored).
func (s *Server) handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var req createDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	deviceID := strings.TrimSpace(req.DeviceID)
	if !deviceIDRe.MatchString(deviceID) {
		http.Error(w, "device_id must be 1-64 chars of letters, digits, dot, dash or underscore", http.StatusBadRequest)
		return
	}

	if _, err := s.store.GetDevice(r.Context(), deviceID); err == nil {
		http.Error(w, "device already exists", http.StatusConflict)
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		s.logger.Error("checking device", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	key, hash, err := newDeviceKey()
	if err != nil {
		s.logger.Error("generating device key", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := s.store.CreateDevice(r.Context(), deviceID, hash); err != nil {
		s.logger.Error("creating device", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, deviceKeyView{DeviceID: deviceID, APIKey: key})
}

// handleRotateDevice issues a fresh API key for an existing device.
func (s *Server) handleRotateDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceID")

	key, hash, err := newDeviceKey()
	if err != nil {
		s.logger.Error("generating device key", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := s.store.RotateDeviceKey(r.Context(), deviceID, hash); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		s.logger.Error("rotating device key", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, deviceKeyView{DeviceID: deviceID, APIKey: key})
}

// handleDeleteDevice removes a device and its stored metrics.
func (s *Server) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceID")
	if err := s.store.DeleteDevice(r.Context(), deviceID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		s.logger.Error("deleting device", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func newDeviceKey() (key, hash string, err error) {
	key, err = auth.GenerateAPIKey()
	if err != nil {
		return "", "", err
	}
	hash, err = auth.HashSecret(key)
	if err != nil {
		return "", "", err
	}
	return key, hash, nil
}

func (s *Server) handleLatest(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceID")

	metrics, err := s.store.LatestMetrics(r.Context(), deviceID)
	if err != nil {
		s.logger.Error("fetching latest metrics", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"device_id": deviceID, "metrics": metrics})
}

func (s *Server) handleMetricSeries(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceID")
	metricName := chi.URLParam(r, "metricName")

	now := time.Now()
	from := parseTime(r.URL.Query().Get("from"), now.Add(-defaultSeriesWindow))
	to := parseTime(r.URL.Query().Get("to"), now)
	if !to.After(from) {
		http.Error(w, "to must be after from", http.StatusBadRequest)
		return
	}

	// Downsample to a bounded number of points; for short windows the stride
	// falls below the heartbeat interval so the series is effectively raw.
	stride := to.Sub(from) / targetSeriesPoints
	if stride < time.Second {
		stride = time.Second
	}

	points, err := s.store.MetricSeriesBucketed(r.Context(), deviceID, metricName, from, to, stride)
	if err != nil {
		s.logger.Error("fetching metric series", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"points": points})
}

func parseTime(s string, def time.Time) time.Time {
	if s == "" {
		return def
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return def
	}
	return t
}
