// Package middleware provides reusable middleware for nanoserve.
package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/libsib/nanoserve"
)

// CORSConfig defines the configuration for the CORS middleware.
// It mirrors (a small subset of) Fiber's CORS middleware.
type CORSConfig struct {
	// AllowOrigins is a comma separated list of origins that may access the
	// resource. Use "*" to allow any origin. Default: "*".
	AllowOrigins string

	// AllowMethods is a comma separated list of methods the client may use.
	// Default: "GET,POST,HEAD,PUT,DELETE,PATCH".
	AllowMethods string

	// AllowHeaders is a comma separated list of headers the client may use.
	// When empty it reflects the request's Access-Control-Request-Headers.
	AllowHeaders string

	// ExposeHeaders is a comma separated list of headers exposed to the client.
	ExposeHeaders string

	// AllowCredentials indicates whether the response to the request can be
	// exposed when credentials are present. Cannot be used with "*" origin.
	AllowCredentials bool

	// MaxAge indicates how long (in seconds) the preflight result can be cached.
	MaxAge int
}

// DefaultCORSConfig is the default configuration used by CORS().
var DefaultCORSConfig = CORSConfig{
	AllowOrigins: "*",
	AllowMethods: strings.Join([]string{
		http.MethodGet,
		http.MethodPost,
		http.MethodHead,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}, ","),
}

// CORS returns a middleware with the default configuration.
func CORS() nanoserve.HandlerFunction {
	return CORSWithConfig(DefaultCORSConfig)
}

// CORSWithConfig returns a CORS middleware using the given configuration.
func CORSWithConfig(config CORSConfig) nanoserve.HandlerFunction {
	// Apply defaults for unset fields.
	if config.AllowOrigins == "" {
		config.AllowOrigins = DefaultCORSConfig.AllowOrigins
	}
	if config.AllowMethods == "" {
		config.AllowMethods = DefaultCORSConfig.AllowMethods
	}

	// Pre-compute the list of allowed origins for fast matching.
	allowAllOrigins := config.AllowOrigins == "*"
	allowedOrigins := make([]string, 0)
	for _, o := range strings.Split(config.AllowOrigins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			allowedOrigins = append(allowedOrigins, o)
		}
	}
	maxAge := ""
	if config.MaxAge > 0 {
		maxAge = strconv.Itoa(config.MaxAge)
	}

	return func(c *nanoserve.Context) error {
		origin := c.GetHeader("Origin")

		// Not a CORS request; continue the chain untouched.
		if origin == "" {
			return c.Next()
		}

		allowOrigin := ""
		switch {
		case allowAllOrigins && config.AllowCredentials:
			// "*" is not allowed with credentials, so reflect the origin.
			allowOrigin = origin
		case allowAllOrigins:
			allowOrigin = "*"
		case originAllowed(origin, allowedOrigins):
			allowOrigin = origin
		}

		// Origin not permitted: continue without CORS headers.
		if allowOrigin == "" {
			return c.Next()
		}

		c.SetHeader("Access-Control-Allow-Origin", allowOrigin)
		c.SetHeader("Vary", "Origin")
		if config.AllowCredentials {
			c.SetHeader("Access-Control-Allow-Credentials", "true")
		}
		if config.ExposeHeaders != "" {
			c.SetHeader("Access-Control-Expose-Headers", config.ExposeHeaders)
		}

		// Preflight request.
		if c.Method() == http.MethodOptions {
			c.SetHeader("Access-Control-Allow-Methods", config.AllowMethods)

			if config.AllowHeaders != "" {
				c.SetHeader("Access-Control-Allow-Headers", config.AllowHeaders)
			} else if reqHeaders := c.GetHeader("Access-Control-Request-Headers"); reqHeaders != "" {
				c.SetHeader("Access-Control-Allow-Headers", reqHeaders)
			}

			if maxAge != "" {
				c.SetHeader("Access-Control-Max-Age", maxAge)
			}

			c.AbortWithStatus(http.StatusNoContent)
			return nil
		}

		return c.Next()
	}
}

// originAllowed reports whether origin is present in the allowed list.
func originAllowed(origin string, allowed []string) bool {
	for _, o := range allowed {
		if o == origin {
			return true
		}
	}
	return false
}
