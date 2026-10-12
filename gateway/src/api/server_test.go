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
}

func newEnv(t *testing.T) *env {
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
	return &env{app: New(a, sig.New(x, fake.URL, log), "Test Install", webFS, nil, log), fake: fake}
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
