package totpsecret

import (
	"encoding/base32"
	"errors"
	"fmt"
	"strings"

	"github.com/trezorg/totpcli/internal/provider"
)

func CanonicalBase32(secret string) (string, error) {
	decoded, err := DecodeBase32(secret)
	if err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(decoded), nil
}

// ErrSecretRequired indicates the TOTP secret is empty.
var ErrSecretRequired = errors.New("totp secret is required")

func DecodeBase32(secret string) ([]byte, error) {
	normalized := strings.ToUpper(strings.TrimSpace(secret))
	if normalized == "" {
		return nil, ErrSecretRequired
	}
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.TrimRight(normalized, "="))
	if err != nil {
		return nil, fmt.Errorf("invalid totp secret: %w", err)
	}
	return decoded, nil
}

func ValidateMinimumBytes(secret string, algorithm provider.Algorithm) error {
	decoded, err := DecodeBase32(secret)
	if err != nil {
		return err
	}
	minBytes := minBytes(algorithm)
	if len(decoded) < minBytes {
		return fmt.Errorf("invalid totp secret: decoded key is %d bytes, want at least %d bytes for %s", len(decoded), minBytes, algorithm)
	}
	return nil
}

func minBytes(algorithm provider.Algorithm) int {
	switch algorithm {
	case provider.AlgorithmSHA256:
		return 32
	case provider.AlgorithmSHA512:
		return 64
	default:
		return 20
	}
}
