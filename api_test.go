package signalmgr

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// Points API_URL at a test server for the duration of the test.
func serve(t *testing.T, h http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(h)
	old := API_URL
	API_URL = srv.URL
	t.Cleanup(func() {
		API_URL = old
		srv.Close()
	})
}

func TestEscapedPathSegmentArrivesIntact(t *testing.T) {
	var got string
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.EscapedPath()
		w.Write([]byte("{}"))
	})
	if _, err := (&Account{Number: "+27123"}).GetGroup("ab/c+d==", GetGroupOptions{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "ab%2Fc+d==") || strings.Contains(got, "/ab/c") {
		t.Fatalf("escaped path = %q, want it to contain ab%%2Fc+d==", got)
	}
}

func TestDiscardedBodyIgnoresPlainText(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	if err := (&Account{Number: "+27123"}).DeleteGroup("g"); err != nil {
		t.Fatal(err)
	}
}

func TestErrorStatusYieldsAPIError(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"x","account":"+1","challenge_tokens":["t"]}`))
	})
	err := (&Account{Number: "+27123"}).DeleteGroup("g")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiErr.Status != 400 || !reflect.DeepEqual(apiErr.ChallengeTokens, []string{"t"}) {
		t.Fatalf("got status %d tokens %v", apiErr.Status, apiErr.ChallengeTokens)
	}
}

func TestGetAccountsBothShapes(t *testing.T) {
	const aci = "5f1c2d3e-4b5a-4c6d-8e7f-90a1b2c3d4e5"
	cases := []struct {
		name, body string
		want       []AccountInfo
		wantIDs    []string
	}{
		{"strings", `["+27111","+27222"]`, []AccountInfo{{Number: "+27111"}, {Number: "+27222"}}, []string{"+27111", "+27222"}},
		{"objects", `[{"number":"+27111","aci":"` + aci + `"},{"number":"","aci":"` + aci + `"},{"number":null,"aci":null}]`,
			[]AccountInfo{{Number: "+27111", ACI: aci}, {ACI: aci}, {}}, []string{"+27111", aci, ""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			serve(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(c.body)) })
			got, err := GetAccounts()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
			for i, a := range got {
				if a.ID() != c.wantIDs[i] {
					t.Fatalf("ID() = %q, want %q", a.ID(), c.wantIDs[i])
				}
			}
		})
	}
}

func TestACIPathSegment(t *testing.T) {
	const aci = "5f1c2d3e-4b5a-4c6d-8e7f-90a1b2c3d4e5"
	var got string
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.EscapedPath()
		w.Write([]byte("[]"))
	})
	if _, err := (&Account{Number: aci}).GetDevices(); err != nil {
		t.Fatal(err)
	}
	if got != "/v1/devices/"+aci {
		t.Fatalf("path = %q", got)
	}
}
