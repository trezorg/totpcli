package totpcode

import (
	"fmt"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/trezorg/totpcli/internal/provider"
	"github.com/trezorg/totpcli/internal/totpsecret"
)

const Period = 30

type Result struct {
	Code      string
	Remaining int
}

type Generator struct{}

func (Generator) Generate(selected provider.Provider, now time.Time) (Result, error) {
	return Generate(selected, now)
}

func Generate(selected provider.Provider, now time.Time) (Result, error) {
	secret, err := totpsecret.CanonicalBase32(selected.TOTPSecret)
	if err != nil {
		return Result{}, fmt.Errorf("generate TOTP for %s: %w", selected.DisplayLabel(), err)
	}
	code, err := totp.GenerateCodeCustom(secret, now, totp.ValidateOpts{
		Period:    Period,
		Digits:    digits(selected.Digits),
		Algorithm: algorithm(selected.Algorithm),
	})
	if err != nil {
		return Result{}, fmt.Errorf("generate TOTP for %s: %w", selected.DisplayLabel(), err)
	}

	return Result{
		Code:      code,
		Remaining: RemainingSeconds(now),
	}, nil
}

func RemainingSeconds(now time.Time) int {
	// At an exact boundary, the just-generated code has the full next period remaining.
	elapsed := int(now.Unix() % Period)
	if elapsed == 0 {
		return Period
	}
	return Period - elapsed
}

func digits(value int) otp.Digits {
	if value == 8 {
		return otp.DigitsEight
	}
	return otp.DigitsSix
}

func algorithm(value provider.Algorithm) otp.Algorithm {
	switch value {
	case provider.AlgorithmSHA256:
		return otp.AlgorithmSHA256
	case provider.AlgorithmSHA512:
		return otp.AlgorithmSHA512
	default:
		return otp.AlgorithmSHA1
	}
}
