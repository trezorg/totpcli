package totpcode

import (
	"strconv"
	"testing"
	"time"

	"github.com/trezorg/totpcli/internal/provider"
)

func TestGenerateReturnsSixDigitCode(t *testing.T) {
	t.Parallel()

	result, err := Generate(provider.Provider{
		Issuer:     "GitHub",
		Name:       "trezorg",
		Algorithm:  provider.AlgorithmSHA1,
		Digits:     6,
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	}, time.Unix(59, 0))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if len(result.Code) != 6 {
		t.Fatalf("len(Code) = %d, want 6", len(result.Code))
	}
	if _, err := strconv.Atoi(result.Code); err != nil {
		t.Fatalf("Code = %q, want numeric code", result.Code)
	}
	if result.Remaining != 1 {
		t.Fatalf("Remaining = %d, want 1", result.Remaining)
	}
}

func TestGenerateSupportsSHA256SHA512AndEightDigits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		algorithm provider.Algorithm
		digits    int
	}{
		{name: "sha256", algorithm: provider.AlgorithmSHA256, digits: 6},
		{name: "sha512 eight", algorithm: provider.AlgorithmSHA512, digits: 8},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := Generate(provider.Provider{
				Issuer:     "GitHub",
				Algorithm:  tt.algorithm,
				Digits:     tt.digits,
				TOTPSecret: "JBSWY3DPEHPK3PXP",
			}, time.Unix(30, 0))
			if err != nil {
				t.Fatalf("Generate returned error: %v", err)
			}
			if len(result.Code) != tt.digits {
				t.Fatalf("len(Code) = %d, want %d", len(result.Code), tt.digits)
			}
		})
	}
}

func TestGenerateRejectsInvalidSecret(t *testing.T) {
	t.Parallel()

	_, err := Generate(provider.Provider{
		Issuer:     "GitHub",
		Algorithm:  provider.AlgorithmSHA1,
		Digits:     6,
		TOTPSecret: "not base32!",
	}, time.Unix(0, 0))
	if err == nil {
		t.Fatal("Generate returned nil error for invalid secret")
	}
}

func TestRemainingSeconds(t *testing.T) {
	t.Parallel()

	tests := map[int64]int{
		0:  30,
		1:  29,
		29: 1,
		30: 30,
		44: 16,
	}

	for epoch, want := range tests {
		epoch := epoch
		want := want
		t.Run(strconv.FormatInt(epoch, 10), func(t *testing.T) {
			t.Parallel()

			if got := RemainingSeconds(time.Unix(epoch, 0)); got != want {
				t.Fatalf("RemainingSeconds = %d, want %d", got, want)
			}
		})
	}
}
