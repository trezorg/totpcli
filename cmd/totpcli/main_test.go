package main

import (
	"testing"

	"github.com/trezorg/totpcli/internal/cli"
)

func TestMainHelpExitsZero(t *testing.T) {
	t.Parallel()

	code := cli.App{}.Run([]string{"--help"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}
