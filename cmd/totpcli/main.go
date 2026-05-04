package main

import (
	"os"

	"github.com/trezorg/totpcli/internal/cli"
)

func main() {
	os.Exit(cli.App{}.Run(os.Args[1:]))
}
