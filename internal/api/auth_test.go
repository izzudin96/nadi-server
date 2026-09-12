package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/izzudin96/nadi-server/internal/config"
)

func postJSON(t *testing.T, url, path string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url+path, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestRegisterLoginMe(t *testing.T) {
	srv, _, _ := newTestServer(t)

	resp := postJSON(t, srv.URL, "/api/auth/register", credentials{Email: "a@example.com", Password: "password123"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d, want 201", resp.StatusCode)
	}
	var cookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == cookieName {
			cookie = c
		}
	}
	if cookie == nil || cookie.HttpOnly == false {
		t.Fatalf("expected httpOnly auth cookie, got %+v", resp.Cookies())
	}

	// /me with the cookie should succeed.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/auth/me", nil)
	req.AddCookie(cookie)
	meResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer meResp.Body.Close()
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("/me status = %d, want 200", meResp.StatusCode)
	}

	// /me without a cookie should 401.
	req2, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/auth/me", nil)
	anonResp, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer anonResp.Body.Close()
	if anonResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/me without cookie status = %d, want 401", anonResp.StatusCode)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	srv, _, _ := newTestServer(t)

	if resp := postJSON(t, srv.URL, "/api/auth/register", credentials{Email: "dup@example.com", Password: "password123"}); resp.StatusCode != http.StatusCreated {
		t.Fatalf("first register status = %d, want 201", resp.StatusCode)
	}
	if resp := postJSON(t, srv.URL, "/api/auth/register", credentials{Email: "dup@example.com", Password: "password123"}); resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate register status = %d, want 409", resp.StatusCode)
	}
}

func TestRegistrationAutoClosesAfterFirstUser(t *testing.T) {
	srv, _, _ := newTestServerCfg(t, config.Config{JWTSecret: "test-secret", AllowRegistration: "auto"})

	status := func() bool {
		resp := getWithCookie(t, srv.URL, "/api/auth/registration", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("registration status = %d, want 200", resp.StatusCode)
		}
		var body struct {
			Open bool `json:"open"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		return body.Open
	}

	if !status() {
		t.Fatal("registration should be open before the first user")
	}
	if resp := postJSON(t, srv.URL, "/api/auth/register", credentials{Email: "first@example.com", Password: "password123"}); resp.StatusCode != http.StatusCreated {
		t.Fatalf("first register status = %d, want 201", resp.StatusCode)
	}
	if status() {
		t.Fatal("registration should be closed after the first user")
	}
	if resp := postJSON(t, srv.URL, "/api/auth/register", credentials{Email: "second@example.com", Password: "password123"}); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("second register status = %d, want 403", resp.StatusCode)
	}
}

func TestRegistrationAlwaysOpen(t *testing.T) {
	srv, _, _ := newTestServerCfg(t, config.Config{JWTSecret: "test-secret", AllowRegistration: "true"})

	postJSON(t, srv.URL, "/api/auth/register", credentials{Email: "a@example.com", Password: "password123"})
	if resp := postJSON(t, srv.URL, "/api/auth/register", credentials{Email: "b@example.com", Password: "password123"}); resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d, want 201", resp.StatusCode)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	srv, _, _ := newTestServer(t)
	postJSON(t, srv.URL, "/api/auth/register", credentials{Email: "u@example.com", Password: "password123"})

	if resp := postJSON(t, srv.URL, "/api/auth/login", credentials{Email: "u@example.com", Password: "wrongpass"}); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("login status = %d, want 401", resp.StatusCode)
	}
}

func TestLoginUnknownUser(t *testing.T) {
	srv, _, _ := newTestServer(t)
	if resp := postJSON(t, srv.URL, "/api/auth/login", credentials{Email: "nobody@example.com", Password: "password123"}); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("login status = %d, want 401", resp.StatusCode)
	}
}
