package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

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

func TestDashboardEndpointsRequireAuth(t *testing.T) {
	srv, _, _ := newTestServer(t)

	for _, path := range []string{"/api/devices", "/api/devices/dev-1/latest", "/api/devices/dev-1/metrics/cpu.usage_percent"} {
		resp := getWithCookie(t, srv.URL, path, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s without auth = %d, want 401", path, resp.StatusCode)
		}
	}
}
