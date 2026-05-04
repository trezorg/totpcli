package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/trezorg/totpcli/internal/provider"
	"github.com/trezorg/totpcli/internal/search/digram"
	"github.com/trezorg/totpcli/internal/search/lexical"
	"github.com/trezorg/totpcli/internal/totpcode"
	"github.com/trezorg/totpcli/internal/tui"
)

const (
	searchEngineDigram  = "digram"
	searchEngineLexical = "lexical"
)

type App struct {
	Home            string
	Stdout          io.Writer
	Stderr          io.Writer
	NewSearchEngine func(string, []provider.Provider) (tui.SearchEngine, error)
	RunTUI          func(tui.Model) error
}

func (a App) Run(args []string) int {
	stdout := writerOrDefault(a.Stdout, os.Stdout)
	stderr := writerOrDefault(a.Stderr, os.Stderr)

	var totpPath string
	var format string
	var initialSearch string
	searchEngine := searchEngineDigram
	exitCode := 0

	cmd := &cobra.Command{
		Use:   "totpcli",
		Short: "Browse configured TOTP providers and show generated codes",
		Long: `totpcli loads configured TOTP providers, filters them in a terminal UI,
and shows generated TOTP codes for the selected provider.

Environment variables:
  TOTPCLI_TOTP_PATH       provider file path fallback for --totp
  TOTPCLI_FORMAT          provider file format fallback for --format
  TOTPCLI_SEARCH          initial provider search fallback for --search
  TOTPCLI_SEARCH_ENGINE   provider search engine fallback for --search-engine`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			effectivePath := effectiveString(cmd, "totp", totpPath, "TOTPCLI_TOTP_PATH")
			effectiveFormat := effectiveString(cmd, "format", format, "TOTPCLI_FORMAT")
			effectiveSearch := effectiveString(cmd, "search", initialSearch, "TOTPCLI_SEARCH")
			effectiveSearchEngine := effectiveString(cmd, "search-engine", searchEngine, "TOTPCLI_SEARCH_ENGINE")
			effectiveSearchEngine, err := parseSearchEngineName(effectiveSearchEngine)
			if err != nil {
				exitCode = 2
				return err
			}

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

			newSearchEngine := a.NewSearchEngine
			if newSearchEngine == nil {
				newSearchEngine = newSearchEngineForName
			}
			engine, err := newSearchEngine(effectiveSearchEngine, providers)
			if err != nil {
				exitCode = 2
				return err
			}

			model := tui.NewModel(providers, engine, effectiveSearch, totpcode.Generator{}, time.Now)
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
	cmd.Flags().StringVar(&searchEngine, "search-engine", searchEngineDigram, "Provider search engine: digram or lexical (env TOTPCLI_SEARCH_ENGINE)")

	if err := cmd.Execute(); err != nil {
		if exitCode == 0 {
			exitCode = 2
		}
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return exitCode
	}
	return 0
}

func newSearchEngineForName(name string, providers []provider.Provider) (tui.SearchEngine, error) {
	name, err := parseSearchEngineName(name)
	if err != nil {
		return nil, err
	}

	switch name {
	case searchEngineDigram:
		return digram.New(providers), nil
	case searchEngineLexical:
		return lexical.New(providers), nil
	default:
		return nil, fmt.Errorf("unsupported search engine %q: expected digram or lexical", name)
	}
}

func parseSearchEngineName(name string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", searchEngineDigram:
		return searchEngineDigram, nil
	case searchEngineLexical:
		return searchEngineLexical, nil
	default:
		return "", fmt.Errorf("unsupported search engine %q: expected digram or lexical", name)
	}
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
