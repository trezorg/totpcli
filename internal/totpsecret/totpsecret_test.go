package totpsecret

import (
	"testing"

	"github.com/trezorg/totpcli/internal/provider"
)

func TestCanonicalBase32NormalizesCaseWhitespaceAndPadding(t *testing.T) {
	t.Parallel()

	got, err := CanonicalBase32(" jbswy3dpehpk3pxp==== ")
	if err != nil {
		t.Fatalf("CanonicalBase32 returned error: %v", err)
	}
	if got != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("CanonicalBase32 = %q, want JBSWY3DPEHPK3PXP", got)
	}
}

func TestValidateMinimumBytesRejectsEmptySecretClearly(t *testing.T) {
	t.Parallel()

	err := ValidateMinimumBytes("", provider.AlgorithmSHA1)
	if err == nil {
		t.Fatal("ValidateMinimumBytes returned nil error for empty secret")
	}
	if err.Error() != "totp secret is required" {
		t.Fatalf("error = %q, want totp secret is required", err.Error())
	}
}

func TestValidateMinimumBytesRejectsShortDecodedKey(t *testing.T) {
	t.Parallel()

	err := ValidateMinimumBytes("JBSWY3DPEHPK3PXP", provider.AlgorithmSHA1)
	if err == nil {
		t.Fatal("ValidateMinimumBytes returned nil error for short key")
	}
}
