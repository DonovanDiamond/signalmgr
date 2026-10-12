// Package auth signs users in (local accounts and optional Active Directory)
// and issues and verifies the JWTs that carry the session. It knows nothing
// about HTTP; the api package wraps it in middleware.
package auth

import (
	"context"
	"crypto/rand"
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
	// ErrNoRole means the directory accepted the password but the person is
	// in none of the groups mapped to a role, so they may not sign in.
	ErrNoRole = errors.New("not a member of any group mapped to a role")
)

// User is the signed-in identity handed to the api layer.
type User struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

// Service issues and verifies sessions.
type Service struct {
	x     *sqlx.DB
	keys  *KeyPair
	ttl   time.Duration
	dir   directory // nil when AD is disabled
	log   zerolog.Logger
	now   func() time.Time
	dummy []byte // compared against when no real hash exists, to keep timing even
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
	s := &Service{x: x, keys: keys, ttl: cfg.JWT.TokenTTL, log: log, now: time.Now, dummy: dummy}
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
	if err := db.InsertLocalUser(ctx, s.x, fa.Username, fa.Username, RoleAdmin, string(hash)); err != nil {
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

// Login checks the credentials and returns a signed token. A local account
// is judged by its own password only, so an AD outage cannot affect it.
// Any other username is tried against the directory when AD is enabled.
func (s *Service) Login(ctx context.Context, username, password string) (string, User, error) {
	if username == "" || password == "" {
		return "", User{}, ErrInvalidCredentials
	}
	stored, err := db.GetUserByUsername(ctx, s.x, username)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", User{}, fmt.Errorf("auth: login of %q: %w", username, err)
	}

	var u db.User
	switch {
	case err == nil && stored.Source == db.SourceLocal:
		if !s.passwordMatches(stored.PasswordHash, password) {
			return "", User{}, ErrInvalidCredentials
		}
		u = stored
	case s.dir != nil:
		du, err := s.dir.Authenticate(ctx, username, password)
		if err != nil {
			return "", User{}, err
		}
		if u, err = db.UpsertADUser(ctx, s.x, username, du.Name, du.Role); err != nil {
			return "", User{}, fmt.Errorf("auth: recording directory user %q: %w", username, err)
		}
	default:
		_ = bcrypt.CompareHashAndPassword(s.dummy, []byte(password))
		return "", User{}, ErrInvalidCredentials
	}

	token, err := s.issue(ctx, u)
	if err != nil {
		return "", User{}, err
	}
	return token, User{Username: u.Username, Name: u.Name, Role: u.Role}, nil
}

// passwordMatches always spends one bcrypt comparison, even without a hash.
func (s *Service) passwordMatches(hash *string, password string) bool {
	if hash == nil {
		_ = bcrypt.CompareHashAndPassword(s.dummy, []byte(password))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(*hash), []byte(password)) == nil
}

func (s *Service) issue(ctx context.Context, u db.User) (string, error) {
	now := s.now().UTC()
	// Expired sessions can no longer verify, so pruning them is invisible.
	if err := db.DeleteExpiredSessions(ctx, s.x, now); err != nil {
		return "", err
	}
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", fmt.Errorf("auth: generating session id: %w", err)
	}
	id := hex.EncodeToString(idBytes)
	exp := now.Add(s.ttl)
	if err := db.InsertSession(ctx, s.x, db.Session{ID: id, UserID: u.ID, ExpiresAt: exp}); err != nil {
		return "", err
	}
	claims := jwt.RegisteredClaims{
		Subject: u.Username, Issuer: issuer, ID: id,
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(exp),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(s.keys.Private)
	if err != nil {
		return "", fmt.Errorf("auth: signing token: %w", err)
	}
	return token, nil
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
	return User{Username: u.Username, Name: u.Name, Role: u.Role}, nil
}

// Logout revokes the token's session. A token that is not genuine is refused.
func (s *Service) Logout(ctx context.Context, token string) error {
	claims, err := s.parse(token)
	if err != nil {
		return err
	}
	return db.RevokeSession(ctx, s.x, claims.ID, s.now())
}
