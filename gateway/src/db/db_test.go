package db

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func open(t *testing.T) *sqlx.DB {
	t.Helper()
	x, err := Open(filepath.Join(t.TempDir(), "sub", "gw.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { x.Close() })
	return x
}

func TestOpenIsIdempotent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "gw.db")
	for i := 0; i < 2; i++ {
		x, err := Open(p)
		if err != nil {
			t.Fatalf("open #%d: %v", i, err)
		}
		x.Close()
	}
}

func TestNumbers(t *testing.T) {
	ctx := context.Background()
	x := open(t)

	if err := InsertNumber(ctx, x, "+27821234567", "Reception"); err != nil {
		t.Fatal(err)
	}
	if err := InsertNumber(ctx, x, "+27821234567", "again"); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate insert = %v, want ErrDuplicate", err)
	}
	if _, err := GetNumber(ctx, x, "+27000000000"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing get = %v, want ErrNotFound", err)
	}
	if err := InsertNumber(ctx, x, "not-a-number", ""); err == nil {
		t.Fatal("malformed number accepted by the CHECK constraint")
	}

	tests := []struct {
		name  string
		do    func() error
		check func(t *testing.T, n Number)
		num   string
	}{
		{"label upsert updates", func() error { return UpsertNumberLabel(ctx, x, "+27821234567", "Front desk") },
			func(t *testing.T, n Number) {
				if n.Label != "Front desk" {
					t.Errorf("label = %q", n.Label)
				}
			}, "+27821234567"},
		{"label upsert creates", func() error { return UpsertNumberLabel(ctx, x, "+27831112222", "Sales") },
			func(t *testing.T, n Number) {
				if n.Label != "Sales" || n.RegisterRequestedAt != nil || n.CreatedAt.IsZero() {
					t.Errorf("row = %+v", n)
				}
			}, "+27831112222"},
		{"register requested creates", func() error {
			at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("x", 7200))
			return SetRegisterRequested(ctx, x, "+27844445555", &at)
		}, func(t *testing.T, n Number) {
			want := time.Date(2026, 1, 2, 1, 4, 5, 0, time.UTC)
			if n.RegisterRequestedAt == nil || !n.RegisterRequestedAt.Equal(want) {
				t.Errorf("requested_at = %v, want %v", n.RegisterRequestedAt, want)
			}
		}, "+27844445555"},
		{"register requested clears", func() error { return SetRegisterRequested(ctx, x, "+27844445555", nil) },
			func(t *testing.T, n Number) {
				if n.RegisterRequestedAt != nil {
					t.Errorf("requested_at = %v, want nil", n.RegisterRequestedAt)
				}
			}, "+27844445555"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.do(); err != nil {
				t.Fatal(err)
			}
			n, err := GetNumber(ctx, x, tc.num)
			if err != nil {
				t.Fatal(err)
			}
			tc.check(t, n)
		})
	}

	all, err := ListNumbers(ctx, x)
	if err != nil || len(all) != 3 || all[0].Number != "+27821234567" {
		t.Fatalf("list = %+v, %v", all, err)
	}
}

func TestAudit(t *testing.T) {
	ctx := context.Background()
	x := open(t)

	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, e := range []AuditEntry{
		{At: old, Actor: "a", Action: "register", Number: "+1", Result: "ok"},
		{Actor: "b", Action: "verify", Number: "+1", Detail: "d", Result: "error"},
		{Actor: "c", Action: "unregister", Result: "ok"},
	} {
		if err := InsertAudit(ctx, x, e); err != nil {
			t.Fatal(err)
		}
	}

	got, err := ListAudit(ctx, x, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Actor != "c" || got[1].Actor != "b" {
		t.Fatalf("newest-first / limit wrong: %+v", got)
	}
	if time.Since(got[0].At) > time.Minute || got[0].At.Location() != time.UTC {
		t.Errorf("zero At not defaulted to now UTC: %v", got[0].At)
	}
	all, _ := ListAudit(ctx, x, 10)
	if len(all) != 3 || !all[2].At.Equal(old) {
		t.Fatalf("explicit At lost: %+v", all)
	}
	if _, err := ListAudit(ctx, x, 0); err == nil {
		t.Fatal("limit 0 accepted")
	}
}

func TestUsersAndSessions(t *testing.T) {
	ctx := context.Background()
	x := open(t)
	now := time.Now()

	if err := InsertLocalUser(ctx, x, "admin", "Admin", "admin", "hash"); err != nil {
		t.Fatal(err)
	}
	if err := InsertLocalUser(ctx, x, "admin", "x", "user", "hash"); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate user = %v", err)
	}
	if err := InsertLocalUser(ctx, x, "bad", "x", "root", "hash"); err == nil {
		t.Fatal("invalid role accepted")
	}

	ad, err := UpsertADUser(ctx, x, "jo", "Jo", "user")
	if err != nil || ad.PasswordHash != nil {
		t.Fatalf("ad upsert: %+v, %v", ad, err)
	}
	ad, err = UpsertADUser(ctx, x, "jo", "Jo Bloggs", "admin")
	if err != nil || ad.Name != "Jo Bloggs" || ad.Role != "admin" {
		t.Fatalf("ad refresh: %+v, %v", ad, err)
	}
	if _, err := UpsertADUser(ctx, x, "admin", "Evil", "admin"); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("AD must not take over a local account, got %v", err)
	}
	if u, _ := GetUserByUsername(ctx, x, "admin"); u.Name != "Admin" {
		t.Fatalf("local account modified: %+v", u)
	}
	if n, _ := CountUsers(ctx, x); n != 2 {
		t.Fatalf("count = %d", n)
	}

	u, _ := GetUserByUsername(ctx, x, "admin")
	for _, s := range []Session{
		{"live", u.ID, now.Add(time.Hour)},
		{"expired", u.ID, now.Add(-time.Hour)},
		{"revoked", u.ID, now.Add(time.Hour)},
	} {
		if err := InsertSession(ctx, x, s); err != nil {
			t.Fatal(err)
		}
	}
	if err := RevokeSession(ctx, x, "revoked", now); err != nil {
		t.Fatal(err)
	}
	for id, wantOK := range map[string]bool{"live": true, "expired": false, "revoked": false, "unknown": false} {
		_, err := GetSessionUser(ctx, x, id, now)
		if wantOK != (err == nil) || (!wantOK && !errors.Is(err, ErrNotFound)) {
			t.Errorf("session %q: err = %v, want ok=%v", id, err, wantOK)
		}
	}
	if err := DeleteExpiredSessions(ctx, x, now); err != nil {
		t.Fatal(err)
	}
	var left int
	x.Get(&left, `SELECT COUNT(*) FROM sessions`)
	if left != 2 {
		t.Fatalf("sessions left = %d, want 2", left)
	}
}
