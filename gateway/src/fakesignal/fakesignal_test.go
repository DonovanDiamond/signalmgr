package fakesignal

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestLinkDevice(t *testing.T) {
	h := New(true)
	uri := `{"uri":"sgnl://linkdevice?uuid=x&pub_key=y"}`
	if rec := do(t, h, "POST", "/v1/devices/+27835550142", uri); rec.Code != http.StatusNoContent {
		t.Fatalf("link: %d %s", rec.Code, rec.Body)
	}
	rec := do(t, h, "GET", "/v1/devices/+27835550142", "")
	if strings.Count(rec.Body.String(), `"id"`) != 2 || !strings.Contains(rec.Body.String(), `"New device"`) {
		t.Fatalf("devices: %s", rec.Body)
	}
	if rec := do(t, h, "POST", "/v1/devices/+27835550142", `{"uri":"http://evil"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad uri: %d", rec.Code)
	}
	if rec := do(t, h, "POST", "/v1/devices/+27000000000", uri); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown number: %d", rec.Code)
	}
}

func TestLinkAfterAddsAccount(t *testing.T) {
	h := New(false)
	do(t, h, "GET", "/v1/qrcodelink/raw?device_name=x", "")
	if rec := do(t, h, "GET", "/v1/accounts", ""); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("LinkAfter zero must never link: %s", rec.Body)
	}
	h.LinkAfter = 10 * time.Millisecond
	do(t, h, "GET", "/v1/qrcodelink?device_name=x", "")
	time.Sleep(200 * time.Millisecond)
	if rec := do(t, h, "GET", "/v1/accounts", ""); !strings.Contains(rec.Body.String(), "+27605550001") {
		t.Fatalf("accounts: %s", rec.Body)
	}
}

func TestSeedDesign(t *testing.T) {
	h := New(false)
	h.SeedDesign()
	rec := do(t, h, "GET", "/v1/devices/+27825550143", "")
	if strings.Count(rec.Body.String(), `"id"`) != 3 {
		t.Fatalf("devices: %s", rec.Body)
	}
	if rec := do(t, h, "GET", "/v1/devices/+27715550198", ""); rec.Code != http.StatusOK {
		t.Fatalf("pastoral: %d", rec.Code)
	}
}

func TestAccountsFormats(t *testing.T) {
	strs := do(t, New(true), "GET", "/v1/accounts", "")
	if !strings.HasPrefix(strs.Body.String(), `["+27821234567"`) {
		t.Errorf("strings format: %s", strs.Body)
	}
	s := New(false)
	s.Objects = true
	s.SeedDesign()
	objs := do(t, s, "GET", "/v1/accounts", "").Body.String()
	for _, want := range []string{`{"number":"+27825550143","aci":null}`, `{"number":"","aci":"` + NumberlessACI + `"}`} {
		if !strings.Contains(objs, want) {
			t.Errorf("objects format lacks %s: %s", want, objs)
		}
	}
	if rec := do(t, s, "GET", "/v1/devices/"+NumberlessACI, ""); rec.Code != http.StatusOK {
		t.Errorf("numberless devices: %d", rec.Code)
	}
}
