package nanoserve

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
)

// lookupFn is the common signature for Search and Find so we can run the same
// test cases against both methods without duplicating anything.
type lookupFn func(method, path string) *RouteMatch

func dummyHandler(c *Context) error { return nil }

// runLookupTests runs a shared set of test cases against any lookupFn — used
// to test both Search and Find with identical expectations.
func runLookupTests(t *testing.T, lookup lookupFn, r *TrieRouter) {
	t.Helper()

	t.Run("static route match", func(t *testing.T) {
		r.Insert("GET", "/users", dummyHandler)

		match := lookup("GET", "/users")

		if len(match.Handler) == 0 {
			t.Fatal("expected a handler, got none")
		}
	})

	t.Run("param route extracts value", func(t *testing.T) {
		r.Insert("GET", "/users/:id", dummyHandler)

		match := lookup("GET", "/users/42")

		if len(match.Handler) == 0 {
			t.Fatal("expected a handler, got none")
		}
		if match.Params.Get("id") != "42" {
			t.Fatalf("expected param id=42, got %q", match.Params.Get("id"))
		}
	})

	t.Run("unregistered path returns no handler", func(t *testing.T) {
		match := lookup("GET", "/does/not/exist")
		fmt.Println(match.Handler)
		if len(match.Handler) != 0 {
			t.Fatalf("expected no handler for unknown path, got %d", len(match.Handler))
		}
	})

}

// newLookup is a factory: given a fresh router, return the bound lookup method.
// This lets each subtest own its router while sharing test logic across Search/Find.
func testMiddlewares(t *testing.T, newLookup func(*TrieRouter) lookupFn) {
	t.Helper()

	t.Run("global middlewares present even when no route matches", func(t *testing.T) {
		r := NewTrieRouter()
		lookup := newLookup(r)
		r.AddMiddleware("/", func(ctx *Context) error { return ctx.Next() })
		r.AddMiddleware("/", func(ctx *Context) error { return ctx.Next() })

		match := lookup("GET", "/no-match")

		if len(match.Handler) != 2 {
			t.Fatalf("expected 2 global middlewares, got %d", len(match.Handler))
		}
	})

	t.Run("path-scoped middleware only appears for matching prefix", func(t *testing.T) {
		r := NewTrieRouter()
		lookup := newLookup(r)
		r.AddMiddleware("/api/*", func(ctx *Context) error { return ctx.Next() })
		r.Insert("GET", "/api/users", func(ctx *Context) error { return nil })
		r.Insert("GET", "/other", func(ctx *Context) error { return nil })

		apiMatch := lookup("GET", "/api/users")

		if len(apiMatch.Handler) != 2 { // 1 middleware + 1 handler
			t.Fatalf("/api/users: expected 2 handlers (middleware+handler), got %d", len(apiMatch.Handler))
		}
		otherMatch := lookup("GET", "/other")
		if len(otherMatch.Handler) != 1 { // handler only, no /api middleware
			t.Fatalf("/other: expected 1 handler (no middleware), got %d", len(otherMatch.Handler))
		}
	})

	t.Run("global middleware comes before route handler", func(t *testing.T) {
		r := NewTrieRouter()
		lookup := newLookup(r)
		order := []string{}
		r.AddMiddleware("/", func(ctx *Context) error {
			order = append(order, "middleware")
			return nil
		})
		r.Insert("GET", "/ping", func(ctx *Context) error {
			order = append(order, "handler")
			return nil
		})

		match := lookup("GET", "/ping")

		if len(match.Handler) != 2 {
			t.Fatalf("expected 2 handlers, got %d", len(match.Handler))
		}
		match.Handler[0](nil)
		match.Handler[1](nil)
		if order[0] != "middleware" || order[1] != "handler" {
			t.Fatalf("wrong execution order: %v", order)
		}
	})

	t.Run("it shouldnt include /user midl", func(t *testing.T) {
		r := NewTrieRouter()
		lookup := newLookup(r)

		r.AddMiddleware("/user", func(ctx *Context) error {return nil})
		r.Insert("GET","/user/me",func(ctx *Context) error {return nil})

		match := lookup("GET", "/user/me")

		if len(match.Handler) != 1 {
			t.Fatalf("expected 1, got %d",len(match.Handler))
		}
	})
}

func runParamLookupTests(t *testing.T, lookup lookupFn, r *TrieRouter) {
	t.Helper()

	t.Run("single param segment", func(t *testing.T) {
		r.Insert("GET", "/some/:id", dummyHandler)

		match := lookup("GET", "/some/42")

		if len(match.Handler) == 0 {
			t.Fatal("expected a handler, got none")
		}
		if match.Params.Get("id") != "42" {
			t.Fatalf("expected id=42, got %q", match.Params.Get("id"))
		}
	})

	t.Run("multiple param segments", func(t *testing.T) {
		r.Insert("GET", "/orgs/:org/repos/:repo", dummyHandler)

		match := lookup("GET", "/orgs/acme/repos/widget")

		if len(match.Handler) == 0 {
			t.Fatal("expected a handler, got none")
		}
		if match.Params.Get("org") != "acme" {
			t.Fatalf("expected org=acme, got %q", match.Params.Get("org"))
		}
		if match.Params.Get("repo") != "widget" {
			t.Fatalf("expected repo=widget, got %q", match.Params.Get("repo"))
		}
	})

	t.Run("param does not match static sibling", func(t *testing.T) {
		r.Insert("GET", "/items/featured", dummyHandler)
		r.Insert("GET", "/items/:id", dummyHandler)

		staticMatch := lookup("GET", "/items/featured")
		if len(staticMatch.Handler) == 0 {
			t.Fatal("/items/featured: expected handler, got none")
		}
		
		paramMatch := lookup("GET", "/items/99")
		if len(paramMatch.Handler) == 0 {
			t.Fatal("/items/99: expected handler, got none")
		}
		if paramMatch.Params.Get("id") != "99" {
			t.Fatalf("expected id=99, got %q", paramMatch.Params.Get("id"))
		}
	})

	t.Run("unregistered param path returns no handler", func(t *testing.T) {
		match := lookup("GET", "/some/42/extra/segments")
		if len(match.Handler) != 0 {
			t.Fatalf("expected no handler for deep unregistered path, got %d", len(match.Handler))
		}
	})
}

func TestParamSearch(t *testing.T) {
	r := NewTrieRouter()
	runParamLookupTests(t, r.Search, r)
}

func TestParamFind(t *testing.T) {
	r := NewTrieRouter()
	runParamLookupTests(t, r.Find, r)
}

// runParamCollisionTests guards against the bug where two methods sharing the
// same ":" node (e.g. GET /users/:id and POST /users/:name) overwrote each
// other's param name, so a GET lookup reported the param under "name" instead
// of "id".
func runParamCollisionTests(t *testing.T, lookup lookupFn, r *TrieRouter) {
	t.Helper()

	r.Insert("GET", "/users/:id", dummyHandler)
	r.Insert("POST", "/users/:name", dummyHandler)

	getMatch := lookup("GET", "/users/42")
	if len(getMatch.Handler) == 0 {
		t.Fatal("GET /users/42: expected a handler, got none")
	}
	if got := getMatch.Params.Get("id"); got != "42" {
		t.Fatalf("GET /users/42: expected id=42, got id=%q (params: %+v)", got, getMatch.Params)
	}

	postMatch := lookup("POST", "/users/alice")
	if len(postMatch.Handler) == 0 {
		t.Fatal("POST /users/alice: expected a handler, got none")
	}
	if got := postMatch.Params.Get("name"); got != "alice" {
		t.Fatalf("POST /users/alice: expected name=alice, got name=%q (params: %+v)", got, postMatch.Params)
	}
}

func TestParamCollisionSearch(t *testing.T) {
	r := NewTrieRouter()
	runParamCollisionTests(t, r.Search, r)
}

func TestParamCollisionFind(t *testing.T) {
	r := NewTrieRouter()
	runParamCollisionTests(t, r.Find, r)
}

func TestSearch(t *testing.T) {
	r := NewTrieRouter()
	runLookupTests(t, r.Search, r)
}

func TestFind(t *testing.T) {
	r := NewTrieRouter()
	runLookupTests(t, r.Find, r)
}

func TestMiddlewaresSearch(t *testing.T) {
	testMiddlewares(t, func(r *TrieRouter) lookupFn { return r.Search })
}

func TestMiddlewaresFind(t *testing.T) {
	testMiddlewares(t, func(r *TrieRouter) lookupFn { return r.Find })
}

// sameChain reports whether two handler chains hold the same functions in the
// same order.
func sameChain(a, b []HandlerFunction) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if reflect.ValueOf(a[i]).Pointer() != reflect.ValueOf(b[i]).Pointer() {
			return false
		}
	}
	return true
}

// TestStaticCache checks that static paths are served from the cache and that
// the cached result matches what the trie walk (Search) returns, including
// middleware registered after the routes.
func TestStaticCache(t *testing.T) {
	r := NewTrieRouter()
	getUsers := func(c *Context) error { return nil }
	postUsers := func(c *Context) error { return nil }
	allHealth := func(c *Context) error { return nil }
	apiMw := func(c *Context) error { return c.Next() }
	globalMw := func(c *Context) error { return c.Next() }

	r.Insert("GET", "/api/users", getUsers)
	r.Insert("POST", "/api/users", postUsers)
	r.Insert("GET", "/api/users/:id", dummyHandler)
	r.Insert("ALL", "/health", allHealth)
	r.AddMiddleware("/api/*", apiMw)
	r.AddMiddleware("/", globalMw)

	cases := []struct {
		method, path string
		want         []HandlerFunction
	}{
		{"GET", "/api/users", []HandlerFunction{globalMw, apiMw, getUsers}},
		{"POST", "/api/users", []HandlerFunction{globalMw, apiMw, postUsers}},
		{"GET", "/health", []HandlerFunction{globalMw, allHealth}},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			if r.getStaticMapFor(tc.method)[tc.path] == nil {
				t.Fatal("expected a static cache entry, got none")
			}
			found := r.Find(tc.method, tc.path)
			if !sameChain(found.Handler, tc.want) {
				t.Fatalf("Find: expected %d handlers in order, got %d", len(tc.want), len(found.Handler))
			}
			if !sameChain(found.Handler, r.Search(tc.method, tc.path).Handler) {
				t.Fatal("Find (cache) and Search (trie) returned different chains")
			}
		})
	}

	t.Run("param paths are not cached", func(t *testing.T) {
		if slices.Contains(r.staticPaths, "/api/users/:id") {
			t.Fatalf("param path should not be cached, staticPaths=%v", r.staticPaths)
		}
		if got := r.Find("GET", "/api/users/42").Params.Get("id"); got != "42" {
			t.Fatalf("expected id=42, got %q", got)
		}
	})
}
