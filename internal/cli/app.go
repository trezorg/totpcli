package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/trezorg/totpcli/internal/provider"
	"github.com/trezorg/totpcli/internal/search"
	"github.com/trezorg/totpcli/internal/totpcode"
	"github.com/trezorg/totpcli/internal/tui"
)

type App struct {
	Home   string
	Stdout io.Writer
	Stderr io.Writer
	RunTUI func(tui.Model) error
}

func (a App) Run(args []string) int {
	stdout := writerOrDefault(a.Stdout, os.Stdout)
	stderr := writerOrDefault(a.Stderr, os.Stderr)

	flags := flag.NewFlagSet("totpcli", flag.ContinueOnError)
	flags.SetOutput(stderr)
	totpPath := flags.String("totp", "", "TOTP providers file path")
	format := flags.String("format", "", "TOTP providers file format: json, yaml, or toml")
	initialSearch := flags.String("search", "", "Initial provider search query")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	resolved, err := provider.ResolveOptions(provider.Options{
		Path:   *totpPath,
		Format: *format,
		Home:   a.Home,
	})
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	providers, err := provider.Load(resolved.Path, resolved.Format)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	model := tui.NewModel(providers, search.Lexical{}, *initialSearch, totpcode.Generator{}, time.Now)
	runTUI := a.RunTUI
	if runTUI == nil {
		runTUI = runBubbleTea(stdout)
	}
	if err := runTUI(model); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func runBubbleTea(output io.Writer) func(tui.Model) error {
	return func(model tui.Model) error {
		program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithOutput(output))
		_, err := program.Run()
		return err
	}
}

func writerOrDefault(writer io.Writer, fallback io.Writer) io.Writer {
	if writer != nil {
		return writer
	}
	return fallback
}
