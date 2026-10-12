package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/DonovanDiamond/signalmgr/gateway/src/auth"
	"github.com/DonovanDiamond/signalmgr/gateway/src/config"
	"github.com/DonovanDiamond/signalmgr/gateway/src/db"
	"github.com/DonovanDiamond/signalmgr/gateway/src/fakesignal"
	sig "github.com/DonovanDiamond/signalmgr/gateway/src/signal"
)

const (
	adminPass = "admin-password-1234"
	userPass  = "user-password-12345"
)

// env is one app wired to a temp SQLite database and a fake signal server.
// The signalmgr library keeps its URL in a global, so tests do not run in parallel.
type env struct {
	app  *fiber.App
	fake *httptest.Server
	auth *auth.Service
}

func newEnv(t *testing.T) *env { return newEnvProxies(t, nil) }

func newEnvProxies(t *testing.T, trusted []string) *env {
	t.Helper()
	dir := t.TempDir()
	yaml := fmt.Sprintf("db: %s\njwt:\n  private_key_path: %s\n  public_key_path: %s\nfirst_admin:\n  password: %s\n",
		filepath.Join(dir, "t.db"), filepath.Join(dir, "priv.pem"), filepath.Join(dir, "pub.pem"), adminPass)
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	x, err := db.Open(cfg.DB)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { x.Close() })
	log := zerolog.Nop()
	a, err := auth.New(cfg, x, log)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(userPass), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InsertLocalUser(t.Context(), x, "viewer", "Viewer", auth.RoleUser, string(hash)); err != nil {
		t.Fatal(err)
	}

	fake := httptest.NewServer(fakesignal.New(true))
	t.Cleanup(fake.Close)
	webFS := fstest.MapFS{"index.html": {Data: []byte("<html>spa</html>")}, "app.js": {Data: []byte("js")}}
	return &env{app: New(a, sig.New(x, fake.URL, log), "Test Install", trusted, webFS, log), fake: fake, auth: a}
}

type reply struct {
	status int
	body   string
}

func (e *env) do(t *testing.T, method, path, token, body string) reply {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := e.app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second, FailOnTimeout: true})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return reply{res.StatusCode, string(b)}
}

func (e *env) login(t *testing.T, username, password string) string {
	t.Helper()
	r := e.do(t, "POST", "/api/auth/login", "", fmt.Sprintf(`{"username":%q,"password":%q}`, username, password))
	if r.status != 200 {
		t.Fatalf("login %s: %d %s", username, r.status, r.body)
	}
	var out struct{ Token string }
	if err := json.Unmarshal([]byte(r.body), &out); err != nil || out.Token == "" {
		t.Fatalf("login body %q: %v", r.body, err)
	}
	return out.Token
}

func TestLogin(t *testing.T) {
	e := newEnv(t)
	tests := []struct {
		name, body string
		status     int
		contains   string
	}{
		{"ok", `{"username":"admin","password":"` + adminPass + `"}`, 200, `"role":"admin"`},
		{"wrong password", `{"username":"admin","password":"nope-nope-nope"}`, 401, "wrong username or password"},
		{"unknown user", `{"username":"ghost","password":"` + adminPass + `"}`, 401, "wrong username or password"},
		{"not json", `hello`, 400, "JSON object"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := e.do(t, "POST", "/api/auth/login", "", tc.body)
			if r.status != tc.status || !strings.Contains(r.body, tc.contains) {
				t.Errorf("got %d %s, want %d containing %q", r.status, r.body, tc.status, tc.contains)
			}
		})
	}
}

func TestLoginRateLimit(t *testing.T) {
	e := newEnv(t)
	bad := `{"username":"admin","password":"nope-nope-nope"}`
	for i := 0; i < loginFailuresPerMinute; i++ {
		if r := e.do(t, "POST", "/api/auth/login", "", bad); r.status != 401 {
			t.Fatalf("attempt %d: %d", i, r.status)
		}
	}
	if r := e.do(t, "POST", "/api/auth/login", "", bad); r.status != 429 {
		t.Errorf("after %d failures: %d %s", loginFailuresPerMinute, r.status, r.body)
	}
}

func TestAuthRequiredAndRoles(t *testing.T) {
	e := newEnv(t)
	admin := e.login(t, "admin", adminPass)
	viewer := e.login(t, "viewer", userPass)

	tests := []struct {
		name, method, path, body, token string
		status                          int
	}{
		{"numbers without token", "GET", "/api/numbers", "", "", 401},
		{"numbers with garbage token", "GET", "/api/numbers", "", "garbage", 401},
		{"me without token", "GET", "/api/auth/me", "", "", 401},
		{"user may read numbers", "GET", "/api/numbers", "", viewer, 200},
		{"user may read gateway", "GET", "/api/gateway", "", viewer, 200},
		{"user add", "POST", "/api/numbers", `{"number":"+27820000001"}`, viewer, 403},
		{"user relabel", "PATCH", "/api/numbers/%2B27821234567", `{"label":"x"}`, viewer, 403},
		{"user register", "POST", "/api/numbers/%2B27821234567/register", `{"captcha":"signalcaptcha://x"}`, viewer, 403},
		{"user verify", "POST", "/api/numbers/%2B27821234567/verify", `{"code":"123456"}`, viewer, 403},
		{"user unregister", "POST", "/api/numbers/%2B27821234567/unregister", `{}`, viewer, 403},
		{"user remove device", "DELETE", "/api/numbers/%2B27821234567/devices/2", "", viewer, 403},
		{"user set pin", "PUT", "/api/numbers/%2B27821234567/pin", `{"pin":"1234"}`, viewer, 403},
		{"user remove pin", "DELETE", "/api/numbers/%2B27821234567/pin", "", viewer, 403},
		{"user rate-limit", "POST", "/api/numbers/%2B27821234567/rate-limit", `{}`, viewer, 403},
		{"user link qr", "GET", "/api/link/qr?deviceName=x", "", viewer, 403},
		{"user link uri", "GET", "/api/link/uri?deviceName=x", "", viewer, 403},
		{"user audit", "GET", "/api/audit", "", viewer, 403},
		{"admin audit", "GET", "/api/audit", "", admin, 200},
		{"admin link uri", "GET", "/api/link/uri?deviceName=x", "", admin, 200},
		{"unknown api path", "GET", "/api/nope", "", admin, 404},
		{"unknown api path without token", "POST", "/api/nope", "", "", 404},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if r := e.do(t, tc.method, tc.path, tc.token, tc.body); r.status != tc.status {
				t.Errorf("got %d %s, want %d", r.status, r.body, tc.status)
			}
		})
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	e := newEnv(t)
	token := e.login(t, "admin", adminPass)
	if r := e.do(t, "POST", "/api/auth/logout", token, ""); r.status != 204 {
		t.Fatalf("logout: %d %s", r.status, r.body)
	}
	if r := e.do(t, "GET", "/api/auth/me", token, ""); r.status != 401 {
		t.Errorf("after logout: %d %s", r.status, r.body)
	}
}

func TestGatewayInfo(t *testing.T) {
	e := newEnv(t)
	r := e.do(t, "GET", "/api/gateway", e.login(t, "admin", adminPass), "")
	if r.status != 200 || !strings.Contains(r.body, `"online":true`) || !strings.Contains(r.body, `"installationName":"Test Install"`) {
		t.Errorf("got %d %s", r.status, r.body)
	}
}

func TestAddNumber(t *testing.T) {
	e := newEnv(t)
	admin := e.login(t, "admin", adminPass)
	tests := []struct {
		name, body string
		status     int
	}{
		{"ok", `{"number":"+27820000001","label":"Front desk"}`, 204},
		{"duplicate", `{"number":"+27820000001","label":"again"}`, 409},
		{"no plus", `{"number":"27820000002"}`, 400},
		{"letters", `{"number":"+2782abc"}`, 400},
		{"too long label", `{"number":"+27820000003","label":"` + strings.Repeat("x", 65) + `"}`, 400},
		{"empty body object", `{}`, 400},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if r := e.do(t, "POST", "/api/numbers", admin, tc.body); r.status != tc.status {
				t.Errorf("got %d %s, want %d", r.status, r.body, tc.status)
			}
		})
	}
}

// app.Test cannot show the 413: fasthttp rejects the body before the handler
// chain, so this goes over a real socket.
func TestBodyLimit(t *testing.T) {
	e := newEnv(t)
	admin := e.login(t, "admin", adminPass)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go e.app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true})
	t.Cleanup(func() { e.app.Shutdown() })

	body := `{"number":"+27820000009","label":"` + strings.Repeat("x", bodyLimit) + `"}`
	req, _ := http.NewRequest("POST", "http://"+ln.Addr().String()+"/api/numbers", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+admin)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != 413 || !strings.Contains(string(b), `"error"`) {
		t.Errorf("got %d %s", res.StatusCode, b)
	}
}

func stateOf(t *testing.T, body, number string) string {
	t.Helper()
	var ns []struct{ Number, State string }
	if err := json.Unmarshal([]byte(body), &ns); err != nil {
		t.Fatalf("numbers body %q: %v", body, err)
	}
	for _, n := range ns {
		if n.Number == number {
			return n.State
		}
	}
	return "absent"
}

func TestRegisterVerifyFlow(t *testing.T) {
	e := newEnv(t)
	admin := e.login(t, "admin", adminPass)
	const number = "+27829990001"

	if r := e.do(t, "POST", "/api/numbers", admin, `{"number":"`+number+`","label":"New"}`); r.status != 204 {
		t.Fatal(r)
	}
	if got := stateOf(t, e.do(t, "GET", "/api/numbers", admin, "").body, number); got != "not_registered" {
		t.Errorf("after add: %s", got)
	}
	if r := e.do(t, "POST", "/api/numbers/%2B27829990001/register", admin, `{"captcha":"nope"}`); r.status != 400 {
		t.Errorf("bad captcha: %d %s", r.status, r.body)
	}
	if r := e.do(t, "POST", "/api/numbers/%2B27829990001/register", admin, `{"captcha":"signalcaptcha://abc","voice":false}`); r.status != 204 {
		t.Fatalf("register: %d %s", r.status, r.body)
	}
	if got := stateOf(t, e.do(t, "GET", "/api/numbers", admin, "").body, number); got != "awaiting_code" {
		t.Errorf("after register: %s", got)
	}
	if r := e.do(t, "POST", "/api/numbers/%2B27829990001/verify", admin, `{"code":"000000"}`); r.status != 400 {
		t.Errorf("wrong code should pass the upstream 400 through: %d %s", r.status, r.body)
	}
	if r := e.do(t, "POST", "/api/numbers/%2B27829990001/verify", admin, `{"code":"`+fakesignal.ValidCode+`"}`); r.status != 204 {
		t.Fatalf("verify: %d %s", r.status, r.body)
	}
	if got := stateOf(t, e.do(t, "GET", "/api/numbers", admin, "").body, number); got != "registered" {
		t.Errorf("after verify: %s", got)
	}
}

func TestUpstreamErrors(t *testing.T) {
	e := newEnv(t)
	admin := e.login(t, "admin", adminPass)

	t.Run("429 passes through without the request path", func(t *testing.T) {
		r := e.do(t, "POST", "/api/numbers/%2B15550000429/register", admin, `{"captcha":"signalcaptcha://abc"}`)
		if r.status != 429 || strings.Contains(r.body, "/v1/") {
			t.Errorf("got %d %s", r.status, r.body)
		}
	})
	t.Run("bad verify code does not leak into the response", func(t *testing.T) {
		r := e.do(t, "POST", "/api/numbers/%2B27829990001/verify", admin, `{"code":"654321"}`)
		if strings.Contains(r.body, "654321") {
			t.Errorf("code echoed: %s", r.body)
		}
	})
	t.Run("device 1 is refused", func(t *testing.T) {
		if r := e.do(t, "DELETE", "/api/numbers/%2B27821234567/devices/1", admin, ""); r.status != 400 {
			t.Errorf("got %d %s", r.status, r.body)
		}
	})
	t.Run("non-numeric device id", func(t *testing.T) {
		if r := e.do(t, "DELETE", "/api/numbers/%2B27821234567/devices/abc", admin, ""); r.status != 400 {
			t.Errorf("got %d %s", r.status, r.body)
		}
	})
	t.Run("unreachable", func(t *testing.T) {
		e.fake.Close()
		for _, path := range []string{"/api/numbers", "/api/link/uri?deviceName=x"} {
			r := e.do(t, "GET", path, admin, "")
			if r.status != 502 || strings.Contains(r.body, e.fake.URL) {
				t.Errorf("%s: got %d %s", path, r.status, r.body)
			}
		}
		if r := e.do(t, "GET", "/api/gateway", admin, ""); r.status != 200 || !strings.Contains(r.body, `"online":false`) {
			t.Errorf("gateway offline: %d %s", r.status, r.body)
		}
	})
}

func TestAuditRecordsActions(t *testing.T) {
	e := newEnv(t)
	admin := e.login(t, "admin", adminPass)
	e.do(t, "POST", "/api/numbers", admin, `{"number":"+27820000001"}`)
	r := e.do(t, "GET", "/api/audit?limit=5", admin, "")
	if r.status != 200 || !strings.Contains(r.body, `"action":"add_number"`) || !strings.Contains(r.body, `"actor":"admin"`) {
		t.Errorf("got %d %s", r.status, r.body)
	}
	if r := e.do(t, "GET", "/api/audit?limit=0", admin, ""); r.status != 400 {
		t.Errorf("limit 0: %d %s", r.status, r.body)
	}
}

func TestLinkQR(t *testing.T) {
	e := newEnv(t)
	req := httptest.NewRequest("GET", "/api/link/qr?deviceName=Desk", nil)
	req.Header.Set("Authorization", "Bearer "+e.login(t, "admin", adminPass))
	res, err := e.app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second, FailOnTimeout: true})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/png" {
		t.Errorf("got %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
}

func TestSPA(t *testing.T) {
	e := newEnv(t)
	tests := []struct {
		name, path, contains string
		status               int
	}{
		{"root", "/", "spa", 200},
		{"asset", "/app.js", "js", 200},
		{"deep link falls back to index", "/login", "spa", 200},
		{"unknown api path is JSON, not the SPA", "/api/missing", `"error"`, 404},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := e.do(t, "GET", tc.path, "", "")
			if r.status != tc.status || !strings.Contains(r.body, tc.contains) {
				t.Errorf("got %d %s", r.status, r.body)
			}
		})
	}
}

func (e *env) loginTokens(t *testing.T, username, password string) (token, refresh string) {
	t.Helper()
	r := e.do(t, "POST", "/api/auth/login", "", fmt.Sprintf(`{"username":%q,"password":%q}`, username, password))
	var out struct{ Token, RefreshToken string }
	if err := json.Unmarshal([]byte(r.body), &out); r.status != 200 || err != nil {
		t.Fatalf("login %s: %d %s", username, r.status, r.body)
	}
	return out.Token, out.RefreshToken
}

func TestRefreshEndpoint(t *testing.T) {
	e := newEnv(t)
	token, refresh := e.loginTokens(t, "admin", adminPass)
	if r := e.do(t, "POST", "/api/auth/login", "", `{"username":"admin","password":"`+adminPass+`"}`); !strings.Contains(r.body, `"source":"local"`) || !strings.Contains(r.body, `"name":"Administrator"`) {
		t.Errorf("login user shape: %s", r.body)
	}

	r := e.do(t, "POST", "/api/auth/refresh", "", fmt.Sprintf(`{"refreshToken":%q}`, refresh))
	var out struct {
		Token, RefreshToken string
		User                auth.User
	}
	if err := json.Unmarshal([]byte(r.body), &out); r.status != 200 || err != nil || out.Token == "" || out.RefreshToken == refresh || out.User.Username != "admin" {
		t.Fatalf("refresh: %d %s", r.status, r.body)
	}
	if r := e.do(t, "GET", "/api/auth/me", out.Token, ""); r.status != 200 {
		t.Errorf("new token: %d", r.status)
	}
	if r := e.do(t, "GET", "/api/auth/me", token, ""); r.status != 401 {
		t.Errorf("old access token after rotation: %d", r.status)
	}
	tests := []struct {
		name, body string
		status     int
	}{
		{"spent token", fmt.Sprintf(`{"refreshToken":%q}`, refresh), 401},
		{"garbage", `{"refreshToken":"x"}`, 401},
		{"missing", `{}`, 401},
		{"not json", `hi`, 400},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if r := e.do(t, "POST", "/api/auth/refresh", "", tc.body); r.status != tc.status {
				t.Errorf("got %d %s, want %d", r.status, r.body, tc.status)
			}
		})
	}
	for i := 0; i < loginFailuresPerMinute; i++ {
		e.do(t, "POST", "/api/auth/refresh", "", `{"refreshToken":"x"}`)
	}
	if r := e.do(t, "POST", "/api/auth/refresh", "", `{"refreshToken":"x"}`); r.status != 429 {
		t.Errorf("refresh not rate limited: %d", r.status)
	}
}

func TestDisabledUserEndToEnd(t *testing.T) {
	e := newEnv(t)
	admin := e.login(t, "admin", adminPass)
	token, refresh := e.loginTokens(t, "viewer", userPass)
	if r := e.do(t, "PATCH", "/api/users/viewer", admin, `{"disabled":true}`); r.status != 204 {
		t.Fatalf("disable: %d %s", r.status, r.body)
	}
	if r := e.do(t, "GET", "/api/numbers", token, ""); r.status != 401 {
		t.Errorf("disabled user's token: %d", r.status)
	}
	if r := e.do(t, "POST", "/api/auth/refresh", "", fmt.Sprintf(`{"refreshToken":%q}`, refresh)); r.status != 401 {
		t.Errorf("disabled user refresh: %d", r.status)
	}
	if r := e.do(t, "POST", "/api/auth/login", "", `{"username":"viewer","password":"`+userPass+`"}`); r.status != 403 {
		t.Errorf("disabled user login: %d %s", r.status, r.body)
	}
}

func TestUsersRoutes(t *testing.T) {
	e := newEnv(t)
	admin := e.login(t, "admin", adminPass)
	viewer := e.login(t, "viewer", userPass)
	const newUser = `{"username":"carol","name":"Carol","role":"user","password":"carol-password-1"}`
	tests := []struct {
		name, method, path, body, token string
		status                          int
	}{
		{"list without token", "GET", "/api/users", "", "", 401},
		{"list as user", "GET", "/api/users", "", viewer, 403},
		{"create as user", "POST", "/api/users", newUser, viewer, 403},
		{"patch as user", "PATCH", "/api/users/viewer", `{"disabled":true}`, viewer, 403},
		{"reset as user", "PUT", "/api/users/admin/password", `{"password":"hacked-password-1"}`, viewer, 403},
		{"link device as user", "POST", "/api/numbers/%2B27821234567/devices", `{"uri":"sgnl://linkdevice?uuid=x"}`, viewer, 403},
		{"list as admin", "GET", "/api/users", "", admin, 200},
		{"create not json", "POST", "/api/users", `hi`, admin, 400},
		{"create short password", "POST", "/api/users", `{"username":"dave","name":"Dave","role":"user","password":"short"}`, admin, 400},
		{"create bad role", "POST", "/api/users", `{"username":"dave","name":"Dave","role":"root","password":"dave-password-12"}`, admin, 400},
		{"create ok", "POST", "/api/users", newUser, admin, 204},
		{"create duplicate", "POST", "/api/users", newUser, admin, 409},
		{"disable self", "PATCH", "/api/users/admin", `{"disabled":true}`, admin, 400},
		{"patch without field", "PATCH", "/api/users/carol", `{}`, admin, 400},
		{"patch unknown", "PATCH", "/api/users/ghost", `{"disabled":true}`, admin, 404},
		{"patch ok", "PATCH", "/api/users/carol", `{"disabled":true}`, admin, 204},
		{"reset short", "PUT", "/api/users/carol/password", `{"password":"short"}`, admin, 400},
		{"reset unknown", "PUT", "/api/users/ghost/password", `{"password":"ghost-password-123"}`, admin, 404},
		{"reset ok", "PUT", "/api/users/carol/password", `{"password":"carol-reset-password"}`, admin, 204},
		{"link device ok", "POST", "/api/numbers/%2B27821234567/devices", `{"uri":"sgnl://linkdevice?uuid=x"}`, admin, 204},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if r := e.do(t, tc.method, tc.path, tc.token, tc.body); r.status != tc.status {
				t.Errorf("got %d %s, want %d", r.status, r.body, tc.status)
			}
		})
	}
	r := e.do(t, "GET", "/api/users", admin, "")
	if !strings.Contains(r.body, `{"username":"carol","name":"Carol","role":"user","source":"local","disabled":true}`) {
		t.Errorf("list body: %s", r.body)
	}
	if a := e.do(t, "GET", "/api/audit", admin, ""); !strings.Contains(a.body, "create_user") || !strings.Contains(a.body, "reset_user_password") {
		t.Errorf("user management not audited: %s", a.body)
	}
}

func TestChangePasswordRoute(t *testing.T) {
	e := newEnv(t)
	viewer := e.login(t, "viewer", userPass)
	tests := []struct {
		name, body string
		status     int
	}{
		{"wrong current is 400 not 401", `{"current":"wrong-password-xx","new":"another-password-1"}`, 400},
		{"too short", `{"current":"` + userPass + `","new":"short"}`, 400},
		{"not json", `hi`, 400},
		{"ok", `{"current":"` + userPass + `","new":"another-password-1"}`, 204},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if r := e.do(t, "POST", "/api/auth/password", viewer, tc.body); r.status != tc.status {
				t.Errorf("got %d %s, want %d", r.status, r.body, tc.status)
			}
		})
	}
	if r := e.do(t, "POST", "/api/auth/password", "", `{}`); r.status != 401 {
		t.Errorf("without token: %d", r.status)
	}
	e.login(t, "viewer", "another-password-1")
}

// app.Test connects from 0.0.0.0; X-Forwarded-For only counts when that
// address is a trusted proxy.
func TestTrustedProxyKeysLimiter(t *testing.T) {
	bad := `{"username":"admin","password":"nope-nope-nope"}`
	tests := []struct {
		name    string
		trusted []string
		// the Nth failure from a different forwarded address is limited or not
		wantLimited bool
	}{
		{"untrusted: header ignored, all clients share the proxy's address", nil, true},
		{"trusted: each forwarded client has its own counter", []string{"0.0.0.0/32"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnvProxies(t, tc.trusted)
			last := 0
			for i := 0; i <= loginFailuresPerMinute; i++ {
				req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(bad))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Forwarded-For", fmt.Sprintf("203.0.113.%d", i+1))
				res, err := e.app.Test(req)
				if err != nil {
					t.Fatal(err)
				}
				res.Body.Close()
				last = res.StatusCode
			}
			if limited := last == 429; limited != tc.wantLimited {
				t.Errorf("last status %d, limited = %t, want %t", last, limited, tc.wantLimited)
			}
		})
	}
}

// The proxy appends the real client address; whatever the client put to its
// left must not give it a fresh limiter key.
func TestSpoofedForwardedForSharesLimiter(t *testing.T) {
	bad := `{"username":"admin","password":"nope-nope-nope"}`
	e := newEnvProxies(t, []string{"0.0.0.0/32"})
	last := 0
	for i := 0; i <= loginFailuresPerMinute; i++ {
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(bad))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("203.0.113.%d, 198.51.100.7", i+1))
		res, err := e.app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		last = res.StatusCode
	}
	if last != 429 {
		t.Errorf("last status %d, want 429: spoofed leftmost entries bypass the limiter", last)
	}
}
