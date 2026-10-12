// Package auth signs users in (local accounts and optional Active Directory)
// and issues and verifies the JWTs that carry the session. It knows nothing
// about HTTP; the api package wraps it in middleware.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/DonovanDiamond/signalmgr/gateway/src/config"
	"github.com/DonovanDiamond/signalmgr/gateway/src/db"
)

const (
	RoleAdmin = "admin"
	RoleUser  = "user"

	issuer = "signal-gateway"
)

var (
	// ErrInvalidCredentials covers a wrong password and an unknown user alike.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrDirectoryUnreachable means Active Directory could not answer. It is
	// an outage, never a wrong password, and never blocks local accounts.
	ErrDirectoryUnreachable = errors.New("active directory unreachable")
	// ErrInvalidToken covers a bad signature, expiry, or a revoked session.
	ErrInvalidToken = errors.New("invalid token")
	// ErrDisabled means the password was right but an administrator has
	// disabled the account.
	ErrDisabled = errors.New("account is disabled")
	// ErrInvalidInput means a user-management request was rejected as sent;
	// the wrapping message says why and is safe to show.
	ErrInvalidInput = errors.New("invalid input")
	// ErrNoRole means the directory accepted the password but the person is
	// in none of the groups mapped to a role, so they may not sign in.
	ErrNoRole = errors.New("not a member of any group mapped to a role")
)

// DirectoryTimeoutError is ErrDirectoryUnreachable for a directory that did
// not answer in time; its text is shown to the person signing in.
type DirectoryTimeoutError struct{ After time.Duration }

func (e *DirectoryTimeoutError) Error() string {
	return fmt.Sprintf("Active Directory did not answer within %s", e.After)
}

func (e *DirectoryTimeoutError) Is(target error) bool { return target == ErrDirectoryUnreachable }

// User is the signed-in identity handed to the api layer.
type User struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Source   string `json:"source"` // local or ad
}

func toUser(u db.User) User {
	return User{Username: u.Username, Name: u.Name, Role: u.Role, Source: u.Source}
}

// Tokens is what a sign-in or refresh hands out: a short-lived signed access
// token and the opaque refresh token that renews it exactly once.
type Tokens struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

// Service issues and verifies sessions.
type Service struct {
	x          *sqlx.DB
	keys       *KeyPair
	accessTTL  time.Duration
	refreshTTL time.Duration
	adTimeout  time.Duration
	dir        directory // nil when AD is disabled
	log        zerolog.Logger
	now        func() time.Time
	dummy      []byte // compared against when no real hash exists, to keep timing even
}

// New loads or generates the signing key pair and makes sure a first
// administrator exists.
func New(cfg *config.Config, x *sqlx.DB, log zerolog.Logger) (*Service, error) {
	keys, err := LoadOrGenerateKeys(cfg.JWT.PrivateKeyPath, cfg.JWT.PublicKeyPath)
	if err != nil {
		return nil, err
	}
	dummy, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-constant-time"), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("auth: preparing timing hash: %w", err)
	}
	s := &Service{x: x, keys: keys, accessTTL: cfg.JWT.AccessTTL, refreshTTL: cfg.JWT.RefreshTTL, adTimeout: cfg.AD.Timeout, log: log, now: time.Now, dummy: dummy}
	if cfg.AD.Enabled {
		s.dir = newADDirectory(cfg.AD, log)
	}
	if err := s.ensureFirstAdmin(context.Background(), cfg.FirstAdmin); err != nil {
		return nil, err
	}
	return s, nil
}

// ensureFirstAdmin creates the administrator on an empty users table. With no
// configured password it generates one and logs it once: the operator has to
// be able to sign in, and there is no other channel on a first run.
func (s *Service) ensureFirstAdmin(ctx context.Context, fa config.FirstAdminConfig) error {
	n, err := db.CountUsers(ctx, s.x)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	password, generated := fa.Password, false
	if password == "" {
		b := make([]byte, 18)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("auth: generating first admin password: %w", err)
		}
		password, generated = base64.RawURLEncoding.EncodeToString(b), true
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth: hashing first admin password: %w", err)
	}
	if err := db.InsertLocalUser(ctx, s.x, fa.Username, "Administrator", RoleAdmin, string(hash)); err != nil {
		return fmt.Errorf("auth: creating first administrator %q: %w", fa.Username, err)
	}
	if generated {
		s.log.Warn().Str("username", fa.Username).Str("password", password).
			Msg("created the first administrator with a one-time password; sign in and change it, this is shown only once")
	} else {
		s.log.Warn().Str("username", fa.Username).Msg("created the first administrator from configuration")
	}
	return nil
}

// Login checks the credentials and returns signed tokens. A local account
// is judged by its own password only, so an AD outage cannot affect it.
// Any other username is tried against the directory when AD is enabled.
func (s *Service) Login(ctx context.Context, username, password string) (Tokens, User, error) {
	if username == "" || password == "" {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	stored, err := db.GetUserByUsername(ctx, s.x, username)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return Tokens{}, User{}, fmt.Errorf("auth: login of %q: %w", username, err)
	}

	var u db.User
	switch {
	case err == nil && stored.Source == db.SourceLocal:
		if !s.passwordMatches(stored.PasswordHash, password) {
			return Tokens{}, User{}, ErrInvalidCredentials
		}
		u = stored
	case s.dir != nil:
		du, err := s.authenticateDirectory(ctx, username, password)
		if err != nil {
			return Tokens{}, User{}, err
		}
		if u, err = db.UpsertADUser(ctx, s.x, username, du.Name, du.Role); err != nil {
			return Tokens{}, User{}, fmt.Errorf("auth: recording directory user %q: %w", username, err)
		}
	default:
		_ = bcrypt.CompareHashAndPassword(s.dummy, []byte(password))
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	if u.Disabled {
		return Tokens{}, User{}, ErrDisabled
	}

	tokens, err := s.issue(ctx, u, s.now().Add(s.refreshTTL))
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, toUser(u), nil
}

// authenticateDirectory bounds the directory call by ad.timeout. The LDAP
// client takes no deadline, so the call runs on its own goroutine and is
// abandoned (it ends when the OS gives up on the connection) if it overruns.
func (s *Service) authenticateDirectory(ctx context.Context, username, password string) (directoryUser, error) {
	cctx, cancel := context.WithTimeout(ctx, s.adTimeout)
	defer cancel()
	type result struct {
		u   directoryUser
		err error
	}
	ch := make(chan result, 1)
	go func() {
		u, err := s.dir.Authenticate(cctx, username, password)
		ch <- result{u, err}
	}()
	select {
	case r := <-ch:
		return r.u, r.err
	case <-cctx.Done():
		if err := ctx.Err(); err != nil {
			return directoryUser{}, err
		}
		s.log.Error().Str("username", username).Dur("timeout", s.adTimeout).Msg("active directory did not answer in time")
		return directoryUser{}, &DirectoryTimeoutError{After: s.adTimeout}
	}
}

// Refresh spends a refresh token: it and its session die and a fresh pair is
// issued. Unknown, spent, expired, revoked and disabled-user all look alike.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (Tokens, User, error) {
	if refreshToken == "" {
		return Tokens{}, User{}, ErrInvalidToken
	}
	userID, refreshExpires, err := db.ClaimRefresh(ctx, s.x, hashRefresh(refreshToken), s.now())
	if errors.Is(err, db.ErrNotFound) {
		return Tokens{}, User{}, ErrInvalidToken
	}
	if err != nil {
		return Tokens{}, User{}, fmt.Errorf("auth: refreshing session: %w", err)
	}
	u, err := db.GetUserByID(ctx, s.x, userID)
	if err != nil {
		return Tokens{}, User{}, fmt.Errorf("auth: refreshing session: %w", err)
	}
	if u.Disabled {
		return Tokens{}, User{}, ErrInvalidToken
	}
	// The deadline is carried forward, so the session ends refresh_ttl after sign-in however often it rotates.
	tokens, err := s.issue(ctx, u, refreshExpires)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, toUser(u), nil
}

func hashRefresh(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// passwordMatches always spends one bcrypt comparison, even without a hash.
func (s *Service) passwordMatches(hash *string, password string) bool {
	if hash == nil {
		_ = bcrypt.CompareHashAndPassword(s.dummy, []byte(password))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(*hash), []byte(password)) == nil
}

func (s *Service) issue(ctx context.Context, u db.User, refreshExpires time.Time) (Tokens, error) {
	now := s.now().UTC()
	// Sessions past both expiries can no longer be used, so pruning them is invisible.
	if err := db.DeleteExpiredSessions(ctx, s.x, now); err != nil {
		return Tokens{}, err
	}
	raw := make([]byte, 16+32)
	if _, err := rand.Read(raw); err != nil {
		return Tokens{}, fmt.Errorf("auth: generating session id: %w", err)
	}
	id := hex.EncodeToString(raw[:16])
	refresh := base64.RawURLEncoding.EncodeToString(raw[16:])
	exp := now.Add(s.accessTTL)
	err := db.InsertSession(ctx, s.x, db.Session{
		ID: id, UserID: u.ID, ExpiresAt: exp,
		RefreshHash: hashRefresh(refresh), RefreshExpiresAt: refreshExpires,
	})
	if err != nil {
		return Tokens{}, err
	}
	claims := jwt.RegisteredClaims{
		Subject: u.Username, Issuer: issuer, ID: id,
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(exp),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(s.keys.Private)
	if err != nil {
		return Tokens{}, fmt.Errorf("auth: signing token: %w", err)
	}
	return Tokens{Token: token, RefreshToken: refresh}, nil
}

// parse checks signature, algorithm (EdDSA only, which rules out "none" and
// key-confusion forgeries), issuer and expiry.
func (s *Service) parse(token string) (*jwt.RegisteredClaims, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) { return s.keys.Public, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithIssuer(issuer), jwt.WithExpirationRequired(), jwt.WithTimeFunc(s.now))
	if err != nil || claims.ID == "" {
		return nil, ErrInvalidToken
	}
	return &claims, nil
}

// Verify accepts a token only if it is genuine, unexpired and its session is
// not revoked. Name and role are read fresh from the database.
func (s *Service) Verify(ctx context.Context, token string) (User, error) {
	claims, err := s.parse(token)
	if err != nil {
		return User{}, err
	}
	u, err := db.GetSessionUser(ctx, s.x, claims.ID, s.now())
	if errors.Is(err, db.ErrNotFound) {
		return User{}, ErrInvalidToken
	}
	if err != nil {
		return User{}, fmt.Errorf("auth: verifying session: %w", err)
	}
	return toUser(u), nil
}

// Logout revokes the token's session. A token that is not genuine is refused.
func (s *Service) Logout(ctx context.Context, token string) error {
	claims, err := s.parse(token)
	if err != nil {
		return err
	}
	return db.RevokeSession(ctx, s.x, claims.ID, s.now())
}
