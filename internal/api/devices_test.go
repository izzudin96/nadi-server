package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func requestJSON(t *testing.T, method, url, path string, cookie *http.Cookie, body any) *http.Response {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func registerUser(t *testing.T, srv *httptest.Server, email string) *http.Cookie {
	t.Helper()
	resp := postJSON(t, srv.URL, "/api/auth/register", credentials{Email: email, Password: "password123"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d, want 201", resp.StatusCode)
	}
	for _, c := range resp.Cookies() {
		if c.Name == cookieName {
			return c
		}
	}
	t.Fatal("no auth cookie returned")
	return nil
}

func seedHeartbeat(t *testing.T, srv *httptest.Server, deviceID, key string) {
	t.Helper()
	hb := sampleHeartbeat(deviceID)
	hb.Timestamp = time.Now().UTC().Format(time.RFC3339)
	postHeartbeat(t, srv.URL, deviceID, key, hb)
}

func getWithCookie(t *testing.T, url, path string, cookie *http.Cookie) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestDashboardEndpoints(t *testing.T) {
	srv, st, _ := newTestServer(t)
	cookie := registerUser(t, srv, "dash@example.com")

	registerTestDevice(t, st, "dev-1", "secret-key")
	seedHeartbeat(t, srv, "dev-1", "secret-key")

	// List devices.
	resp := getWithCookie(t, srv.URL, "/api/devices", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list devices status = %d", resp.StatusCode)
	}
	var list struct {
		Devices []deviceView `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list.Devices) != 1 || list.Devices[0].DeviceID != "dev-1" || !list.Devices[0].Online {
		t.Fatalf("devices = %+v", list.Devices)
	}

	// Latest metrics snapshot.
	resp2 := getWithCookie(t, srv.URL, "/api/devices/dev-1/latest", cookie)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("latest status = %d", resp2.StatusCode)
	}
	var latest struct {
		Metrics []struct {
			Name  string  `json:"name"`
			Value float64 `json:"value"`
		} `json:"metrics"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&latest); err != nil {
		t.Fatal(err)
	}
	if len(latest.Metrics) != 2 {
		t.Fatalf("latest metrics = %+v, want 2", latest.Metrics)
	}

	// Metric time series.
	resp3 := getWithCookie(t, srv.URL, "/api/devices/dev-1/metrics/cpu.usage_percent", cookie)
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("series status = %d", resp3.StatusCode)
	}
	var series struct {
		Points []struct {
			Value float64 `json:"value"`
		} `json:"points"`
	}
	if err := json.NewDecoder(resp3.Body).Decode(&series); err != nil {
		t.Fatal(err)
	}
	if len(series.Points) != 1 || series.Points[0].Value != 42.5 {
		t.Fatalf("series = %+v, want one point with value 42.5", series.Points)
	}
}

func TestMetricSeriesRejectsInvalidRange(t *testing.T) {
	srv, _, _ := newTestServer(t)
	cookie := registerUser(t, srv, "range@example.com")

	// to before from is rejected rather than silently returning nothing.
	resp := getWithCookie(t, srv.URL,
		"/api/devices/dev-1/metrics/cpu.usage_percent?from=2026-01-02T00:00:00Z&to=2026-01-01T00:00:00Z", cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestDeviceManagement(t *testing.T) {
	srv, _, _ := newTestServer(t)
	cookie := registerUser(t, srv, "mgmt@example.com")

	// Create a device: the API key is returned exactly once.
	resp := requestJSON(t, http.MethodPost, srv.URL, "/api/devices", cookie, createDeviceRequest{DeviceID: "dev-mgmt"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create device status = %d, want 201", resp.StatusCode)
	}
	var created deviceKeyView
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.DeviceID != "dev-mgmt" || created.APIKey == "" {
		t.Fatalf("created = %+v", created)
	}

	// Creating the same id again is a conflict, not a silent key rotation.
	if dup := requestJSON(t, http.MethodPost, srv.URL, "/api/devices", cookie, createDeviceRequest{DeviceID: "dev-mgmt"}); dup.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate create status = %d, want 409", dup.StatusCode)
	}

	// The generated key authenticates heartbeats.
	if hb := postHeartbeat(t, srv.URL, "dev-mgmt", created.APIKey, sampleHeartbeat("dev-mgmt")); hb.StatusCode != http.StatusOK {
		t.Fatalf("heartbeat with new key = %d, want 200", hb.StatusCode)
	}

	// Rotate: the old key stops working, the new one works.
	rotateResp := requestJSON(t, http.MethodPost, srv.URL, "/api/devices/dev-mgmt/rotate", cookie, nil)
	if rotateResp.StatusCode != http.StatusOK {
		t.Fatalf("rotate status = %d, want 200", rotateResp.StatusCode)
	}
	var rotated deviceKeyView
	if err := json.NewDecoder(rotateResp.Body).Decode(&rotated); err != nil {
		t.Fatal(err)
	}
	if rotated.APIKey == "" || rotated.APIKey == created.APIKey {
		t.Fatalf("rotate key = %q, want a new key", rotated.APIKey)
	}
	if hb := postHeartbeat(t, srv.URL, "dev-mgmt", created.APIKey, sampleHeartbeat("dev-mgmt")); hb.StatusCode != http.StatusUnauthorized {
		t.Fatalf("heartbeat with rotated-away key = %d, want 401", hb.StatusCode)
	}
	if hb := postHeartbeat(t, srv.URL, "dev-mgmt", rotated.APIKey, sampleHeartbeat("dev-mgmt")); hb.StatusCode != http.StatusOK {
		t.Fatalf("heartbeat with new key = %d, want 200", hb.StatusCode)
	}

	// Delete: subsequent heartbeats and a second delete both 404/401.
	if del := requestJSON(t, http.MethodDelete, srv.URL, "/api/devices/dev-mgmt", cookie, nil); del.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", del.StatusCode)
	}
	if hb := postHeartbeat(t, srv.URL, "dev-mgmt", rotated.APIKey, sampleHeartbeat("dev-mgmt")); hb.StatusCode != http.StatusUnauthorized {
		t.Fatalf("heartbeat after delete = %d, want 401", hb.StatusCode)
	}
	if del := requestJSON(t, http.MethodDelete, srv.URL, "/api/devices/dev-mgmt", cookie, nil); del.StatusCode != http.StatusNotFound {
		t.Fatalf("second delete status = %d, want 404", del.StatusCode)
	}
}

func TestDeviceManagementRequiresAuth(t *testing.T) {
	srv, _, _ := newTestServer(t)

	for _, tc := range []struct {
		method, path string
	}{
		{http.MethodPost, "/api/devices"},
		{http.MethodPost, "/api/devices/dev-1/rotate"},
		{http.MethodDelete, "/api/devices/dev-1"},
	} {
		resp := requestJSON(t, tc.method, srv.URL, tc.path, nil, createDeviceRequest{DeviceID: "dev-1"})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s %s without auth = %d, want 401", tc.method, tc.path, resp.StatusCode)
		}
	}
}

func TestDashboardEndpointsRequireAuth(t *testing.T) {
	srv, _, _ := newTestServer(t)

	for _, path := range []string{"/api/devices", "/api/devices/dev-1/latest", "/api/devices/dev-1/metrics/cpu.usage_percent"} {
		resp := getWithCookie(t, srv.URL, path, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s without auth = %d, want 401", path, resp.StatusCode)
		}
	}
}
