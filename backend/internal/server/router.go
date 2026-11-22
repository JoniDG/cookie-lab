// Package server builds the HTTP router for the cookie lab.
package server

import (
	"crypto/rand"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const (
	cookieName   = "session"
	cookieMaxAge = 3600
)

// NewRouter wires the API and, from static, the SPA build. static is the root
// of the frontend build (it should contain index.html).
func NewRouter(cfg Config, static fs.FS) (*gin.Engine, error) {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// ClientIP is not used and Render does not publish its proxy IP ranges, so
	// no proxy is trusted: X-Forwarded-For is ignored instead of spoofable.
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, err
	}

	if cfg.Mode == ModeCrossSite {
		r.Use(cors.New(cors.Config{
			AllowOrigins:     []string{cfg.AllowedOrigin},
			AllowMethods:     []string{http.MethodGet, http.MethodPost},
			AllowHeaders:     []string{"Content-Type"},
			AllowCredentials: true,
			MaxAge:           10 * time.Minute,
		}))
	}

	api := r.Group("/api")
	api.POST("/login", func(c *gin.Context) {
		value := rand.Text()
		http.SetCookie(c.Writer, newCookie(cfg.Mode, value, cookieMaxAge))
		c.JSON(http.StatusOK, gin.H{"set": true, "value": value})
	})
	api.GET("/me", func(c *gin.Context) {
		value, err := c.Cookie(cookieName)
		c.JSON(http.StatusOK, gin.H{
			"received":        err == nil,
			"value":           value,
			"rawCookieHeader": c.GetHeader("Cookie"),
		})
	})
	api.POST("/logout", func(c *gin.Context) {
		http.SetCookie(c.Writer, newCookie(cfg.Mode, "", -1))
		c.JSON(http.StatusOK, gin.H{"cleared": true})
	})

	index, err := fs.ReadFile(static, "index.html")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	httpFS := http.FS(static)
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		isGet := c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead
		if !isGet || p == "/api" || strings.HasPrefix(p, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if index == nil {
			c.String(http.StatusNotFound, "frontend not built: run `make web`")
			return
		}
		name := strings.TrimPrefix(path.Clean(p), "/")
		if info, err := fs.Stat(static, name); err == nil && !info.IsDir() && name != "index.html" {
			c.FileFromFS(name, httpFS)
			return
		}
		// Served from memory: FileFromFS("index.html") makes http.FileServer
		// redirect to "./", which would loop on SPA routes.
		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	})

	return r, nil
}

// newCookie builds the session cookie. Logout must send the same Path and
// SameSite (and no Domain, like login) or the browser keeps the old cookie.
// maxAge < 0 serializes as Max-Age=0, which deletes it.
func newCookie(mode CookieMode, value string, maxAge int) *http.Cookie {
	c := &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	if mode == ModeCrossSite {
		c.SameSite = http.SameSiteNoneMode
	}
	return c
}
