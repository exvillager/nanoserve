package nanoserve

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func serve(app *NanoServe, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestNoMatchReturns404WithGlobalMiddleware(t *testing.T) {
	ran := 0
	app := New()
	app.Use(func(c *Context) error {
		ran++
		return c.Next()
	})
	app.GET("/check", func(c *Context) error { return c.String("ok") })

	cases := []struct {
		name, method, path string
	}{
		{"unknown path", http.MethodGet, "/nope"},
		{"wrong method", http.MethodPost, "/check"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ran = 0
			w := serve(app, tc.method, tc.path)
			if w.Code != http.StatusNotFound {
				t.Fatalf("expected 404, got %d", w.Code)
			}
			if ran != 1 {
				t.Fatalf("expected global middleware to run once, ran %d times", ran)
			}
		})
	}
}

func TestMatchedRouteStillRunsHandler(t *testing.T) {
	app := New()
	app.Use(func(c *Context) error { return c.Next() })
	app.GET("/check", func(c *Context) error { return c.String("ok") })

	w := serve(app, http.MethodGet, "/check")
	if w.Code != http.StatusOK || w.Body.String() != "ok" {
		t.Fatalf("expected 200 ok, got %d %q", w.Code, w.Body.String())
	}
}

func TestNoMatchReturns404WithoutMiddleware(t *testing.T) {
	app := New()
	app.GET("/check", func(c *Context) error { return c.String("ok") })

	if w := serve(app, http.MethodPost, "/check"); w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
