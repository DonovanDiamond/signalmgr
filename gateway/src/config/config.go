// Package config loads and validates the gateway's config.yaml.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"
	_ "time/tzdata" // so Timezone resolves on Windows hosts without a zoneinfo database

	"gopkg.in/yaml.v3"
)

// Environment overrides for the first administrator, so the password need not
// sit in a file.
const (
	EnvAdminUsername = "SIGNALGW_ADMIN_USERNAME"
	EnvAdminPassword = "SIGNALGW_ADMIN_PASSWORD"
)

// MinPasswordLength is the shortest local password accepted anywhere.
const MinPasswordLength = 12

var validADSecurity = map[string]bool{"none": true, "tls": true, "starttls": true}

// Config is the whole of config.yaml.
type Config struct {
	InstallationName string `yaml:"installation_name"`
	Listen           string `yaml:"listen"`
	Timezone         string `yaml:"timezone"`
	DB               string `yaml:"db"`
	SignalAPIURL     string `yaml:"signal_api_url"`

	JWT        JWTConfig        `yaml:"jwt"`
	AD         ADConfig         `yaml:"ad"`
	FirstAdmin FirstAdminConfig `yaml:"first_admin"`
}

// JWTConfig holds the signing key pair paths and the session lifetime.
// The pair is generated at first run when both files are absent.
type JWTConfig struct {
	PrivateKeyPath string        `yaml:"private_key_path"`
	PublicKeyPath  string        `yaml:"public_key_path"`
	TokenTTL       time.Duration `yaml:"token_ttl"`
}

// ADConfig is optional; nothing else in it is read unless Enabled.
// There is no default role: someone in none of GroupMap's groups cannot sign in.
type ADConfig struct {
	Enabled      bool              `yaml:"enabled"`
	Server       string            `yaml:"server"`
	Port         int               `yaml:"port"`
	BaseDN       string            `yaml:"base_dn"`
	Security     string            `yaml:"security"` // none, tls or starttls
	BindDN       string            `yaml:"bind_dn"`
	BindPassword string            `yaml:"bind_password"`
	GroupMap     map[string]string `yaml:"group_map"` // AD group -> admin|user
}

// FirstAdminConfig seeds the first local administrator on an empty database.
// An empty Password means a one-time password is generated and logged.
type FirstAdminConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func defaults() Config {
	return Config{
		InstallationName: "Signal Gateway",
		Listen:           ":8080",
		Timezone:         "UTC",
		DB:               "data/gateway.db",
		SignalAPIURL:     "http://127.0.0.1:8081",
		JWT: JWTConfig{
			PrivateKeyPath: "data/jwt_private.pem",
			PublicKeyPath:  "data/jwt_public.pem",
			TokenTTL:       8 * time.Hour,
		},
		FirstAdmin: FirstAdminConfig{Username: "admin"},
	}
}

// Load reads, applies defaults and environment overrides to, and validates
// the file at path. Unknown keys are refused so a typo cannot silently fall
// back to a default.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: reading %q: %w", path, err)
	}

	cfg := defaults()
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("config: parsing %q as YAML: %w", path, err)
	}

	if v := os.Getenv(EnvAdminUsername); v != "" {
		cfg.FirstAdmin.Username = v
	}
	if v := os.Getenv(EnvAdminPassword); v != "" {
		cfg.FirstAdmin.Password = v
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Location is the configured timezone; validate guarantees it loads.
func (c *Config) Location() *time.Location {
	loc, _ := time.LoadLocation(c.Timezone)
	return loc
}

// validate never echoes secret values.
func (c *Config) validate() error {
	if c.InstallationName == "" {
		return fmt.Errorf("config: installation_name must not be empty, expected a string")
	}
	if c.Listen == "" {
		return fmt.Errorf("config: listen must not be empty, expected an address such as \":8080\"")
	}
	if _, err := time.LoadLocation(c.Timezone); err != nil {
		return fmt.Errorf("config: timezone %q is not an IANA zone name (e.g. \"Africa/Johannesburg\"): %w", c.Timezone, err)
	}
	if c.DB == "" {
		return fmt.Errorf("config: db must not be empty, expected a file path")
	}
	u, err := url.Parse(c.SignalAPIURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("config: signal_api_url %q must be an http(s) URL such as \"http://127.0.0.1:8081\"", c.SignalAPIURL)
	}
	if c.JWT.PrivateKeyPath == "" || c.JWT.PublicKeyPath == "" {
		return fmt.Errorf("config: jwt.private_key_path and jwt.public_key_path must not be empty, expected file paths")
	}
	if c.JWT.TokenTTL <= 0 {
		return fmt.Errorf("config: jwt.token_ttl must be greater than 0, got %s", c.JWT.TokenTTL)
	}
	if c.FirstAdmin.Username == "" {
		return fmt.Errorf("config: first_admin.username must not be empty")
	}
	if p := c.FirstAdmin.Password; p != "" && len(p) < MinPasswordLength {
		return fmt.Errorf("config: first_admin password must be at least %d characters, got %d", MinPasswordLength, len(p))
	}
	return c.AD.validate()
}

func (a *ADConfig) validate() error {
	if !a.Enabled {
		return nil
	}
	if a.Server == "" {
		return fmt.Errorf("config: ad.server must not be empty when ad.enabled is true")
	}
	if a.Port < 1 || a.Port > 65535 {
		return fmt.Errorf("config: ad.port must be between 1 and 65535, got %d", a.Port)
	}
	if a.BaseDN == "" {
		return fmt.Errorf("config: ad.base_dn must not be empty when ad.enabled is true")
	}
	if !validADSecurity[a.Security] {
		return fmt.Errorf("config: ad.security %q is invalid, must be one of none, tls, starttls", a.Security)
	}
	if len(a.GroupMap) == 0 {
		return fmt.Errorf("config: ad.group_map must not be empty when ad.enabled is true, expected at least one group mapped to admin or user")
	}
	for group, role := range a.GroupMap {
		if role != "admin" && role != "user" {
			return fmt.Errorf("config: ad.group_map[%q] role %q is invalid, must be admin or user", group, role)
		}
	}
	// A bind_dn without a password would silently make the directory reads anonymous.
	if a.BindDN != "" && a.BindPassword == "" {
		return fmt.Errorf("config: ad.bind_password must not be empty when ad.bind_dn is set")
	}
	return nil
}
