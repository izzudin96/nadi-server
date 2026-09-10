package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/izzudin96/nadi-server/internal/auth"
	"github.com/izzudin96/nadi-server/internal/store"
)

const (
	maxMetrics = 100
	maxNameLen = 64
	maxBody    = 1 << 20 // 1 MiB
)

// metricNameRe mirrors the agent's naming contract (PLAN §4.3): lowercase,
// dot-separated segments.
var metricNameRe = regexp.MustCompile(`^[a-z0-9_]+(\.[a-z0-9_]+)+$`)

// heartbeat is the wire format POSTed by the agent (spec §6). Duplicated from
// the agent per PLAN §4.1 — keep the small struct instead of a shared module.
type heartbeat struct {
	DeviceID  string   `json:"device_id"`
	Timestamp string   `json:"timestamp"`
	Metrics   []metric `json:"metrics"`
	Meta      struct {
		Hostname     string `json:"hostname"`
		OS           string `json:"os"`
		Arch         string `json:"arch"`
		AgentVersion string `json:"agent_version"`
	} `json:"meta"`
}

type metric struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
	if err != nil {
		http.Error(w, "reading body", http.StatusBadRequest)
		return
	}

	var hb heartbeat
	if err := json.Unmarshal(body, &hb); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if hb.DeviceID == "" {
		http.Error(w, "missing device_id", http.StatusBadRequest)
		return
	}
	if len(hb.Metrics) > maxMetrics {
		http.Error(w, "too many metrics", http.StatusBadRequest)
		return
	}
	for _, m := range hb.Metrics {
		if !validMetricName(m.Name) {
			http.Error(w, "invalid metric name: "+m.Name, http.StatusBadRequest)
			return
		}
	}

	key := bearerToken(r.Header.Get("Authorization"))
	if key == "" {
		http.Error(w, "missing api key", http.StatusUnauthorized)
		return
	}

	dev, err := s.store.GetDevice(r.Context(), hb.DeviceID)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err != nil {
		s.logger.Error("device lookup failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !auth.CheckSecret(dev.APIKeyHash, key) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ts, err := time.Parse(time.RFC3339, hb.Timestamp)
	if err != nil {
		ts = time.Now().UTC()
	}

	metrics := make([]store.Metric, 0, len(hb.Metrics))
	for _, m := range hb.Metrics {
		metrics = append(metrics, store.Metric{Name: m.Name, Value: m.Value, Unit: m.Unit})
	}

	if err := s.store.UpdateDeviceSeen(r.Context(), hb.DeviceID, hb.Meta.Hostname, hb.Meta.OS, hb.Meta.Arch, hb.Meta.AgentVersion, ts); err != nil {
		s.logger.Error("updating device", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := s.store.InsertMetrics(r.Context(), hb.DeviceID, metrics, ts); err != nil {
		s.logger.Error("inserting metrics", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return header[len(prefix):]
	}
	return ""
}

func validMetricName(name string) bool {
	return len(name) <= maxNameLen && metricNameRe.MatchString(name)
}
