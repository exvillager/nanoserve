# nanoServe

nanoServe is a tiny, fast web framework for Go — think **Fiber**, but smaller. It bundles a trie-based router, a request `Context` (JSON/text/HTML responses, params, query, cookies, body binding), and middleware chaining, all with zero third-party dependencies.

---

## Install

```bash
go get github.com/libsib/nanoserve
```

---

## Quick Start

```go
package main

import "github.com/libsib/nanoserve"

func main() {
	app := nanoserve.New()

	app.Use(func(c *nanoserve.Context) error {
		println("request:", c.Request.URL.Path)
		return c.Next()
	})

	app.GET("/hello/:name", func(c *nanoserve.Context) error {
		return c.JSON(map[string]string{"hello": c.Param("name")})
	})

	app.Run(":8080")
}
```

Visit [http://localhost:8080/hello/world](http://localhost:8080/hello/world)

---

## Routing

All standard HTTP methods are supported:

```go
app.GET(path, handlers...)
app.POST(path, handlers...)
app.PUT(path, handlers...)
app.PATCH(path, handlers...)
app.DELETE(path, handlers...)
app.HEAD(path, handlers...)
app.OPTIONS(path, handlers...)
app.CONNECT(path, handlers...)
app.TRACE(path, handlers...)

app.Handle(method, path, handlers...) // any custom method
app.ANY(path, handlers...)            // matches every method
```

Route params use a `:` prefix and are read back with `c.Param`:

```go
app.GET("/users/:id", func(c *nanoserve.Context) error {
	return c.Text("user " + c.Param("id"))
})
```

A trailing `/*` matches everything after the prefix (used by `Static` and `Sub` below).

---

## Middleware

A handler can `return c.Next()` to continue the chain, or stop it early with `c.Abort()` / `c.AbortWithStatus(code)`.

```go
// global — runs on every request
app.Use(func(c *nanoserve.Context) error {
	return c.Next()
})

// scoped to a path prefix
app.Use("/api", func(c *nanoserve.Context) error {
	return c.Next()
})

// per-route — pass extra handlers before the final one
app.GET("/admin", authMiddleware, adminHandler)
```

Middleware runs in registration order, global middleware first.

---

## Context

The `Context` passed to every handler wraps the request and response:

```go
c.Param("id")          // route param
c.Query("q")           // URL query param
c.GetHeader("X-Token") // request header
c.SetHeader("X-Foo", "bar")
c.Set("user", u)       // stash a value for this request
c.Get("user")          // retrieve it later in the chain

c.Text("plain text")
c.JSON(data)
c.HTML("<h1>hi</h1>")
c.Send(data, "application/octet-stream")
c.NoContent(204)

c.Bind(&payload)       // decode JSON body
c.BodyBytes()          // raw body, safe to call alongside Bind

c.GetCookie("session")
c.SetCookie(http.Cookie{Name: "session", Value: "..."})
c.Redirect("/login", 302)
c.IP()                 // best-effort client IP (X-Forwarded-For / X-Real-IP / RemoteAddr)

c.Status(200).JSON(data) // chain a status code before writing
```

Handlers return an `error`; a non-nil error is passed to `app.ErrorHandler` (which defaults to a `500` response, and can be overridden).

---

## Static Files

```go
app.Static("/assets", "./public")
```

---

## Sub-routers

Mount one `NanoServe` instance inside another under a prefix:

```go
api := nanoserve.New()
api.GET("/users", listUsers)

app := nanoserve.New()
app.Sub("/api/*", api)
```

---

## Running

```go
app.Run(":8080") // shorthand for http.ListenAndServe(addr, app)
```

`NanoServe` also implements `http.Handler` directly, so it drops into any existing `net/http` server:

```go
http.ListenAndServe(":8080", app)
```

---

## Philosophy

* Small core, no dependencies
* Idiomatic Go — no magic, no reflection-heavy DSL
* Fast by default: trie routing, pooled request contexts
* Easy to read end to end in one sitting

---

## License

MIT License

---

## Contributing

Contributions are welcome. Feel free to open an issue or a PR.
