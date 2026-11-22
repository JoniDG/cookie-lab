package server

import (
	"strings"
	"testing"
)

func envMap(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr string
	}{
		{
			name: "defaults",
			env:  map[string]string{},
			want: Config{Port: "8080", Mode: ModeSameSite},
		},
		{
			name: "explicit same-site with port",
			env:  map[string]string{"PORT": "10000", "COOKIE_MODE": "same-site"},
			want: Config{Port: "10000", Mode: ModeSameSite},
		},
		{
			name: "cross-site with origin",
			env:  map[string]string{"COOKIE_MODE": "cross-site", "ALLOWED_ORIGIN": "https://front.example.com"},
			want: Config{Port: "8080", Mode: ModeCrossSite, AllowedOrigin: "https://front.example.com"},
		},
		{
			name: "cross-site with localhost port",
			env:  map[string]string{"COOKIE_MODE": "cross-site", "ALLOWED_ORIGIN": "http://localhost:5173"},
			want: Config{Port: "8080", Mode: ModeCrossSite, AllowedOrigin: "http://localhost:5173"},
		},
		{
			name:    "invalid mode",
			env:     map[string]string{"COOKIE_MODE": "lax"},
			wantErr: "invalid COOKIE_MODE",
		},
		{
			name:    "origin in same-site mode",
			env:     map[string]string{"ALLOWED_ORIGIN": "https://front.example.com"},
			wantErr: "only valid with COOKIE_MODE=cross-site",
		},
		{
			name:    "cross-site without origin",
			env:     map[string]string{"COOKIE_MODE": "cross-site"},
			wantErr: "ALLOWED_ORIGIN is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadConfig(envMap(tt.env))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLoadConfigRejectsNonExactOrigins(t *testing.T) {
	for _, origin := range []string{
		"*",
		"front.example.com",
		"ftp://front.example.com",
		"https://",
		"https://front.example.com/",
		"https://front.example.com/app",
		"https://front.example.com?x=1",
		"https://front.example.com#x",
		"https://user@front.example.com",
		"https://front example.com",
	} {
		t.Run(origin, func(t *testing.T) {
			_, err := LoadConfig(envMap(map[string]string{"COOKIE_MODE": "cross-site", "ALLOWED_ORIGIN": origin}))
			if err == nil || !strings.Contains(err.Error(), "invalid ALLOWED_ORIGIN") {
				t.Fatalf("err = %v, want invalid ALLOWED_ORIGIN", err)
			}
		})
	}
}
