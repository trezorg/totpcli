package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"github.com/trezorg/totpcli/internal/importer"
	"github.com/trezorg/totpcli/internal/provider"
	"github.com/trezorg/totpcli/internal/search/digram"
	"github.com/trezorg/totpcli/internal/search/lexical"
	"github.com/trezorg/totpcli/internal/storage"
	"github.com/trezorg/totpcli/internal/totpcode"
	"github.com/trezorg/totpcli/internal/totpsecret"
	"github.com/trezorg/totpcli/internal/tui"
	"gopkg.in/yaml.v3"
)

const (
	searchEngineDigram  = "digram"
	searchEngineLexical = "lexical"
	redactedShowValue   = "[REDACTED]"
)

type showFormat string

const (
	showFormatTable showFormat = "table"
	showFormatJSON  showFormat = "json"
	showFormatYAML  showFormat = "yaml"
	showFormatTOML  showFormat = "toml"
)

func (f showFormat) String() string { return string(f) }

func (f *showFormat) Set(value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(showFormatTable):
		*f = showFormatTable
	case string(showFormatJSON):
		*f = showFormatJSON
	case string(showFormatYAML):
		*f = showFormatYAML
	case string(showFormatTOML):
		*f = showFormatTOML
	default:
		return fmt.Errorf("unsupported show format %q: expected table, json, yaml, or toml", value)
	}
	return nil
}

func (showFormat) Type() string { return "format" }

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

type addOptions struct {
	path       string
	issuer     string
	name       string
	totpSecret string
	secret     string
	algorithm  string
	digits     string
}

type deleteOptions struct {
	path       string
	issuer     string
	name       string
	totpSecret string
}

type providerRepository interface {
	Init() error
	ListProviders() ([]provider.Provider, error)
	InspectTables() ([]storage.TableSnapshot, error)
	InsertProvider(provider.RawProvider) (provider.Provider, error)
	DeleteProvider(storage.DeleteSelector) (provider.Provider, error)
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
	addOpts := addOptions{}
	deleteOpts := deleteOptions{}
	importFileOpts := struct {
		path   string
		format string
	}{}
	showOpts := struct {
		path          string
		format        showFormat
		revealSecrets bool
	}{format: showFormatTable}
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

When --totp is omitted, providers are imported into ~/totp.sqlite. Duplicate
providers are overridden.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			effectivePath := effectiveString(cmd, "totp", importOpts.path, "TOTPCLI_TOTP_PATH")
			dbPath, err := storage.ResolveDatabasePath(effectivePath, filepath.Join("~", "totp.sqlite"), a.Home)
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

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add one TOTP provider to SQLite storage",
		Long: `Add one TOTP provider to SQLite storage.

The command creates TOTP providers only. The type defaults to TOTP, algorithm
uses default SHA1, digits use default SIX, and duplicate providers are updated.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			effectivePath := effectiveString(cmd, "totp", addOpts.path, "TOTPCLI_TOTP_PATH")
			raw := provider.RawProvider{
				Secret:     addOpts.secret,
				Name:       addOpts.name,
				Issuer:     addOpts.issuer,
				Algorithm:  addOpts.algorithm,
				Digits:     addOpts.digits,
				Type:       "TOTP",
				TOTPSecret: addOpts.totpSecret,
			}
			if err := validateAddProvider(raw); err != nil {
				exitCode = 2
				return err
			}
			dbPath, err := storage.ResolveDatabasePath(effectivePath, filepath.Join("~", "totp.sqlite"), a.Home)
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
			added, err := repo.InsertProvider(raw)
			if err != nil {
				exitCode = 1
				return err
			}
			return printAddResult(stdout, dbPath, added)
		},
	}
	addCmd.Flags().StringVar(&addOpts.path, "totp", "", "SQLite provider database path (env TOTPCLI_TOTP_PATH)")
	addCmd.Flags().StringVar(&addOpts.issuer, "issuer", "", "Provider issuer")
	addCmd.Flags().StringVar(&addOpts.name, "name", "", "Provider account name")
	addCmd.Flags().StringVar(&addOpts.totpSecret, "totp-secret", "", "Provider TOTP secret")
	addCmd.Flags().StringVar(&addOpts.secret, "secret", "", "Provider secret metadata")
	addCmd.Flags().StringVar(&addOpts.algorithm, "algorithm", "", "Provider algorithm: SHA1, SHA256, or SHA512 (default SHA1)")
	addCmd.Flags().StringVar(&addOpts.digits, "digits", "", "Provider digits: SIX, EIGHT, 6, or 8 (default SIX)")

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete one TOTP provider from SQLite storage",
		Long: `Delete one TOTP provider from SQLite storage.

The command deletes only when the supplied issuer, name, and optional TOTP
secret selector matches exactly one stored provider.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			effectivePath := effectiveString(cmd, "totp", deleteOpts.path, "TOTPCLI_TOTP_PATH")
			selector := storage.DeleteSelector{
				Issuer:     deleteOpts.issuer,
				Name:       deleteOpts.name,
				TOTPSecret: deleteOpts.totpSecret,
			}
			if err := validateDeleteSelector(selector); err != nil {
				exitCode = 2
				return err
			}
			dbPath, err := storage.ResolveDatabasePath(effectivePath, filepath.Join("~", "totp.sqlite"), a.Home)
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
			deleted, err := repo.DeleteProvider(selector)
			if err != nil {
				exitCode = 1
				return deleteError(err)
			}
			return printDeleteResult(stdout, dbPath, deleted)
		},
	}
	deleteCmd.Flags().StringVar(&deleteOpts.path, "totp", "", "SQLite provider database path (env TOTPCLI_TOTP_PATH)")
	deleteCmd.Flags().StringVar(&deleteOpts.issuer, "issuer", "", "Provider issuer")
	deleteCmd.Flags().StringVar(&deleteOpts.name, "name", "", "Provider account name")
	deleteCmd.Flags().StringVar(&deleteOpts.totpSecret, "totp-secret", "", "Provider TOTP secret for disambiguation")

	importFileCmd := &cobra.Command{
		Use:   "import-file <provider-file>...",
		Short: "Import provider files into SQLite storage",
		Long: `Import JSON, YAML, or TOML provider files into SQLite storage.

When --format is omitted, each input file format is detected from its extension.
When --totp is omitted, providers are imported into ~/totp.sqlite. Duplicate
providers are overridden.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			effectivePath := effectiveString(cmd, "totp", importFileOpts.path, "TOTPCLI_TOTP_PATH")
			dbPath, err := storage.ResolveDatabasePath(effectivePath, filepath.Join("~", "totp.sqlite"), a.Home)
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

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show SQLite provider database tables",
		Long: `Show every non-internal SQLite table in the resolved provider database.

When --totp is omitted, show reads the default provider database path.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			effectivePath := effectiveString(cmd, "totp", showOpts.path, "TOTPCLI_TOTP_PATH")
			dbPath, err := storage.ResolveDatabasePath(effectivePath, filepath.Join("~", "totp.sqlite"), a.Home)
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
			tables, err := repo.InspectTables()
			if err != nil {
				exitCode = 1
				return err
			}
			if !showOpts.revealSecrets {
				tables = redactShowSecrets(tables)
			}
			if err := renderShowTables(stdout, tables, showOpts.format); err != nil {
				exitCode = 1
				return err
			}
			return nil
		},
	}
	showCmd.Flags().StringVar(&showOpts.path, "totp", "", "SQLite provider database path (env TOTPCLI_TOTP_PATH)")
	showCmd.Flags().Var(&showOpts.format, "format", "Show output format: table, json, yaml, or toml")
	showCmd.Flags().BoolVar(&showOpts.revealSecrets, "reveal-secrets", false, "Reveal secret values in show output")

	cmd.AddCommand(totpCmd, addCmd, deleteCmd, importCmd, importFileCmd, showCmd)
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

func validateAddProvider(raw provider.RawProvider) error {
	if strings.TrimSpace(raw.TOTPSecret) == "" {
		return errors.New("totp secret is required")
	}
	normalized, err := provider.Normalize(raw)
	if err != nil {
		return err
	}
	if err := totpsecret.ValidateMinimumBytes(normalized.TOTPSecret, normalized.Algorithm); err != nil {
		return err
	}
	if _, err := totpcode.Generate(normalized, time.Unix(0, 0)); err != nil {
		return fmt.Errorf("invalid totp secret: %w", err)
	}
	return nil
}

func printAddResult(output io.Writer, path string, added provider.Provider) error {
	_, err := fmt.Fprintf(output, "saved provider %s to %s\n", added.DisplayLabel(), path)
	return err
}

func validateDeleteSelector(selector storage.DeleteSelector) error {
	if strings.TrimSpace(selector.Issuer) == "" && strings.TrimSpace(selector.Name) == "" {
		return errors.New("at least one of issuer or name is required")
	}
	if strings.TrimSpace(selector.TOTPSecret) == "" {
		return nil
	}
	if _, err := totpsecret.CanonicalBase32(selector.TOTPSecret); err != nil {
		return fmt.Errorf("invalid totp secret: %w", err)
	}
	return nil
}

func deleteError(err error) error {
	switch {
	case errors.Is(err, storage.ErrProviderNotFound):
		return fmt.Errorf("no matching provider found")
	case errors.Is(err, storage.ErrProviderDeleteAmbiguous):
		return fmt.Errorf("ambiguous provider selector: provide more fields such as --name or --totp-secret")
	default:
		return err
	}
}

func printDeleteResult(output io.Writer, path string, deleted provider.Provider) error {
	_, err := fmt.Fprintf(output, "deleted provider %s from %s\n", deleted.DisplayLabel(), path)
	return err
}

func newSearchEngineForName(name string, providers []provider.Provider) (tui.SearchEngine, error) {
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

type showOutput struct {
	Tables []storage.TableSnapshot `json:"tables" yaml:"tables" toml:"tables"`
}

type tomlShowOutput struct {
	Tables []tomlTableSnapshot `toml:"tables"`
}

type tomlTableSnapshot struct {
	Name    string         `toml:"name"`
	Columns []string       `toml:"columns"`
	Rows    []tomlTableRow `toml:"rows"`
}

type tomlTableRow struct {
	Values []tomlTableCell `toml:"values"`
}

type tomlTableCell struct {
	Column string `toml:"column"`
	Value  string `toml:"value,omitempty"`
	Null   bool   `toml:"null,omitempty"`
}

func renderShowTables(output io.Writer, tables []storage.TableSnapshot, format showFormat) error {
	switch format {
	case showFormatTable:
		return renderShowTableOutput(output, tables)
	case showFormatJSON:
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		return encoder.Encode(showOutput{Tables: tables})
	case showFormatYAML:
		encoder := yaml.NewEncoder(output)
		defer func() {
			_ = encoder.Close()
		}()
		return encoder.Encode(showOutput{Tables: tables})
	case showFormatTOML:
		encoder := toml.NewEncoder(output)
		return encoder.Encode(toTOMLShowOutput(tables))
	default:
		return fmt.Errorf("unsupported show format %q: expected table, json, yaml, or toml", format)
	}
}

func redactShowSecrets(tables []storage.TableSnapshot) []storage.TableSnapshot {
	redacted := make([]storage.TableSnapshot, 0, len(tables))
	for _, snapshot := range tables {
		table := storage.TableSnapshot{
			Name:    snapshot.Name,
			Columns: append([]string(nil), snapshot.Columns...),
			Rows:    make([]map[string]any, 0, len(snapshot.Rows)),
		}
		for _, row := range snapshot.Rows {
			redactedRow := make(map[string]any, len(row))
			for column, value := range row {
				if isShowSecretColumn(column) && value != nil {
					redactedRow[column] = redactedShowValue
					continue
				}
				redactedRow[column] = value
			}
			table.Rows = append(table.Rows, redactedRow)
		}
		redacted = append(redacted, table)
	}
	return redacted
}

func isShowSecretColumn(column string) bool {
	switch strings.ToLower(column) {
	case "secret", "totp_secret", "totp_secret_key":
		return true
	default:
		return false
	}
}

func renderShowTableOutput(output io.Writer, tables []storage.TableSnapshot) error {
	for i, snapshot := range tables {
		if i > 0 {
			if _, err := fmt.Fprintln(output); err != nil {
				return fmt.Errorf("write show table separator: %w", err)
			}
		}
		if _, err := fmt.Fprintln(output, snapshot.Name); err != nil {
			return fmt.Errorf("write show table name: %w", err)
		}
		rows := make([][]string, 0, len(snapshot.Rows))
		for _, row := range snapshot.Rows {
			rendered := make([]string, 0, len(snapshot.Columns))
			for _, column := range snapshot.Columns {
				rendered = append(rendered, formatShowValue(row[column]))
			}
			rows = append(rows, rendered)
		}
		rendered := table.New().
			Border(lipgloss.ASCIIBorder()).
			Headers(snapshot.Columns...).
			Rows(rows...).
			Render()
		if _, err := fmt.Fprintln(output, rendered); err != nil {
			return fmt.Errorf("write show table rows: %w", err)
		}
	}
	return nil
}

func toTOMLShowOutput(tables []storage.TableSnapshot) tomlShowOutput {
	converted := tomlShowOutput{Tables: make([]tomlTableSnapshot, 0, len(tables))}
	for _, snapshot := range tables {
		table := tomlTableSnapshot{
			Name:    snapshot.Name,
			Columns: snapshot.Columns,
			Rows:    make([]tomlTableRow, 0, len(snapshot.Rows)),
		}
		for _, row := range snapshot.Rows {
			convertedRow := tomlTableRow{Values: make([]tomlTableCell, 0, len(snapshot.Columns))}
			for _, column := range snapshot.Columns {
				cell := tomlTableCell{Column: column}
				if value := row[column]; value == nil {
					cell.Null = true
				} else {
					cell.Value = formatShowValue(value)
				}
				convertedRow.Values = append(convertedRow.Values, cell)
			}
			table.Rows = append(table.Rows, convertedRow)
		}
		converted.Tables = append(converted.Tables, table)
	}
	return converted
}

func formatShowValue(value any) string {
	if value == nil {
		return "NULL"
	}
	return fmt.Sprint(value)
}
