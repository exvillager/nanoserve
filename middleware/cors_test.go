package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/libsib/nanoserve"
	"github.com/libsib/nanoserve/middleware"
)

// newServer spins up a real *http.Server (via httptest) running a nanoserve app
// with the given CORS middleware installed globally plus a couple of routes.
func newServer(t *testing.T, mw nanoserve.HandlerFunction) *httptest.Server {
	t.Helper()
	app := nanoserve.New()
	app.Use(mw)
	app.GET("/data", func(c *nanoserve.Context) error { return c.String("ok") })
	app.POST("/data", func(c *nanoserve.Context) error { return c.String("created") })

	srv := httptest.NewServer(app)
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, srv *httptest.Server, method, path string, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func TestCORS_SimpleRequestDefault(t *testing.T) {
	srv := newServer(t, middleware.CORS())

	resp := do(t, srv, http.MethodGet, "/data", map[string]string{"Origin": "https://example.com"})
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want 200", resp.StatusCode)
	}
	if string(body) != "ok" {
		t.Fatalf("body: got %q want %q", body, "ok")
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow-origin: got %q want %q", got, "*")
	}
}

func TestCORS_NonCORSRequestUntouched(t *testing.T) {
	srv := newServer(t, middleware.CORS())

	resp := do(t, srv, http.MethodGet, "/data", nil) // no Origin header
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS header, got %q", got)
	}
}

func TestCORS_PreflightNoContent(t *testing.T) {
	srv := newServer(t, middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: "https://example.com",
		AllowMethods: "GET,POST",
		MaxAge:       600,
	}))

	resp := do(t, srv, http.MethodOptions, "/data", map[string]string{
		"Origin":                         "https://example.com",
		"Access-Control-Request-Method":  "POST",
		"Access-Control-Request-Headers": "X-Custom",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d want 204", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("allow-origin: got %q", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Methods"); got != "GET,POST" {
		t.Fatalf("allow-methods: got %q", got)
	}
	// AllowHeaders empty -> reflects request headers.
	if got := resp.Header.Get("Access-Control-Allow-Headers"); got != "X-Custom" {
		t.Fatalf("allow-headers: got %q want %q", got, "X-Custom")
	}
	if got := resp.Header.Get("Access-Control-Max-Age"); got != "600" {
		t.Fatalf("max-age: got %q want 600", got)
	}
}

func TestCORS_DisallowedOriginNoHeaders(t *testing.T) {
	srv := newServer(t, middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: "https://allowed.com",
	}))

	resp := do(t, srv, http.MethodGet, "/data", map[string]string{"Origin": "https://evil.com"})
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allow-origin for disallowed origin, got %q", got)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want 200 (request still served)", resp.StatusCode)
	}
}

func TestCORS_CredentialsReflectsOrigin(t *testing.T) {
	// "*" + credentials must reflect the concrete origin, never "*".
	srv := newServer(t, middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     "*",
		AllowCredentials: true,
		ExposeHeaders:    "X-Total-Count",
	}))

	resp := do(t, srv, http.MethodGet, "/data", map[string]string{"Origin": "https://example.com"})
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("allow-origin: got %q want reflected origin", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow-credentials: got %q want true", got)
	}
	if got := resp.Header.Get("Access-Control-Expose-Headers"); got != "X-Total-Count" {
		t.Fatalf("expose-headers: got %q", got)
	}
}
