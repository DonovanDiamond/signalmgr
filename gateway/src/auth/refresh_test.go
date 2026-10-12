package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DonovanDiamond/signalmgr/gateway/src/config"
	"github.com/DonovanDiamond/signalmgr/gateway/src/db"
)

func TestRefresh(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		prepare func(t *testing.T, s *Service, first Tokens) string // returns the refresh token to present
		wantErr error
	}{
		{"valid", func(t *testing.T, s *Service, first Tokens) string { return first.RefreshToken }, nil},
		{"unknown", func(*testing.T, *Service, Tokens) string { return "nope" }, ErrInvalidToken},
		{"empty", func(*testing.T, *Service, Tokens) string { return "" }, ErrInvalidToken},
		{"access token is not a refresh token", func(t *testing.T, s *Service, first Tokens) string { return first.Token }, ErrInvalidToken},
		{"already rotated", func(t *testing.T, s *Service, first Tokens) string {
			if _, _, err := s.Refresh(ctx, first.RefreshToken); err != nil {
				t.Fatal(err)
			}
			return first.RefreshToken
		}, ErrInvalidToken},
		{"expired", func(t *testing.T, s *Service, first Tokens) string {
			s.now = func() time.Time { return time.Now().Add(9 * time.Hour) }
			return first.RefreshToken
		}, ErrInvalidToken},
		{"revoked by logout", func(t *testing.T, s *Service, first Tokens) string {
			if err := s.Logout(ctx, first.Token); err != nil {
				t.Fatal(err)
			}
			return first.RefreshToken
		}, ErrInvalidToken},
		{"user disabled", func(t *testing.T, s *Service, first Tokens) string {
			if err := s.CreateUser(ctx, "admin", "bob", "Bob", RoleUser, "bob-password-123"); err != nil {
				t.Fatal(err)
			}
			bob, _, err := s.Login(ctx, "bob", "bob-password-123")
			if err != nil {
				t.Fatal(err)
			}
			if err := s.SetDisabled(ctx, "admin", "bob", true); err != nil {
				t.Fatal(err)
			}
			return bob.RefreshToken
		}, ErrInvalidToken},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newService(t, nil)
			first, _, err := s.Login(ctx, "admin", adminPW)
			if err != nil {
				t.Fatal(err)
			}
			presented := tc.prepare(t, s, first)
			next, u, err := s.Refresh(ctx, presented)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if next.Token == first.Token || next.RefreshToken == first.RefreshToken || u.Username != "admin" {
				t.Fatalf("not rotated: %+v %+v", next, u)
			}
			if _, err := s.Verify(ctx, next.Token); err != nil {
				t.Fatalf("new access token rejected: %v", err)
			}
			if _, err := s.Verify(ctx, first.Token); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("old access token still valid: %v", err)
			}
			if _, _, err := s.Refresh(ctx, first.RefreshToken); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("old refresh token still valid: %v", err)
			}
		})
	}
}

func TestDisabledUser(t *testing.T) {
	ctx := context.Background()
	s, _ := newService(t, nil)
	if err := s.CreateUser(ctx, "admin", "bob", "Bob", RoleUser, "bob-password-123"); err != nil {
		t.Fatal(err)
	}
	tok, _, err := s.Login(ctx, "bob", "bob-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetDisabled(ctx, "admin", "bob", true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Verify(ctx, tok.Token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("disabling must revoke sessions, verify err = %v", err)
	}
	if _, _, err := s.Login(ctx, "bob", "bob-password-123"); !errors.Is(err, ErrDisabled) {
		t.Errorf("disabled login err = %v", err)
	}
	if _, _, err := s.Login(ctx, "bob", "wrong-password-xx"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("wrong password must not reveal the account is disabled, got %v", err)
	}
	if err := s.SetDisabled(ctx, "admin", "bob", false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Login(ctx, "bob", "bob-password-123"); err != nil {
		t.Errorf("re-enabled login: %v", err)
	}
}

func TestSetDisabledRules(t *testing.T) {
	ctx := context.Background()
	s, _ := newService(t, func(c *config.Config) { c.AD.Enabled = true })
	s.dir = stubDir{user: directoryUser{Name: "Jo", Role: RoleUser}}
	if _, _, err := s.Login(ctx, "jo", "pw"); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, target string
		wantErr      error
	}{
		{"self", "admin", ErrInvalidInput},
		{"AD user", "jo", ErrInvalidInput},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := s.SetDisabled(ctx, "admin", tc.target, true); !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestChangeAndResetPassword(t *testing.T) {
	ctx := context.Background()
	s, _ := newService(t, func(c *config.Config) { c.AD.Enabled = true })
	s.dir = stubDir{user: directoryUser{Name: "Jo", Role: RoleUser}}
	if _, _, err := s.Login(ctx, "jo", "pw"); err != nil {
		t.Fatal(err)
	}
	const fresh = "a-brand-new-password"
	tests := []struct {
		name, user, current, next string
		wantErr                   error
	}{
		{"wrong current", "admin", "not-the-password", fresh, ErrInvalidInput},
		{"too short", "admin", adminPW, "short", ErrInvalidInput},
		{"too long for bcrypt", "admin", adminPW, string(make([]byte, 73)), ErrInvalidInput},
		{"AD user", "jo", "pw", fresh, ErrInvalidInput},
		{"ok", "admin", adminPW, fresh, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := s.ChangePassword(ctx, tc.user, tc.current, tc.next)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
	if _, _, err := s.Login(ctx, "admin", fresh); err != nil {
		t.Errorf("new password rejected: %v", err)
	}
	if _, _, err := s.Login(ctx, "admin", adminPW); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("old password still works: %v", err)
	}

	tok, _, _ := s.Login(ctx, "admin", fresh)
	if err := s.SetPassword(ctx, "other", "jo", fresh); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("resetting an AD password: %v", err)
	}
	if err := s.SetPassword(ctx, "other", "admin", "short"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("resetting to a short password: %v", err)
	}
	if err := s.SetPassword(ctx, "other", "admin", "reset-by-someone-else"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Verify(ctx, tok.Token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("reset must revoke sessions, verify err = %v", err)
	}
	if _, _, err := s.Login(ctx, "admin", "reset-by-someone-else"); err != nil {
		t.Errorf("reset password rejected: %v", err)
	}
}

func TestCreateUserValidation(t *testing.T) {
	ctx := context.Background()
	s, _ := newService(t, nil)
	const pw = "long-enough-password"
	tests := []struct {
		name, username, display, role, password string
		wantErr                                 error
	}{
		{"ok", "carol", "Carol", RoleUser, pw, nil},
		{"duplicate", "admin", "Again", RoleUser, pw, db.ErrDuplicate},
		{"bad username", "has space", "X", RoleUser, pw, ErrInvalidInput},
		{"empty username", "", "X", RoleUser, pw, ErrInvalidInput},
		{"empty name", "dave", "", RoleUser, pw, ErrInvalidInput},
		{"bad role", "dave", "Dave", "root", pw, ErrInvalidInput},
		{"short password", "dave", "Dave", RoleUser, "short", ErrInvalidInput},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := s.CreateUser(ctx, "admin", tc.username, tc.display, tc.role, tc.password); !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
	us, err := s.ListUsers(ctx)
	if err != nil || len(us) != 2 || us[0].Username != "admin" || us[1].Username != "carol" || us[1].Source != "local" {
		t.Fatalf("list = %+v, %v", us, err)
	}
}

type slowDir struct{ release chan struct{} }

func (d slowDir) Authenticate(context.Context, string, string) (directoryUser, error) {
	<-d.release
	return directoryUser{}, nil
}

func TestDirectoryTimeout(t *testing.T) {
	ctx := context.Background()
	s, _ := newService(t, func(c *config.Config) { c.AD.Enabled = true })
	release := make(chan struct{})
	defer close(release)
	s.dir = slowDir{release}
	s.adTimeout = 50 * time.Millisecond

	_, _, err := s.Login(ctx, "jo", "pw")
	var te *DirectoryTimeoutError
	if !errors.As(err, &te) || !errors.Is(err, ErrDirectoryUnreachable) {
		t.Fatalf("err = %v, want a DirectoryTimeoutError", err)
	}
	if want := "Active Directory did not answer within 50ms"; err.Error() != want {
		t.Errorf("text = %q, want %q", err.Error(), want)
	}
	if _, _, err := s.Login(ctx, "admin", adminPW); err != nil {
		t.Errorf("local account affected by the directory: %v", err)
	}
}

func TestRefreshDeadlineIsAbsolute(t *testing.T) {
	ctx := context.Background()
	s, _ := newService(t, nil) // refresh_ttl is 8h
	start := time.Now()
	tokens, _, err := s.Login(ctx, "admin", adminPW)
	if err != nil {
		t.Fatal(err)
	}
	// Rotate every 3h: each refresh is well inside its own 8h, but the sign-in deadline is at 8h.
	for _, h := range []time.Duration{3, 6} {
		s.now = func() time.Time { return start.Add(h * time.Hour) }
		if tokens, _, err = s.Refresh(ctx, tokens.RefreshToken); err != nil {
			t.Fatalf("refresh at %dh: %v", h, err)
		}
	}
	s.now = func() time.Time { return start.Add(9 * time.Hour) }
	if _, _, err := s.Refresh(ctx, tokens.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("refresh after the sign-in deadline = %v, want ErrInvalidToken", err)
	}
}
