// Package web embeds the built SvelteKit static output for serving by the API.
package web

import (
	"embed"
	"errors"
	"io/fs"
)

// buildFS embeds the whole build directory, dotfiles included, so the package
// still compiles before the UI has been built.
//
//go:embed all:build
var buildFS embed.FS

// notBuiltFS holds the placeholder page served when buildFS has no UI.
//
//go:embed notbuilt
var notBuiltFS embed.FS

var errNotBuilt = errors.New("web build not found: run `make web` before building the binary")

// FS returns the embedded UI rooted at build/. If build/ holds no index.html it
// returns a placeholder page FS together with errNotBuilt, so the caller can
// warn and still serve something.
func FS() (fs.FS, error) {
	sub, err := fs.Sub(buildFS, "build")
	if err != nil {
		return nil, err
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		fallback, err := fs.Sub(notBuiltFS, "notbuilt")
		if err != nil {
			return nil, err
		}
		return fallback, errNotBuilt
	}
	return sub, nil
}
