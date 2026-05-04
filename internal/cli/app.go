package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
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

	var totpPath string
	var format string
	var initialSearch string
	exitCode := 0

	cmd := &cobra.Command{
		Use:   "totpcli",
		Short: "Browse configured TOTP providers and show generated codes",
		Long: `totpcli loads configured TOTP providers, filters them in a terminal UI,
and shows generated TOTP codes for the selected provider.

Environment variables:
  TOTPCLI_TOTP_PATH  provider file path fallback for --totp
  TOTPCLI_FORMAT     provider file format fallback for --format
  TOTPCLI_SEARCH     initial provider search fallback for --search`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			effectivePath := effectiveString(cmd, "totp", totpPath, "TOTPCLI_TOTP_PATH")
			effectiveFormat := effectiveString(cmd, "format", format, "TOTPCLI_FORMAT")
			effectiveSearch := effectiveString(cmd, "search", initialSearch, "TOTPCLI_SEARCH")

			resolved, err := provider.ResolveOptions(provider.Options{
				Path:   effectivePath,
				Format: effectiveFormat,
				Home:   a.Home,
			})
			if err != nil {
				exitCode = 2
				return err
			}

			providers, err := provider.Load(resolved.Path, resolved.Format)
			if err != nil {
				exitCode = 1
				return err
			}

			model := tui.NewModel(providers, search.Lexical{}, effectiveSearch, totpcode.Generator{}, time.Now)
			runTUI := a.RunTUI
			if runTUI == nil {
				runTUI = runBubbleTea(stdout)
			}
			if err := runTUI(model); err != nil {
				exitCode = 1
				return err
			}
			return nil
		},
	}

	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.Flags().StringVar(&totpPath, "totp", "", "TOTP providers file path (env TOTPCLI_TOTP_PATH)")
	cmd.Flags().StringVar(&format, "format", "", "TOTP providers file format: json, yaml, or toml (env TOTPCLI_FORMAT)")
	cmd.Flags().StringVar(&initialSearch, "search", "", "Initial provider search query (env TOTPCLI_SEARCH)")

	if err := cmd.Execute(); err != nil {
		if exitCode == 0 {
			exitCode = 2
		}
		fmt.Fprintf(stderr, "error: %v\n", err)
		return exitCode
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

func effectiveString(cmd *cobra.Command, flagName string, flagValue string, envName string) string {
	if cmd.Flags().Changed(flagName) {
		return flagValue
	}
	if value, ok := os.LookupEnv(envName); ok && value != "" {
		return value
	}
	return flagValue
}
