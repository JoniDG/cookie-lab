package server

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

const frontOrigin = "https://front.example.com"

var (
	sameSiteCfg  = Config{Port: "8080", Mode: ModeSameSite}
	crossSiteCfg = Config{Port: "8080", Mode: ModeCrossSite, AllowedOrigin: frontOrigin}
	spaFS        = fstest.MapFS{
		"index.html":       {Data: []byte("<!doctype html><title>spa</title>")},
		"assets/app.js":    {Data: []byte("console.log('app')")},
		"assets/style.css": {Data: []byte("body{}")},
	}
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

func newTestRouter(t *testing.T, cfg Config, static fs.FS) *gin.Engine {
	t.Helper()
	r, err := NewRouter(cfg, static)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	return r
}

func do(r http.Handler, method, target string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", w.Body.String(), err)
	}
	return body
}

func sessionCookie(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	headers := w.Result().Header.Values("Set-Cookie")
	if len(headers) != 1 {
		t.Fatalf("Set-Cookie headers = %q, want exactly one", headers)
	}
	c, err := http.ParseSetCookie(headers[0])
	if err != nil {
		t.Fatalf("parse Set-Cookie %q: %v", headers[0], err)
	}
	if c.Name != cookieName {
		t.Fatalf("cookie name = %q, want %q", c.Name, cookieName)
	}
	return c
}

func TestLoginSetsCookieAttributes(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		headers  map[string]string
		sameSite http.SameSite
		raw      string
	}{
		{"same-site", sameSiteCfg, nil, http.SameSiteLaxMode, "SameSite=Lax"},
		{"cross-site", crossSiteCfg, map[string]string{"Origin": frontOrigin}, http.SameSiteNoneMode, "SameSite=None"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := do(newTestRouter(t, tt.cfg, spaFS), http.MethodPost, "/api/login", tt.headers)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}
			c := sessionCookie(t, w)
			if c.Value == "" {
				t.Fatal("empty cookie value")
			}
			if !c.HttpOnly || !c.Secure || c.Path != "/" || c.Domain != "" || c.MaxAge != cookieMaxAge || c.SameSite != tt.sameSite {
				t.Fatalf("unexpected attributes: %+v", c)
			}
			if raw := w.Header().Get("Set-Cookie"); !strings.Contains(raw, tt.raw) {
				t.Fatalf("Set-Cookie %q does not contain %q", raw, tt.raw)
			}
			body := decode(t, w)
			if body["set"] != true || body["value"] != c.Value {
				t.Fatalf("body = %v, want set=true and value=%q", body, c.Value)
			}
		})
	}
}

func TestLoginGeneratesDistinctValues(t *testing.T) {
	r := newTestRouter(t, sameSiteCfg, spaFS)
	a := sessionCookie(t, do(r, http.MethodPost, "/api/login", nil)).Value
	b := sessionCookie(t, do(r, http.MethodPost, "/api/login", nil)).Value
	if a == b {
		t.Fatalf("two logins produced the same value %q", a)
	}
}

func TestMe(t *testing.T) {
	r := newTestRouter(t, sameSiteCfg, spaFS)

	t.Run("with cookie", func(t *testing.T) {
		raw := "other=1; session=abc123"
		body := decode(t, do(r, http.MethodGet, "/api/me", map[string]string{"Cookie": raw}))
		if body["received"] != true || body["value"] != "abc123" || body["rawCookieHeader"] != raw {
			t.Fatalf("body = %v", body)
		}
	})
	t.Run("without cookie", func(t *testing.T) {
		body := decode(t, do(r, http.MethodGet, "/api/me", nil))
		if body["received"] != false || body["value"] != "" || body["rawCookieHeader"] != "" {
			t.Fatalf("body = %v", body)
		}
	})
	t.Run("other cookies only", func(t *testing.T) {
		body := decode(t, do(r, http.MethodGet, "/api/me", map[string]string{"Cookie": "other=1"}))
		if body["received"] != false || body["rawCookieHeader"] != "other=1" {
			t.Fatalf("body = %v", body)
		}
	})
}

func TestLogoutClearsCookieWithSameAttributes(t *testing.T) {
	for _, tt := range []struct {
		cfg      Config
		headers  map[string]string
		sameSite http.SameSite
	}{
		{sameSiteCfg, nil, http.SameSiteLaxMode},
		{crossSiteCfg, map[string]string{"Origin": frontOrigin}, http.SameSiteNoneMode},
	} {
		t.Run(string(tt.cfg.Mode), func(t *testing.T) {
			w := do(newTestRouter(t, tt.cfg, spaFS), http.MethodPost, "/api/logout", tt.headers)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}
			if raw := w.Header().Get("Set-Cookie"); !strings.Contains(raw, "Max-Age=0") {
				t.Fatalf("Set-Cookie %q does not contain Max-Age=0", raw)
			}
			c := sessionCookie(t, w)
			if c.Value != "" || c.MaxAge != -1 || c.Path != "/" || c.Domain != "" || !c.Secure || !c.HttpOnly || c.SameSite != tt.sameSite {
				t.Fatalf("unexpected attributes: %+v", c)
			}
			if body := decode(t, w); body["cleared"] != true {
				t.Fatalf("body = %v", body)
			}
		})
	}
}

func TestCORS(t *testing.T) {
	r := newTestRouter(t, crossSiteCfg, spaFS)

	t.Run("allowed origin", func(t *testing.T) {
		w := do(r, http.MethodGet, "/api/me", map[string]string{"Origin": frontOrigin})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != frontOrigin {
			t.Fatalf("Allow-Origin = %q, want %q", got, frontOrigin)
		}
		if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
			t.Fatalf("Allow-Credentials = %q, want true", got)
		}
	})
	t.Run("disallowed origin", func(t *testing.T) {
		w := do(r, http.MethodPost, "/api/login", map[string]string{"Origin": "https://evil.example.com"})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("Allow-Origin = %q, want empty", got)
		}
		if got := w.Header().Get("Set-Cookie"); got != "" {
			t.Fatalf("Set-Cookie = %q, want none for a rejected origin", got)
		}
	})
	t.Run("preflight allowed", func(t *testing.T) {
		w := do(r, http.MethodOptions, "/api/login", map[string]string{
			"Origin":                         frontOrigin,
			"Access-Control-Request-Method":  http.MethodPost,
			"Access-Control-Request-Headers": "content-type",
		})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
		h := w.Header()
		if h.Get("Access-Control-Allow-Origin") != frontOrigin || h.Get("Access-Control-Allow-Credentials") != "true" {
			t.Fatalf("preflight headers = %v", h)
		}
		if !strings.Contains(h.Get("Access-Control-Allow-Methods"), http.MethodPost) {
			t.Fatalf("Allow-Methods = %q, want POST", h.Get("Access-Control-Allow-Methods"))
		}
		if !strings.Contains(strings.ToLower(h.Get("Access-Control-Allow-Headers")), "content-type") {
			t.Fatalf("Allow-Headers = %q, want content-type", h.Get("Access-Control-Allow-Headers"))
		}
	})
	t.Run("preflight disallowed origin", func(t *testing.T) {
		w := do(r, http.MethodOptions, "/api/login", map[string]string{
			"Origin":                        "https://evil.example.com",
			"Access-Control-Request-Method": http.MethodPost,
		})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("Allow-Origin = %q, want empty", got)
		}
	})
	t.Run("same-site mode sends no CORS headers", func(t *testing.T) {
		w := do(newTestRouter(t, sameSiteCfg, spaFS), http.MethodGet, "/api/me", map[string]string{"Origin": frontOrigin})
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("Allow-Origin = %q, want empty", got)
		}
	})
}

func TestStaticAndSPAFallback(t *testing.T) {
	r := newTestRouter(t, sameSiteCfg, spaFS)

	for _, tt := range []struct {
		target, wantBody, wantType string
		wantCode                   int
	}{
		{"/", "<title>spa</title>", "text/html", http.StatusOK},
		{"/index.html", "<title>spa</title>", "text/html", http.StatusOK},
		{"/some/client/route", "<title>spa</title>", "text/html", http.StatusOK},
		{"/assets", "<title>spa</title>", "text/html", http.StatusOK},
		{"/assets/app.js", "console.log", "javascript", http.StatusOK},
		{"/assets/style.css", "body{}", "text/css", http.StatusOK},
		{"/apiary", "<title>spa</title>", "text/html", http.StatusOK},
		{"/api", `"error":"not found"`, "application/json", http.StatusNotFound},
		{"/api/unknown", `"error":"not found"`, "application/json", http.StatusNotFound},
		{"/api/login", `"error":"not found"`, "application/json", http.StatusNotFound}, // GET on a POST route
	} {
		t.Run(tt.target, func(t *testing.T) {
			w := do(r, http.MethodGet, tt.target, nil)
			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantCode)
			}
			if !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Fatalf("body = %q, want containing %q", w.Body.String(), tt.wantBody)
			}
			if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, tt.wantType) {
				t.Fatalf("Content-Type = %q, want containing %q", ct, tt.wantType)
			}
		})
	}
}

func TestSPAFallbackOnlyForGetAndHead(t *testing.T) {
	r := newTestRouter(t, sameSiteCfg, spaFS)

	if w := do(r, http.MethodHead, "/some/route", nil); w.Code != http.StatusOK {
		t.Fatalf("HEAD = %d, want 200", w.Code)
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		if w := do(r, method, "/some/route", nil); w.Code != http.StatusNotFound {
			t.Fatalf("%s = %d, want 404", method, w.Code)
		}
	}
}

func TestFrontendNotBuilt(t *testing.T) {
	r := newTestRouter(t, sameSiteCfg, fstest.MapFS{".gitkeep": {}})

	if w := do(r, http.MethodGet, "/", nil); w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "frontend not built") {
		t.Fatalf("GET / = %d %q", w.Code, w.Body.String())
	}
	if w := do(r, http.MethodPost, "/api/login", nil); w.Code != http.StatusOK {
		t.Fatalf("API should work without a frontend build, got %d", w.Code)
	}
}

type brokenFS struct{}

func (brokenFS) Open(string) (fs.File, error) { return nil, errors.New("disk on fire") }

func TestNewRouterFailsOnUnreadableStaticFS(t *testing.T) {
	if _, err := NewRouter(sameSiteCfg, brokenFS{}); err == nil || !strings.Contains(err.Error(), "disk on fire") {
		t.Fatalf("err = %v, want the read error", err)
	}
}
