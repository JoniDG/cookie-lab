package server

import (
	"errors"
	"fmt"
	"net/url"
)

// CookieMode selects the cookie attributes and whether CORS is enabled.
type CookieMode string

// Supported cookie modes.
const (
	ModeSameSite  CookieMode = "same-site"
	ModeCrossSite CookieMode = "cross-site"
)

// Config holds the runtime configuration read from the environment.
type Config struct {
	Port          string
	Mode          CookieMode
	AllowedOrigin string
}

// LoadConfig builds a Config from an env lookup function (os.Getenv in main).
func LoadConfig(getenv func(string) string) (Config, error) {
	cfg := Config{
		Port:          getenv("PORT"),
		Mode:          CookieMode(getenv("COOKIE_MODE")),
		AllowedOrigin: getenv("ALLOWED_ORIGIN"),
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.Mode == "" {
		cfg.Mode = ModeSameSite
	}

	switch cfg.Mode {
	case ModeSameSite:
		if cfg.AllowedOrigin != "" {
			return Config{}, errors.New("ALLOWED_ORIGIN is only valid with COOKIE_MODE=cross-site")
		}
	case ModeCrossSite:
		if err := validateOrigin(cfg.AllowedOrigin); err != nil {
			return Config{}, err
		}
	default:
		return Config{}, fmt.Errorf("invalid COOKIE_MODE %q: want %q or %q", cfg.Mode, ModeSameSite, ModeCrossSite)
	}
	return cfg, nil
}

// validateOrigin requires an exact origin (scheme://host[:port]) because the
// browser compares it byte for byte against Access-Control-Allow-Origin.
func validateOrigin(origin string) error {
	if origin == "" {
		return errors.New("ALLOWED_ORIGIN is required with COOKIE_MODE=cross-site")
	}
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" ||
		u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return fmt.Errorf("invalid ALLOWED_ORIGIN %q: want scheme://host[:port] without path or trailing slash", origin)
	}
	return nil
}
