package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// onlineThreshold is how long since a heartbeat a device is still "online".
// Placeholder for the "missed N heartbeats" open question (PLAN §8).
const onlineThreshold = 2 * time.Minute

const defaultSeriesWindow = time.Hour
const defaultSeriesLimit = 1000

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

	points, err := s.store.MetricSeries(r.Context(), deviceID, metricName, from, to, defaultSeriesLimit)
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
