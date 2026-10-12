package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

const adOK = `
ad:
  enabled: true
  server: dc.example.org
  port: 636
  base_dn: DC=example,DC=org
  security: tls
  group_map: {Admins: admin}
`

func TestLoad(t *testing.T) {
	t.Setenv(EnvAdminUsername, "")
	t.Setenv(EnvAdminPassword, "")
	tests := []struct {
		name    string
		yaml    string
		wantErr string // substring; empty means success
	}{
		{"empty file takes defaults", "", ""},
		{"full AD block", adOK, ""},
		{"bad timezone", "timezone: Mars/Base", "timezone"},
		{"bad signal url", "signal_api_url: 127.0.0.1:8081", "signal_api_url"},
		{"non-positive ttl", "jwt: {token_ttl: 0s}", "token_ttl"},
		{"short admin password", "first_admin: {password: short}", "at least 12"},
		{"unknown key refused", "listn: :80", "listn"},
		{"empty name", "installation_name: ''", "installation_name"},
		{"ad missing server", "ad: {enabled: true, port: 636, base_dn: DC=a, security: tls, group_map: {G: user}}", "ad.server"},
		{"ad bad security", strings.Replace(adOK, "security: tls", "security: ssl", 1), "ad.security"},
		{"ad bad role", strings.Replace(adOK, "Admins: admin", "Admins: root", 1), "role"},
		{"ad empty group map", strings.Replace(adOK, "group_map: {Admins: admin}", "group_map: {}", 1), "group_map"},
		{"ad bind dn without password", strings.Replace(adOK, "security: tls", "security: tls\n  bind_dn: CN=svc", 1), "bind_password"},
		{"ad disabled ignores rest", "ad: {enabled: false, port: 0}", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := Load(write(t, tc.yaml))
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if cfg.Listen == "" || cfg.SignalAPIURL == "" {
					t.Fatalf("defaults not applied: %+v", cfg)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestLoadDefaultsAndOverrides(t *testing.T) {
	t.Setenv(EnvAdminUsername, "root")
	t.Setenv(EnvAdminPassword, "a-long-enough-password")
	cfg, err := Load(write(t, "timezone: Africa/Johannesburg"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":8080" || cfg.SignalAPIURL != "http://127.0.0.1:8081" || cfg.JWT.TokenTTL != 8*time.Hour {
		t.Fatalf("defaults wrong: %+v", cfg)
	}
	if cfg.FirstAdmin.Username != "root" || cfg.FirstAdmin.Password != "a-long-enough-password" {
		t.Fatalf("env override not applied: %+v", cfg.FirstAdmin)
	}
	if cfg.Location().String() != "Africa/Johannesburg" {
		t.Fatalf("location = %v", cfg.Location())
	}
}

func TestLoadEnvPasswordValidated(t *testing.T) {
	t.Setenv(EnvAdminPassword, "short")
	if _, err := Load(write(t, "")); err == nil {
		t.Fatal("short env password accepted")
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("missing file accepted")
	}
}
