package auth

import (
	"context"
	"fmt"
	"regexp"

	"golang.org/x/crypto/bcrypt"

	"github.com/DonovanDiamond/signalmgr/gateway/src/config"
	"github.com/DonovanDiamond/signalmgr/gateway/src/db"
)

// maxPasswordBytes is bcrypt's input limit; longer passwords would be
// silently truncated or refused.
const maxPasswordBytes = 72

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9._@-]{1,64}$`)

// UserInfo is the administrator's view of an account.
type UserInfo struct {
	User
	Disabled bool `json:"disabled"`
}

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, fmt.Sprintf(format, a...))
}

func checkNewPassword(p string) error {
	if len(p) < config.MinPasswordLength {
		return invalid("the password must be at least %d characters", config.MinPasswordLength)
	}
	if len(p) > maxPasswordBytes {
		return invalid("the password must be at most %d bytes", maxPasswordBytes)
	}
	return nil
}

func (s *Service) audit(ctx context.Context, actor, action, detail string, err error) {
	result := "ok"
	if err != nil {
		result = err.Error()
	}
	if aerr := db.InsertAudit(ctx, s.x, db.AuditEntry{Actor: actor, Action: action, Detail: detail, Result: result}); aerr != nil {
		s.log.Error().Err(aerr).Str("actor", actor).Str("action", action).Msg("writing audit row")
	}
}

// ListUsers returns every account.
func (s *Service) ListUsers(ctx context.Context) ([]UserInfo, error) {
	us, err := db.ListUsers(ctx, s.x)
	if err != nil {
		return nil, err
	}
	out := make([]UserInfo, len(us))
	for i, u := range us {
		out[i] = UserInfo{User: toUser(u), Disabled: u.Disabled}
	}
	return out, nil
}

// CreateUser adds a local account; db.ErrDuplicate if the username is taken.
func (s *Service) CreateUser(ctx context.Context, actor, username, name, role, password string) (err error) {
	defer func() { s.audit(ctx, actor, "create_user", fmt.Sprintf("user=%s role=%s", username, role), err) }()
	switch {
	case !usernamePattern.MatchString(username):
		return invalid("the username must be 1-64 characters from letters, digits and . _ @ -")
	case name == "" || len(name) > 100:
		return invalid("the name must be 1-100 characters")
	case role != RoleAdmin && role != RoleUser:
		return invalid("the role must be %q or %q", RoleAdmin, RoleUser)
	}
	if err := checkNewPassword(password); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth: hashing password: %w", err)
	}
	return db.InsertLocalUser(ctx, s.x, username, name, role, string(hash))
}

// SetDisabled disables or re-enables an account and, when disabling, ends its
// sessions. Directory accounts are governed by the directory, and an
// administrator cannot lock themselves out.
func (s *Service) SetDisabled(ctx context.Context, actor, username string, disabled bool) (err error) {
	defer func() {
		s.audit(ctx, actor, "set_user_disabled", fmt.Sprintf("user=%s disabled=%t", username, disabled), err)
	}()
	if username == actor {
		return invalid("you cannot disable your own account")
	}
	u, err := db.GetUserByUsername(ctx, s.x, username)
	if err != nil {
		return err
	}
	if u.Source != db.SourceLocal {
		return invalid("Active Directory accounts are disabled in the directory, not here")
	}
	if err := db.SetUserDisabled(ctx, s.x, username, disabled); err != nil {
		return err
	}
	if disabled {
		return db.RevokeUserSessions(ctx, s.x, u.ID, s.now())
	}
	return nil
}

// SetPassword is an administrator's reset of a local account; it ends that
// user's sessions.
func (s *Service) SetPassword(ctx context.Context, actor, username, password string) (err error) {
	defer func() { s.audit(ctx, actor, "reset_user_password", "user="+username, err) }()
	u, err := db.GetUserByUsername(ctx, s.x, username)
	if err != nil {
		return err
	}
	if u.Source != db.SourceLocal {
		return invalid("Active Directory accounts have their password changed in the directory")
	}
	if err := checkNewPassword(password); err != nil {
		return err
	}
	if err := s.storePassword(ctx, username, password); err != nil {
		return err
	}
	return db.RevokeUserSessions(ctx, s.x, u.ID, s.now())
}

// ChangePassword lets a local user replace their own password. A wrong
// current password is ErrInvalidInput, not ErrInvalidCredentials: the person
// is signed in, and a 401 would make the client discard the session.
func (s *Service) ChangePassword(ctx context.Context, username, current, password string) (err error) {
	defer func() { s.audit(ctx, username, "change_password", "", err) }()
	u, err := db.GetUserByUsername(ctx, s.x, username)
	if err != nil {
		return err
	}
	if u.Source != db.SourceLocal {
		return invalid("Active Directory accounts have their password changed in the directory")
	}
	if !s.passwordMatches(u.PasswordHash, current) {
		return invalid("the current password is wrong")
	}
	if err := checkNewPassword(password); err != nil {
		return err
	}
	return s.storePassword(ctx, username, password)
}

func (s *Service) storePassword(ctx context.Context, username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth: hashing password: %w", err)
	}
	return db.SetUserPassword(ctx, s.x, username, string(hash))
}
