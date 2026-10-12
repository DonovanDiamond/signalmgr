package auth

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"

	"github.com/DonovanDiamond/signalmgr/gateway/src/config"
	"github.com/DonovanDiamond/signalmgr/gateway/src/db"
)

type stubDir struct {
	user directoryUser
	err  error
}

func (d stubDir) Authenticate(context.Context, string, string) (directoryUser, error) {
	return d.user, d.err
}

const adminPW = "correct-horse-battery"

func newService(t *testing.T, mutate func(*config.Config)) (*Service, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		JWT:        config.JWTConfig{PrivateKeyPath: filepath.Join(dir, "k", "priv.pem"), PublicKeyPath: filepath.Join(dir, "k", "pub.pem"), AccessTTL: time.Hour, RefreshTTL: 8 * time.Hour},
		AD:         config.ADConfig{Timeout: time.Second},
		FirstAdmin: config.FirstAdminConfig{Username: "admin", Password: adminPW},
	}
	if mutate != nil {
		mutate(cfg)
	}
	x, err := db.Open(filepath.Join(dir, "gw.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { x.Close() })
	var logs bytes.Buffer
	s, err := New(cfg, x, zerolog.New(&logs))
	if err != nil {
		t.Fatal(err)
	}
	return s, &logs
}

func TestLocalLogin(t *testing.T) {
	s, _ := newService(t, nil)
	ctx := context.Background()
	tests := []struct {
		name, user, pass string
		wantErr          error
	}{
		{"ok", "admin", adminPW, nil},
		{"wrong password", "admin", "nope", ErrInvalidCredentials},
		{"unknown user", "ghost", adminPW, ErrInvalidCredentials},
		{"empty password", "admin", "", ErrInvalidCredentials},
		{"empty username", "", adminPW, ErrInvalidCredentials},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tok, u, err := s.Login(ctx, tc.user, tc.pass)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if err == nil && (tok.Token == "" || tok.RefreshToken == "" || u != (User{Username: "admin", Name: "Administrator", Role: RoleAdmin, Source: "local"})) {
				t.Fatalf("token %q user %+v", tok, u)
			}
			if err != nil && tok != (Tokens{}) {
				t.Fatal("token issued on failure")
			}
		})
	}
}

func TestVerify(t *testing.T) {
	ctx := context.Background()
	s, _ := newService(t, nil)
	goodTokens, _, err := s.Login(ctx, "admin", adminPW)
	if err != nil {
		t.Fatal(err)
	}
	good := goodTokens.Token

	forged := func(method jwt.SigningMethod, key any, id string) string {
		now := time.Now()
		tok, err := jwt.NewWithClaims(method, jwt.RegisteredClaims{
			Subject: "admin", Issuer: issuer, ID: id, ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		}).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	_, otherKey, _ := ed25519.GenerateKey(nil)
	parts := strings.Split(good, ".")
	tampered := parts[0] + "." + parts[1] + "x." + parts[2]

	revokedTokens, _, _ := s.Login(ctx, "admin", adminPW)
	revoked := revokedTokens.Token
	if err := s.Logout(ctx, revoked); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		token string
		clock time.Duration // how far to move the clock before verifying
		ok    bool
	}{
		{"valid", good, 0, true},
		{"expired", good, 2 * time.Hour, false},
		{"tampered payload", tampered, 0, false},
		{"signed by another key", forged(jwt.SigningMethodEdDSA, otherKey, "x"), 0, false},
		{"alg none", forged(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, "x"), 0, false},
		{"signed but no session", forged(jwt.SigningMethodEdDSA, s.keys.Private, "unknown-session"), 0, false},
		{"revoked", revoked, 0, false},
		{"garbage", "not.a.jwt", 0, false},
		{"empty", "", 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s.now = func() time.Time { return time.Now().Add(tc.clock) }
			u, err := s.Verify(ctx, tc.token)
			if tc.ok {
				if err != nil || u.Username != "admin" || u.Role != RoleAdmin {
					t.Fatalf("user %+v err %v", u, err)
				}
				return
			}
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("err = %v, want ErrInvalidToken", err)
			}
		})
	}
}

func TestLogoutRejectsBadToken(t *testing.T) {
	s, _ := newService(t, nil)
	if err := s.Logout(context.Background(), "junk"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v", err)
	}
}

func TestDirectoryLogin(t *testing.T) {
	ctx := context.Background()
	adCfg := func(c *config.Config) { c.AD.Enabled = true }
	tests := []struct {
		name    string
		dir     stubDir
		user    string
		pass    string
		wantErr error
		role    string
	}{
		{"unreachable", stubDir{err: ErrDirectoryUnreachable}, "jo", "pw", ErrDirectoryUnreachable, ""},
		{"refused bind", stubDir{err: ErrInvalidCredentials}, "jo", "pw", ErrInvalidCredentials, ""},
		{"no mapped group", stubDir{err: ErrNoRole}, "jo", "pw", ErrNoRole, ""},
		{"ok user", stubDir{user: directoryUser{Name: "Jo B", Role: RoleUser}}, "jo", "pw", nil, RoleUser},
		{"unreachable never blocks local admin", stubDir{err: ErrDirectoryUnreachable}, "admin", adminPW, nil, RoleAdmin},
		{"local wrong password is not an outage", stubDir{err: ErrDirectoryUnreachable}, "admin", "bad", ErrInvalidCredentials, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newService(t, adCfg)
			s.dir = tc.dir
			tok, u, err := s.Login(ctx, tc.user, tc.pass)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if u.Role != tc.role {
				t.Fatalf("role = %q, want %q", u.Role, tc.role)
			}
			if v, err := s.Verify(ctx, tok.Token); err != nil || v != u {
				t.Fatalf("verify: %+v, %v", v, err)
			}
		})
	}
}

func TestADCannotShadowLocalAccount(t *testing.T) {
	s, _ := newService(t, func(c *config.Config) { c.AD.Enabled = true })
	s.dir = stubDir{user: directoryUser{Name: "Evil", Role: RoleAdmin}}
	if _, _, err := s.Login(context.Background(), "admin", "directory-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestADDisabledUnknownUser(t *testing.T) {
	s, _ := newService(t, nil)
	if s.dir != nil {
		t.Fatal("directory configured while AD disabled")
	}
}

func TestFirstAdmin(t *testing.T) {
	t.Run("generated password is logged once and works", func(t *testing.T) {
		s, logs := newService(t, func(c *config.Config) { c.FirstAdmin.Password = "" })
		out := logs.String()
		i := strings.Index(out, `"password":"`)
		if i < 0 {
			t.Fatalf("one-time password not logged: %s", out)
		}
		pw := out[i+len(`"password":"`):]
		pw = pw[:strings.Index(pw, `"`)]
		if _, _, err := s.Login(context.Background(), "admin", pw); err != nil {
			t.Fatalf("generated password rejected: %v", err)
		}
	})
	t.Run("configured password is never logged", func(t *testing.T) {
		_, logs := newService(t, nil)
		if strings.Contains(logs.String(), adminPW) {
			t.Fatal("configured password leaked to the log")
		}
	})
	t.Run("existing users are left alone", func(t *testing.T) {
		s, _ := newService(t, nil)
		before, _ := db.CountUsers(context.Background(), s.x)
		if err := s.ensureFirstAdmin(context.Background(), config.FirstAdminConfig{Username: "second", Password: adminPW}); err != nil {
			t.Fatal(err)
		}
		if after, _ := db.CountUsers(context.Background(), s.x); after != before {
			t.Fatalf("users %d -> %d", before, after)
		}
	})
}

func TestKeys(t *testing.T) {
	dir := t.TempDir()
	priv, pub := filepath.Join(dir, "a", "priv.pem"), filepath.Join(dir, "a", "pub.pem")
	k1, err := LoadOrGenerateKeys(priv, pub)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := LoadOrGenerateKeys(priv, pub)
	if err != nil || !bytes.Equal(k1.Private, k2.Private) {
		t.Fatalf("keys not reloaded: %v", err)
	}
	if err := os.Remove(pub); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrGenerateKeys(priv, pub); err == nil {
		t.Fatal("half a key pair was silently regenerated")
	}
}
