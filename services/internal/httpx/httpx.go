// Package httpx provides shared Gin setup: a router with /healthz and a JSON
// error helper, used by all three service binaries.
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewRouter returns a Gin engine in release mode with recovery + a /healthz
// endpoint returning {"status":"ok","service":service}.
func NewRouter(service string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": service})
	})
	return r
}

// Error writes a JSON error body {"error":..,"detail":..} matching the OpenAPI
// Error schema.
func Error(c *gin.Context, status int, err string, detail string) {
	body := gin.H{"error": err}
	if detail != "" {
		body["detail"] = detail
	}
	c.JSON(status, body)
}

// Env returns the value of key, or def if unset/empty.
func Env(getenv func(string) string, key, def string) string {
	if v := getenv(key); v != "" {
		return v
	}
	return def
}
