package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

// KeyPair is the Ed25519 pair access tokens are signed and verified with.
type KeyPair struct {
	Private ed25519.PrivateKey
	Public  ed25519.PublicKey
}

// LoadOrGenerateKeys loads the Ed25519 pair from privatePath/publicPath, or
// generates and writes a new pair if neither file exists yet. If exactly one
// of the two is present, that's refused rather than silently regenerated —
// regenerating either half of a pair already in use would invalidate every
// live access token without warning.
func LoadOrGenerateKeys(privatePath, publicPath string) (*KeyPair, error) {
	privExists, err := fileExists(privatePath)
	if err != nil {
		return nil, fmt.Errorf("loading keys: %w", err)
	}
	pubExists, err := fileExists(publicPath)
	if err != nil {
		return nil, fmt.Errorf("loading keys: %w", err)
	}

	switch {
	case privExists && pubExists:
		return loadKeys(privatePath, publicPath)
	case !privExists && !pubExists:
		return generateKeys(privatePath, publicPath)
	default:
		return nil, fmt.Errorf(
			"loading keys: %s and %s must both exist or both be absent (found only one); "+
				"refusing to regenerate half of a key pair that may already be signing live tokens",
			privatePath, publicPath)
	}
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func loadKeys(privatePath, publicPath string) (*KeyPair, error) {
	privPEM, err := os.ReadFile(privatePath)
	if err != nil {
		return nil, fmt.Errorf("loading private key: %w", err)
	}
	block, _ := pem.Decode(privPEM)
	if block == nil {
		return nil, fmt.Errorf("loading private key %s: not a PEM file", privatePath)
	}
	privAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("loading private key %s: %w", privatePath, err)
	}
	priv, ok := privAny.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("loading private key %s: not an Ed25519 key", privatePath)
	}

	pubPEM, err := os.ReadFile(publicPath)
	if err != nil {
		return nil, fmt.Errorf("loading public key: %w", err)
	}
	block, _ = pem.Decode(pubPEM)
	if block == nil {
		return nil, fmt.Errorf("loading public key %s: not a PEM file", publicPath)
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("loading public key %s: %w", publicPath, err)
	}
	pub, ok := pubAny.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("loading public key %s: not an Ed25519 key", publicPath)
	}

	return &KeyPair{Private: priv, Public: pub}, nil
}

func generateKeys(privatePath, publicPath string) (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating key pair: %w", err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("generating key pair: marshalling private key: %w", err)
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("generating key pair: marshalling public key: %w", err)
	}

	if err := writePEM(privatePath, "PRIVATE KEY", privBytes, 0o600); err != nil {
		return nil, fmt.Errorf("generating key pair: %w", err)
	}
	if err := writePEM(publicPath, "PUBLIC KEY", pubBytes, 0o644); err != nil {
		return nil, fmt.Errorf("generating key pair: %w", err)
	}

	return &KeyPair{Private: priv, Public: pub}, nil
}

// writePEM creates path's directory (0700, since a private key may live
// there) and writes der as a PEM block at the given file mode.
func writePEM(path, blockType string, der []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: blockType, Bytes: der}); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
