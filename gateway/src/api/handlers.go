package api

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/DonovanDiamond/signalmgr/gateway/src/auth"
	"github.com/DonovanDiamond/signalmgr/gateway/src/signal"
)

// body decodes the JSON request body into T. A malformed body is a 400 that
// says what was expected; the body itself is never echoed.
func body[T any](c fiber.Ctx) (T, error) {
	var v T
	if err := c.Bind().JSON(&v); err != nil {
		return v, fmt.Errorf("%w: request body must be a JSON object with the documented fields", signal.ErrInvalid)
	}
	return v, nil
}

func done(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) }

func (s *server) login(c fiber.Ctx) error {
	req, err := body[struct{ Username, Password string }](c)
	if err != nil {
		return s.fail(c, err)
	}
	tokens, u, err := s.auth.Login(c.Context(), req.Username, req.Password)
	if err != nil {
		return s.fail(c, err)
	}
	return c.JSON(session{tokens, u})
}

// session is the body of a sign-in or refresh.
type session struct {
	auth.Tokens
	User auth.User `json:"user"`
}

func (s *server) refresh(c fiber.Ctx) error {
	req, err := body[struct {
		RefreshToken string `json:"refreshToken"`
	}](c)
	if err != nil {
		return s.fail(c, err)
	}
	tokens, u, err := s.auth.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		return s.fail(c, err)
	}
	return c.JSON(session{tokens, u})
}

func (s *server) changePassword(c fiber.Ctx) error {
	req, err := body[struct{ Current, New string }](c)
	if err == nil {
		err = s.auth.ChangePassword(c.Context(), currentUser(c).Username, req.Current, req.New)
	}
	return s.finish(c, err)
}

func (s *server) users(c fiber.Ctx) error {
	us, err := s.auth.ListUsers(c.Context())
	if err != nil {
		return s.fail(c, err)
	}
	return c.JSON(us)
}

func (s *server) addUser(c fiber.Ctx) error {
	req, err := body[struct{ Username, Name, Role, Password string }](c)
	if err == nil {
		err = s.auth.CreateUser(c.Context(), currentUser(c).Username, req.Username, req.Name, req.Role, req.Password)
	}
	return s.finish(c, err)
}

func (s *server) patchUser(c fiber.Ctx) error {
	// A pointer so that a body without "disabled" is refused, not read as false.
	req, err := body[struct{ Disabled *bool }](c)
	if err == nil && req.Disabled == nil {
		err = fmt.Errorf("%w: request body must be a JSON object with the documented fields", signal.ErrInvalid)
	}
	if err == nil {
		err = s.auth.SetDisabled(c.Context(), currentUser(c).Username, c.Params("username"), *req.Disabled)
	}
	return s.finish(c, err)
}

func (s *server) resetPassword(c fiber.Ctx) error {
	req, err := body[struct{ Password string }](c)
	if err == nil {
		err = s.auth.SetPassword(c.Context(), currentUser(c).Username, c.Params("username"), req.Password)
	}
	return s.finish(c, err)
}

func (s *server) linkDevice(c fiber.Ctx) error {
	req, err := body[struct{ URI string }](c)
	if err == nil {
		err = s.signal.LinkDevice(c.Context(), currentUser(c).Username, c.Params("number"), req.URI)
	}
	return s.finish(c, err)
}

func (s *server) logout(c fiber.Ctx) error {
	token, _ := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
	if err := s.auth.Logout(c.Context(), token); err != nil {
		return s.fail(c, err)
	}
	return done(c)
}

func (s *server) me(c fiber.Ctx) error { return c.JSON(currentUser(c)) }

func (s *server) gateway(c fiber.Ctx) error {
	return c.JSON(struct {
		signal.GatewayInfo
		InstallationName string `json:"installationName"`
	}{s.signal.Gateway(c.Context()), s.installationName})
}

func (s *server) numbers(c fiber.Ctx) error {
	ns, err := s.signal.Numbers(c.Context())
	if err != nil {
		return s.fail(c, err)
	}
	return c.JSON(ns)
}

func (s *server) addNumber(c fiber.Ctx) error {
	req, err := body[struct{ Number, Label string }](c)
	if err == nil {
		err = s.signal.AddNumber(c.Context(), currentUser(c).Username, req.Number, req.Label)
	}
	return s.finish(c, err)
}

func (s *server) setLabel(c fiber.Ctx) error {
	req, err := body[struct{ Label string }](c)
	if err == nil {
		err = s.signal.SetLabel(c.Context(), currentUser(c).Username, c.Params("number"), req.Label)
	}
	return s.finish(c, err)
}

func (s *server) register(c fiber.Ctx) error {
	req, err := body[struct {
		Captcha string
		Voice   bool
	}](c)
	if err == nil {
		err = s.signal.Register(c.Context(), currentUser(c).Username, c.Params("number"), req.Captcha, req.Voice)
	}
	return s.finish(c, err)
}

func (s *server) verify(c fiber.Ctx) error {
	req, err := body[struct{ Code, Pin string }](c)
	if err == nil {
		err = s.signal.Verify(c.Context(), currentUser(c).Username, c.Params("number"), req.Code, req.Pin)
	}
	return s.finish(c, err)
}

func (s *server) unregister(c fiber.Ctx) error {
	req, err := body[struct{ DeleteAccount, DeleteLocalData bool }](c)
	if err == nil {
		err = s.signal.Unregister(c.Context(), currentUser(c).Username, c.Params("number"), req.DeleteAccount, req.DeleteLocalData)
	}
	return s.finish(c, err)
}

func (s *server) removeDevice(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return s.fail(c, fmt.Errorf("%w: device id must be a whole number, got %q", signal.ErrInvalid, c.Params("id")))
	}
	return s.finish(c, s.signal.RemoveDevice(c.Context(), currentUser(c).Username, c.Params("number"), id))
}

func (s *server) setPin(c fiber.Ctx) error {
	req, err := body[struct{ Pin string }](c)
	if err == nil {
		err = s.signal.SetPin(c.Context(), currentUser(c).Username, c.Params("number"), req.Pin)
	}
	return s.finish(c, err)
}

func (s *server) removePin(c fiber.Ctx) error {
	return s.finish(c, s.signal.RemovePin(c.Context(), currentUser(c).Username, c.Params("number")))
}

func (s *server) rateLimit(c fiber.Ctx) error {
	req, err := body[struct{ ChallengeToken, Captcha string }](c)
	if err == nil {
		err = s.signal.RateLimit(c.Context(), currentUser(c).Username, c.Params("number"), req.ChallengeToken, req.Captcha)
	}
	return s.finish(c, err)
}

func (s *server) linkQR(c fiber.Ctx) error {
	png, err := s.signal.LinkQR(c.Context(), c.Query("deviceName"))
	if err != nil {
		return s.fail(c, err)
	}
	c.Set(fiber.HeaderContentType, "image/png")
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Send(png)
}

func (s *server) linkURI(c fiber.Ctx) error {
	uri, err := s.signal.LinkURI(c.Context(), c.Query("deviceName"))
	if err != nil {
		return s.fail(c, err)
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(fiber.Map{"uri": uri})
}

func (s *server) audit(c fiber.Ctx) error {
	limit := 200
	if q := c.Query("limit"); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 1 || n > 1000 {
			return s.fail(c, fmt.Errorf("%w: limit must be a whole number from 1 to 1000, got %q", signal.ErrInvalid, q))
		}
		limit = n
	}
	rows, err := s.signal.Audit(c.Context(), limit)
	if err != nil {
		return s.fail(c, err)
	}
	return c.JSON(rows)
}

// finish ends a mutating handler: 204 on success, mapped error otherwise.
func (s *server) finish(c fiber.Ctx, err error) error {
	if err != nil {
		return s.fail(c, err)
	}
	return done(c)
}
