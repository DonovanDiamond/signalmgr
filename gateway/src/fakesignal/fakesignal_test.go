package fakesignal

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestRegisterCases(t *testing.T) {
	tests := []struct {
		name, number, body string
		want               int
	}{
		{"ok", "+27820000001", `{"captcha":"signalcaptcha://abc"}`, http.StatusCreated},
		{"bad captcha", "+27820000001", `{"captcha":"nope"}`, http.StatusBadRequest},
		{"rate limited", RateLimitedNumber, `{"captcha":"signalcaptcha://abc"}`, http.StatusTooManyRequests},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, New(false), "POST", "/v1/register/"+tt.number, tt.body)
			if rec.Code != tt.want {
				t.Fatalf("status %d, want %d: %s", rec.Code, tt.want, rec.Body)
			}
			if tt.want == http.StatusTooManyRequests && !strings.Contains(rec.Body.String(), "challenge_tokens") {
				t.Fatalf("429 body lacks challenge_tokens: %s", rec.Body)
			}
		})
	}
}

func TestVerifyFlow(t *testing.T) {
	s := New(false)
	do(t, s, "POST", "/v1/register/+27820000001", `{"captcha":"signalcaptcha://x"}`)
	if rec := do(t, s, "POST", "/v1/register/+27820000001/verify/000000", `{}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("wrong code: status %d", rec.Code)
	}
	if rec := do(t, s, "POST", "/v1/register/+27820000001/verify/123456", `{}`); rec.Code != http.StatusCreated {
		t.Fatalf("right code: status %d", rec.Code)
	}
	if rec := do(t, s, "GET", "/v1/accounts", ""); !strings.Contains(rec.Body.String(), "+27820000001") {
		t.Fatalf("accounts: %s", rec.Body)
	}
}

func TestSeededDevices(t *testing.T) {
	rec := do(t, New(true), "GET", "/v1/devices/+27821234567", "")
	if rec.Code != http.StatusOK || strings.Count(rec.Body.String(), `"id"`) != 3 {
		t.Fatalf("devices: %d %s", rec.Code, rec.Body)
	}
}

func TestQRCodeIsPNG(t *testing.T) {
	rec := do(t, New(true), "GET", "/v1/qrcodelink?device_name=x", "")
	if !strings.HasPrefix(rec.Body.String(), "\x89PNG") {
		t.Fatal("not a PNG")
	}
}
