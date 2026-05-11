package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
	"github.com/trezorg/totpcli/internal/importer"
	"github.com/trezorg/totpcli/internal/provider"
	"github.com/trezorg/totpcli/internal/search/digram"
	"github.com/trezorg/totpcli/internal/search/lexical"
	"github.com/trezorg/totpcli/internal/storage"
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
	NewRepository   func(string) (providerRepository, error)
	NewSearchEngine func(string, []provider.Provider) (tui.SearchEngine, error)
	RunTUI          func(tui.Model) error
	Now             func() time.Time
	ImportQRDecoder importer.QRDecoder
}

type totpOptions struct {
	path          string
	initialSearch string
	searchEngine  string
	noTUI         bool
}

type providerRepository interface {
	Init() error
	ListProviders() ([]provider.Provider, error)
	UpsertProviders([]provider.RawProvider) (storage.ImportResult, error)
	Close() error
}

func (a App) Run(args []string) int {
	stdout := writerOrDefault(a.Stdout, os.Stdout)
	stderr := writerOrDefault(a.Stderr, os.Stderr)

	totpOpts := totpOptions{searchEngine: searchEngineDigram}
	importOpts := struct {
		path string
	}{}
	importFileOpts := struct {
		path   string
		format string
	}{}
	exitCode := 0
	runTOTP := func(cmd *cobra.Command, _ []string) error {
		code, err := a.runTOTP(cmd, stdout, totpOpts)
		exitCode = code
		return err
	}

	cmd := &cobra.Command{
		Use:   "totpcli",
		Short: "Browse configured TOTP providers and show generated codes",
		Long: `totpcli loads configured TOTP providers, filters them in a terminal UI,
and shows generated TOTP codes for the selected provider.

Environment variables:
  TOTPCLI_TOTP_PATH       provider file path fallback for --totp
  TOTPCLI_SEARCH          initial provider search fallback for --search
  TOTPCLI_SEARCH_ENGINE   provider search engine fallback for --search-engine
  TOTPCLI_NO_TUI          non-interactive output fallback for --no-tui`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runTOTP,
	}
	addTOTPFlags(cmd, &totpOpts)

	totpCmd := &cobra.Command{
		Use:   "totp",
		Short: "Browse configured TOTP providers and show generated codes",
		Long:  "Run the TOTP display workflow with the same flags and environment variable fallbacks as root invocation.",
		RunE:  runTOTP,
	}
	addTOTPFlags(totpCmd, &totpOpts)

	importCmd := &cobra.Command{
		Use:   "import <qr-image>",
		Short: "Import Google Authenticator export QR providers into SQLite storage",
		Long: `Import Google Authenticator export QR providers into SQLite storage.

When --totp is omitted, the database path is derived from the QR image filename
by replacing its extension with .sqlite. Duplicate providers are overridden.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			effectivePath := effectiveString(cmd, "totp", importOpts.path, "TOTPCLI_TOTP_PATH")
			dbPath, err := storage.ResolveDatabasePath(effectivePath, args[0], a.Home)
			if err != nil {
				exitCode = 2
				return err
			}
			repo, err := a.openRepository(dbPath)
			if err != nil {
				exitCode = 1
				return err
			}
			defer func() {
				_ = repo.Close()
			}()

			importService := importer.Service{QRDecoder: a.ImportQRDecoder}
			result, err := importService.Import(args[0], repo)
			if err != nil {
				exitCode = 1
				return err
			}
			return printImportResult(stdout, result)
		},
	}
	importCmd.Flags().StringVar(&importOpts.path, "totp", "", "Output SQLite provider database path (env TOTPCLI_TOTP_PATH)")

	importFileCmd := &cobra.Command{
		Use:   "import-file <provider-file>...",
		Short: "Import provider files into SQLite storage",
		Long: `Import JSON, YAML, or TOML provider files into SQLite storage.

When --format is omitted, each input file format is detected from its extension.
When --totp is omitted, the database path is derived from the first input file by
replacing its extension with .sqlite. Duplicate providers are overridden.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			effectivePath := effectiveString(cmd, "totp", importFileOpts.path, "TOTPCLI_TOTP_PATH")
			dbPath, err := storage.ResolveDatabasePath(effectivePath, args[0], a.Home)
			if err != nil {
				exitCode = 1
				return err
			}
			repo, err := a.openRepository(dbPath)
			if err != nil {
				exitCode = 1
				return err
			}
			defer func() {
				_ = repo.Close()
			}()

			result, err := (importer.FileService{}).Import(args, importFileOpts.format, a.Home, repo)
			if err != nil {
				exitCode = 1
				return err
			}
			return printImportResult(stdout, result)
		},
	}
	importFileCmd.Flags().StringVar(&importFileOpts.path, "totp", "", "Output SQLite provider database path (env TOTPCLI_TOTP_PATH)")
	importFileCmd.Flags().StringVar(&importFileOpts.format, "format", "", "Provider file format: json, yaml, or toml")

	cmd.AddCommand(totpCmd, importCmd, importFileCmd)
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	if err := cmd.Execute(); err != nil {
		if exitCode == 0 {
			exitCode = 2
		}
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return exitCode
	}
	return 0
}

func (a App) runTOTP(cmd *cobra.Command, stdout io.Writer, opts totpOptions) (int, error) {
	effectivePath := effectiveString(cmd, "totp", opts.path, "TOTPCLI_TOTP_PATH")
	effectiveSearch := effectiveString(cmd, "search", opts.initialSearch, "TOTPCLI_SEARCH")
	effectiveSearchEngine := effectiveString(cmd, "search-engine", opts.searchEngine, "TOTPCLI_SEARCH_ENGINE")
	effectiveNoTUI, err := effectiveBool(cmd, "no-tui", opts.noTUI, "TOTPCLI_NO_TUI")
	if err != nil {
		return 2, err
	}
	effectiveSearchEngine, err = parseSearchEngineName(effectiveSearchEngine)
	if err != nil {
		return 2, err
	}

	dbPath, err := storage.ResolveDatabasePath(effectivePath, filepath.Join("~", "totp.sqlite"), a.Home)
	if err != nil {
		return 2, err
	}
	repo, err := a.openRepository(dbPath)
	if err != nil {
		return 1, err
	}
	defer func() {
		_ = repo.Close()
	}()
	providers, err := repo.ListProviders()
	if err != nil {
		return 1, err
	}

	newSearchEngine := a.NewSearchEngine
	if newSearchEngine == nil {
		newSearchEngine = newSearchEngineForName
	}
	engine, err := newSearchEngine(effectiveSearchEngine, providers)
	if err != nil {
		return 2, err
	}

	now := a.Now
	if now == nil {
		now = time.Now
	}
	if effectiveNoTUI {
		if err := printTokens(stdout, engine.Search(effectiveSearch), totpcode.Generator{}, now()); err != nil {
			return 1, err
		}
		return 0, nil
	}

	model := tui.NewModel(providers, engine, effectiveSearch, totpcode.Generator{}, now)
	runTUI := a.RunTUI
	if runTUI == nil {
		runTUI = runBubbleTea(stdout)
	}
	if err := runTUI(model); err != nil {
		return 1, err
	}
	return 0, nil
}

func addTOTPFlags(cmd *cobra.Command, opts *totpOptions) {
	cmd.Flags().StringVar(&opts.path, "totp", "", "TOTP providers SQLite database path (env TOTPCLI_TOTP_PATH)")
	cmd.Flags().StringVar(&opts.initialSearch, "search", "", "Initial provider search query (env TOTPCLI_SEARCH)")
	cmd.Flags().StringVar(&opts.searchEngine, "search-engine", searchEngineDigram, "Provider search engine: digram or lexical (env TOTPCLI_SEARCH_ENGINE)")
	cmd.Flags().BoolVar(&opts.noTUI, "no-tui", false, "Print current tokens and exit without starting the TUI (env TOTPCLI_NO_TUI)")
}

func (a App) openRepository(path string) (providerRepository, error) {
	newRepository := a.NewRepository
	if newRepository == nil {
		newRepository = func(path string) (providerRepository, error) {
			return storage.OpenSQLite(path)
		}
	}
	repo, err := newRepository(path)
	if err != nil {
		return nil, err
	}
	if err := repo.Init(); err != nil {
		_ = repo.Close()
		return nil, err
	}
	return repo, nil
}

func printImportResult(output io.Writer, result storage.ImportResult) error {
	_, err := fmt.Fprintf(output, "imported providers into %s: imported=%d added=%d overridden=%d final=%d\n",
		result.OutputPath, result.Imported, result.Added, result.Overridden, result.Final)
	return err
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

func effectiveBool(cmd *cobra.Command, flagName string, flagValue bool, envName string) (bool, error) {
	if cmd.Flags().Changed(flagName) {
		return flagValue, nil
	}
	if value, ok := os.LookupEnv(envName); ok && value != "" {
		return parseBoolEnv(envName, value)
	}
	return flagValue, nil
}

func parseBoolEnv(envName string, value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "t", "true", "y", "yes", "on":
		return true, nil
	case "0", "f", "false", "n", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("unsupported boolean %q for %s: expected true or false", value, envName)
	}
}

func printTokens(output io.Writer, providers []provider.Provider, generator totpcode.Generator, now time.Time) error {
	if len(providers) == 0 {
		return nil
	}

	rows := make([][]string, 0, len(providers))
	for _, candidate := range providers {
		result, err := generator.Generate(candidate, now)
		if err != nil {
			return err
		}
		rows = append(rows, []string{candidate.DisplayLabel(), result.Code, fmt.Sprintf("%ds", result.Remaining)})
	}

	tokenTable := table.New().
		Border(lipgloss.ASCIIBorder()).
		Headers("Provider", "Token", "Remaining").
		Rows(rows...)
	if _, err := fmt.Fprintln(output, tokenTable.Render()); err != nil {
		return fmt.Errorf("write token output: %w", err)
	}
	return nil
}
