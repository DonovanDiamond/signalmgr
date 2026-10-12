package web

import (
	"errors"
	"io/fs"
	"testing"
)

func TestFSFallbackWhenNotBuilt(t *testing.T) {
	got, err := FS()
	if _, statErr := fs.Stat(buildFS, "build/index.html"); statErr == nil {
		if err != nil {
			t.Fatalf("built UI present, got error %v", err)
		}
		return
	}
	if !errors.Is(err, errNotBuilt) {
		t.Fatalf("err = %v, want errNotBuilt", err)
	}
	if _, err := fs.Stat(got, "index.html"); err != nil {
		t.Fatalf("fallback has no index.html: %v", err)
	}
}
