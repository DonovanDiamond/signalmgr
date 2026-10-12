package signal

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DonovanDiamond/signalmgr"
	"github.com/DonovanDiamond/signalmgr/gateway/src/db"
	"github.com/DonovanDiamond/signalmgr/gateway/src/fakesignal"
	"github.com/rs/zerolog"
)

// The library keeps its URL in a global, so these tests must not run in parallel.
func setup(t *testing.T, seed bool) (*Service, string) {
	t.Helper()
	x, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { x.Close() })
	ts := httptest.NewServer(fakesignal.New(seed))
	t.Cleanup(ts.Close)
	return New(x, ts.URL, zerolog.Nop()), ts.URL
}

func find(t *testing.T, ns []Number, number string) Number {
	t.Helper()
	for _, n := range ns {
		if n.Number == number {
			return n
		}
	}
	t.Fatalf("%s not in %+v", number, ns)
	return Number{}
}

func lastAudit(t *testing.T, s *Service) db.AuditEntry {
	t.Helper()
	rows, err := s.Audit(context.Background(), 1)
	if err != nil || len(rows) != 1 {
		t.Fatalf("audit: %v %v", rows, err)
	}
	return rows[0]
}

func TestStateMerge(t *testing.T) {
	s, _ := setup(t, true)
	ctx := context.Background()
	if err := s.AddNumber(ctx, "a", "+27821234567", "Reception"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddNumber(ctx, "a", "+27829990000", "Spare"); err != nil {
		t.Fatal(err)
	}
	if err := s.Register(ctx, "a", "+27828880000", "signalcaptcha://x", false); err != nil {
		t.Fatal(err)
	}
	ns, err := s.Numbers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	reg := find(t, ns, "+27821234567")
	if reg.State != StateRegistered || reg.Label != "Reception" || len(reg.Devices) != 3 || !reg.Devices[0].Primary || reg.Devices[1].Primary {
		t.Errorf("registered: %+v", reg)
	}
	if reg.Devices[0].Created == nil {
		t.Error("created should be set")
	}
	if n := find(t, ns, "+27835550142"); n.Label != "" || n.State != StateRegistered {
		t.Errorf("upstream-only: %+v", n)
	}
	if n := find(t, ns, "+27829990000"); n.State != StateNotRegistered || n.Devices == nil {
		t.Errorf("db-only: %+v", n)
	}
	if n := find(t, ns, "+27828880000"); n.State != StateAwaitingCode {
		t.Errorf("awaiting: %+v", n)
	}
}

func TestRegisterVerifyFlow(t *testing.T) {
	s, _ := setup(t, false)
	ctx := context.Background()
	const num = "+27821110000"
	if err := s.Register(ctx, "bob", num, "signalcaptcha://x", true); err != nil {
		t.Fatal(err)
	}
	if err := s.Verify(ctx, "bob", num, "654321", ""); err == nil {
		t.Fatal("wrong code accepted")
	} else {
		var apiErr *signalmgr.APIError
		if !errors.As(err, &apiErr) || apiErr.Status != 400 {
			t.Fatalf("want APIError 400, got %v", err)
		}
	}
	if a := lastAudit(t, s); a.Action != "verify" || a.Result == "ok" || strings.Contains(a.Result+a.Detail, "654321") {
		t.Errorf("audit after wrong code: %+v", a)
	}
	if n := find(t, mustNumbers(t, s), num); n.State != StateAwaitingCode {
		t.Fatalf("state %s", n.State)
	}
	if err := s.Verify(ctx, "bob", num, "123456", "1234"); err != nil {
		t.Fatal(err)
	}
	if n := find(t, mustNumbers(t, s), num); n.State != StateRegistered || len(n.Devices) != 1 {
		t.Fatalf("after verify: %+v", n)
	}
	if err := s.SetPin(ctx, "bob", num, "4321"); err != nil {
		t.Fatal(err)
	}
	if a := lastAudit(t, s); a.Action != "set_pin" || a.Result != "ok" || strings.Contains(a.Detail, "4321") {
		t.Errorf("audit: %+v", a)
	}
	if err := s.Unregister(ctx, "bob", num, false, false); err != nil {
		t.Fatal(err)
	}
	if n := find(t, mustNumbers(t, s), num); n.State != StateNotRegistered {
		t.Fatalf("after unregister: %+v", n)
	}
}

func mustNumbers(t *testing.T, s *Service) []Number {
	t.Helper()
	ns, err := s.Numbers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return ns
}

func TestRateLimited(t *testing.T) {
	s, _ := setup(t, false)
	err := s.Register(context.Background(), "bob", fakesignal.RateLimitedNumber, "signalcaptcha://x", false)
	var apiErr *signalmgr.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 429 || len(apiErr.ChallengeTokens) == 0 {
		t.Fatalf("want 429 with tokens, got %v", err)
	}
	if a := lastAudit(t, s); !strings.Contains(a.Result, "429") {
		t.Errorf("audit: %+v", a)
	}
}

func TestUnreachable(t *testing.T) {
	s, _ := setup(t, false)
	s = New(s.db, "http://127.0.0.1:1", zerolog.Nop())
	if _, err := s.Numbers(context.Background()); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("Numbers: %v", err)
	}
	err := s.Verify(context.Background(), "bob", "+27821110000", "123456", "")
	if !errors.Is(err, ErrUnreachable) || strings.Contains(err.Error(), "123456") {
		t.Fatalf("Verify: %v", err)
	}
	if g := s.Gateway(context.Background()); g.Online {
		t.Error("gateway should be offline")
	}
}

func TestValidation(t *testing.T) {
	s, _ := setup(t, true)
	ctx := context.Background()
	const ok = "+27821234567"
	long := strings.Repeat("x", 65)
	tests := []struct {
		name string
		call func() error
	}{
		{"add bad number", func() error { return s.AddNumber(ctx, "a", "0821234567", "") }},
		{"add long label", func() error { return s.AddNumber(ctx, "a", ok, long) }},
		{"label bad number", func() error { return s.SetLabel(ctx, "a", "+0123456", "x") }},
		{"register bad captcha", func() error { return s.Register(ctx, "a", ok, "abc", false) }},
		{"verify short code", func() error { return s.Verify(ctx, "a", ok, "12345", "") }},
		{"verify letters", func() error { return s.Verify(ctx, "a", ok, "12345a", "") }},
		{"verify short pin", func() error { return s.Verify(ctx, "a", ok, "123456", "123") }},
		{"pin letters", func() error { return s.SetPin(ctx, "a", ok, "12ab") }},
		{"pin too long", func() error { return s.SetPin(ctx, "a", ok, strings.Repeat("1", 21)) }},
		{"primary device", func() error { return s.RemoveDevice(ctx, "a", ok, 1) }},
		{"rate limit no token", func() error { return s.RateLimit(ctx, "a", ok, "", "signalcaptcha://x") }},
		{"rate limit bad captcha", func() error { return s.RateLimit(ctx, "a", ok, "tok", "x") }},
		{"qr empty name", func() error { _, err := s.LinkQR(ctx, ""); return err }},
		{"uri long name", func() error { _, err := s.LinkURI(ctx, long); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, ErrInvalid) {
				t.Fatalf("want ErrInvalid, got %v", err)
			}
		})
	}
	for _, n := range []string{"+27821234567", "+1234567", "+123456789012345"} {
		if err := ValidateNumber(n); err != nil {
			t.Errorf("%s: %v", n, err)
		}
	}
	for _, n := range []string{"", "27821234567", "+0123456", "+123456", "+1234567890123456", "+27 82"} {
		if err := ValidateNumber(n); !errors.Is(err, ErrInvalid) {
			t.Errorf("%q accepted", n)
		}
	}
}

func TestRemoveDeviceAndLink(t *testing.T) {
	s, _ := setup(t, true)
	ctx := context.Background()
	if err := s.RemoveDevice(ctx, "a", "+27821234567", 3); err != nil {
		t.Fatal(err)
	}
	if n := find(t, mustNumbers(t, s), "+27821234567"); len(n.Devices) != 2 {
		t.Fatalf("devices: %+v", n.Devices)
	}
	if png, err := s.LinkQR(ctx, "Desk"); err != nil || len(png) < 8 {
		t.Fatalf("qr: %d %v", len(png), err)
	}
	if uri, err := s.LinkURI(ctx, "Desk"); err != nil || !strings.HasPrefix(uri, "sgnl://") {
		t.Fatalf("uri: %q %v", uri, err)
	}
	if a := lastAudit(t, s); a.Action != "remove_device" || a.Actor != "a" || a.Result != "ok" {
		t.Errorf("audit: %+v", a)
	}
}

func TestLinkDevice(t *testing.T) {
	s, _ := setup(t, true)
	ctx := context.Background()
	uri := "sgnl://linkdevice?uuid=x&pub_key=y"
	if err := s.LinkDevice(ctx, "a", "+27835550142", uri); err != nil {
		t.Fatal(err)
	}
	if n := find(t, mustNumbers(t, s), "+27835550142"); len(n.Devices) != 2 {
		t.Fatalf("devices: %+v", n.Devices)
	}
	if a := lastAudit(t, s); a.Action != "link_device" || a.Actor != "a" || a.Result != "ok" || strings.Contains(a.Detail, "uuid") {
		t.Errorf("audit: %+v", a)
	}
	for name, c := range map[string][2]string{
		"bad number": {"12", uri},
		"bad prefix": {"+27835550142", "http://x"},
		"too long":   {"+27835550142", uri + strings.Repeat("a", maxLinkURI)},
	} {
		if err := s.LinkDevice(ctx, "a", c[0], c[1]); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: want ErrInvalid, got %v", name, err)
		}
	}
	if err := s.LinkDevice(ctx, "a", "+27000000000", uri); err == nil || errors.Is(err, ErrInvalid) {
		t.Errorf("unknown number: want upstream error, got %v", err)
	}
	if a := lastAudit(t, s); a.Result == "ok" {
		t.Errorf("failure not audited: %+v", a)
	}
}
