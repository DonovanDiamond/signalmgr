// Package api is the gateway's HTTP surface: routing, the JWT middleware, the
// role split and the translation of service errors into JSON responses.
// Handlers only parse input and call auth and signal; the rules live there.
package api

import (
	"errors"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/rs/zerolog"

	"github.com/DonovanDiamond/signalmgr"
	"github.com/DonovanDiamond/signalmgr/gateway/src/auth"
	"github.com/DonovanDiamond/signalmgr/gateway/src/db"
	"github.com/DonovanDiamond/signalmgr/gateway/src/signal"
)

const (
	// Every body is a few short fields; the largest legitimate one is a
	// 4096-character challenge token.
	bodyLimit = 16 << 10

	loginFailuresPerMinute = 10
)

type server struct {
	auth             *auth.Service
	signal           *signal.Service
	installationName string
	log              zerolog.Logger
}

type userKey struct{}

// New builds the app. webFS is web.FS(), which is a "UI not built" fallback
// when there is no build, so the API always has something to serve. Only
// requests from trustedProxies (CIDRs) may set the client IP through
// X-Forwarded-For; with none, the header is ignored.
func New(a *auth.Service, s *signal.Service, installationName string, trustedProxies []string, webFS fs.FS, log zerolog.Logger) *fiber.App {
	srv := &server{auth: a, signal: s, installationName: installationName, log: log}
	cfg := fiber.Config{BodyLimit: bodyLimit, UnescapePath: true, ErrorHandler: srv.errorHandler}
	if len(trustedProxies) > 0 {
		cfg.TrustProxy = true
		cfg.ProxyHeader = fiber.HeaderXForwardedFor
		// Without validation c.IP() is the raw header, so a spoofed leftmost entry mints a fresh limiter key.
		cfg.EnableIPValidation = true
		cfg.TrustProxyConfig = fiber.TrustProxyConfig{Proxies: trustedProxies}
	}
	app := fiber.New(cfg)

	// Each route gets its own counter, keyed on the client IP.
	failureLimit := func() fiber.Handler {
		return limiter.New(limiter.Config{
			Max:                    loginFailuresPerMinute,
			Expiration:             time.Minute,
			SkipSuccessfulRequests: true,
			LimitReached: func(c fiber.Ctx) error {
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many failed sign-in attempts; wait a minute and try again"})
			},
		})
	}

	api := app.Group("/api")
	api.Post("/auth/login", failureLimit(), srv.login)
	api.Post("/auth/refresh", failureLimit(), srv.refresh)

	// Role is enforced here, per route, never inside a handler.
	in, adm := srv.requireAuth, requireAdmin

	api.Post("/auth/logout", in, srv.logout)
	api.Get("/auth/me", in, srv.me)
	api.Post("/auth/password", in, srv.changePassword)
	api.Get("/gateway", in, srv.gateway)
	api.Get("/numbers", in, srv.numbers)
	api.Get("/audit", in, adm, srv.audit)

	api.Get("/users", in, adm, srv.users)
	api.Post("/users", in, adm, srv.addUser)
	api.Patch("/users/:username", in, adm, srv.patchUser)
	api.Put("/users/:username/password", in, adm, srv.resetPassword)

	api.Post("/numbers", in, adm, srv.addNumber)
	api.Patch("/numbers/:number", in, adm, srv.setLabel)
	api.Post("/numbers/:number/register", in, adm, srv.register)
	api.Post("/numbers/:number/verify", in, adm, srv.verify)
	api.Post("/numbers/:number/unregister", in, adm, srv.unregister)
	api.Delete("/numbers/:number/devices/:id", in, adm, srv.removeDevice)
	api.Put("/numbers/:number/pin", in, adm, srv.setPin)
	api.Delete("/numbers/:number/pin", in, adm, srv.removePin)
	api.Post("/numbers/:number/devices", in, adm, srv.linkDevice)
	api.Post("/numbers/:number/rate-limit", in, adm, srv.rateLimit)
	api.Get("/link/qr", in, adm, srv.linkQR)
	api.Get("/link/uri", in, adm, srv.linkURI)

	// Without this an unknown /api path would fall through to the SPA and
	// answer 200 with HTML.
	api.All("/*", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no such endpoint: " + c.Method() + " " + c.Path()})
	})

	// A deep link such as /login is a client-side route, not a missing file.
	app.Get("/*", static.New("", static.Config{
		FS: webFS,
		NotFoundHandler: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusOK).SendFile("index.html", fiber.SendFile{FS: webFS})
		},
	}))
	return app
}

func (s *server) requireAuth(c fiber.Ctx) error {
	token, ok := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
	if !ok || token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "sign in required: send Authorization: Bearer <token>"})
	}
	u, err := s.auth.Verify(c.Context(), token)
	if err != nil {
		return s.fail(c, err)
	}
	c.Locals(userKey{}, u)
	return c.Next()
}

func requireAdmin(c fiber.Ctx) error {
	if currentUser(c).Role != auth.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "this needs the admin role; your account is read-only"})
	}
	return c.Next()
}

func currentUser(c fiber.Ctx) auth.User {
	u, _ := c.Locals(userKey{}).(auth.User)
	return u
}

// fail maps a service error to a status and a message safe to show: upstream
// errors give only Status and Message, because APIError.Error() contains the
// request path, which for verify includes the code.
func (s *server) fail(c fiber.Ctx, err error) error {
	status, msg := fiber.StatusInternalServerError, "internal error"
	var apiErr *signalmgr.APIError
	var timeout *auth.DirectoryTimeoutError
	switch {
	case errors.As(err, &apiErr):
		status, msg = apiErr.Status, apiErr.Message
		if status < 400 || status > 599 {
			status = fiber.StatusBadGateway
		}
		if msg == "" {
			msg = http.StatusText(status)
		}
	case errors.Is(err, signal.ErrInvalid), errors.Is(err, auth.ErrInvalidInput):
		status, msg = fiber.StatusBadRequest, err.Error()
	case errors.Is(err, db.ErrNotFound):
		status, msg = fiber.StatusNotFound, "not found"
	case errors.Is(err, db.ErrDuplicate):
		status, msg = fiber.StatusConflict, "already exists"
	case errors.Is(err, signal.ErrUnreachable):
		status, msg = fiber.StatusBadGateway, "the Signal API is unreachable"
	case errors.Is(err, auth.ErrInvalidCredentials):
		status, msg = fiber.StatusUnauthorized, "wrong username or password"
	case errors.Is(err, auth.ErrInvalidToken):
		status, msg = fiber.StatusUnauthorized, "session is invalid or expired; sign in again"
	case errors.As(err, &timeout):
		status, msg = fiber.StatusServiceUnavailable, timeout.Error()
	case errors.Is(err, auth.ErrDisabled):
		status, msg = fiber.StatusForbidden, "this account is disabled; ask an administrator"
	case errors.Is(err, auth.ErrDirectoryUnreachable):
		status, msg = fiber.StatusServiceUnavailable, "the Active Directory server is unreachable; local accounts still work"
	case errors.Is(err, auth.ErrNoRole):
		status, msg = fiber.StatusForbidden, "your account is not in any group that grants access"
	default:
		s.log.Error().Err(err).Str("method", c.Method()).Str("path", c.Path()).Msg("unhandled error")
	}
	return c.Status(status).JSON(fiber.Map{"error": msg})
}

// errorHandler covers errors fiber raises itself (oversized body, bad method).
func (s *server) errorHandler(c fiber.Ctx, err error) error {
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
	}
	return s.fail(c, err)
}
