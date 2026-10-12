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

var errNotBuilt = errors.New("web build not found: run `make web` before building the binary")

// FS returns the embedded UI rooted at build/, or an error if it holds no
// index.html.
func FS() (fs.FS, error) {
	sub, err := fs.Sub(buildFS, "build")
	if err != nil {
		return nil, err
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, errNotBuilt
	}
	return sub, nil
}
