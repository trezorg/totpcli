package importer

import (
	"encoding/base64"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/trezorg/totpcli/internal/provider"
	"github.com/trezorg/totpcli/internal/storage"
	"github.com/trezorg/totpcli/internal/testutil"
	"google.golang.org/protobuf/encoding/protowire"
)

func TestImageQRDecoderDecodesLocalImagePath(t *testing.T) {
	t.Parallel()

	want := migrationURI(
		migrationPayload(
			otpParameters([]byte("hello"), "trezorg", "GitHub", 1, 1, 2),
			protowire.AppendVarint(protowire.AppendTag(nil, 2, protowire.VarintType), 1),
		),
	)
	code, err := qr.Encode(want, qr.M, qr.Auto)
	if err != nil {
		t.Fatalf("encode QR fixture: %v", err)
	}
	scaled, err := barcode.Scale(code, 256, 256)
	if err != nil {
		t.Fatalf("scale QR fixture: %v", err)
	}

	path := filepath.Join(t.TempDir(), "export.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create QR fixture: %v", err)
	}
	if err := png.Encode(file, scaled); err != nil {
		_ = file.Close()
		t.Fatalf("write QR fixture: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close QR fixture: %v", err)
	}

	got, err := ImageQRDecoder{}.Decode(path)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if got != want {
		t.Fatalf("Decode = %q, want %q", got, want)
	}
}

func TestImageQRDecoderFallsBackToZbarimg(t *testing.T) {
	t.Parallel()

	run := func(name string, args ...string) *exec.Cmd {
		if name != "zbarimg" {
			t.Fatalf("command = %q, want zbarimg", name)
		}
		return exec.Command("printf", "QR-Code:otpauth-migration://offline?data=abc\n")
	}

	got, err := decodeWithZbarimg("/tmp/export.jpg", run)
	if err != nil {
		t.Fatalf("decodeWithZbarimg returned error: %v", err)
	}
	if got != "otpauth-migration://offline?data=abc" {
		t.Fatalf("decoded = %q, want migration URI", got)
	}
}

func TestMigrationDecoderParsesURIAndPayload(t *testing.T) {
	t.Parallel()

	uri := migrationURI(
		migrationPayload(
			otpParameters([]byte("hello"), "trezorg", "GitHub", 1, 1, 2),
			protowire.AppendVarint(protowire.AppendTag(nil, 2, protowire.VarintType), 1),
			protowire.AppendVarint(protowire.AppendTag(nil, 3, protowire.VarintType), 1),
			protowire.AppendVarint(protowire.AppendTag(nil, 4, protowire.VarintType), 0),
			protowire.AppendVarint(protowire.AppendTag(nil, 5, protowire.VarintType), 42),
		),
	)

	payload, err := MigrationDecoder{}.Decode(uri)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	if payload.Version != 1 {
		t.Fatalf("Version = %d, want 1", payload.Version)
	}
	if len(payload.Accounts) != 1 {
		t.Fatalf("len(Accounts) = %d, want 1", len(payload.Accounts))
	}
	got := payload.Accounts[0]
	if got.Name != "trezorg" || got.Issuer != "GitHub" {
		t.Fatalf("account = %#v, want issuer/name preserved", got)
	}
	if string(got.Secret) != "hello" {
		t.Fatalf("secret = %q, want hello", string(got.Secret))
	}
}

func TestMigrationDecoderAcceptsVersionTwoPayload(t *testing.T) {
	t.Parallel()

	uri := migrationURI(
		migrationPayload(
			otpParameters([]byte("hello"), "trezorg", "GitHub", 1, 1, 2),
			protowire.AppendVarint(protowire.AppendTag(nil, 2, protowire.VarintType), 2),
		),
	)

	payload, err := MigrationDecoder{}.Decode(uri)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if payload.Version != 2 {
		t.Fatalf("Version = %d, want 2", payload.Version)
	}
}

func TestMigrationDecoderParsesOTPAuthURI(t *testing.T) {
	t.Parallel()

	uri := "otpauth://totp/gitlab-internal.wildberries.ru:gitlab-internal.wildberries.ru_nemilencev.igor%40rwb.ru?secret=YS6ZG7EEB52IOU42XIZUWRGS2W7CFPQZ&issuer=gitlab-internal.wildberries.ru"

	payload, err := MigrationDecoder{}.Decode(uri)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if len(payload.Accounts) != 1 {
		t.Fatalf("len(Accounts) = %d, want 1", len(payload.Accounts))
	}
	account := payload.Accounts[0]
	if account.Issuer != "gitlab-internal.wildberries.ru" {
		t.Fatalf("Issuer = %q, want gitlab-internal.wildberries.ru", account.Issuer)
	}
	if account.Name != "gitlab-internal.wildberries.ru_nemilencev.igor@rwb.ru" {
		t.Fatalf("Name = %q, want decoded label account name", account.Name)
	}
	if account.Type != AccountTypeTOTP {
		t.Fatalf("Type = %d, want TOTP", account.Type)
	}
	if account.Algorithm != AlgorithmSHA1 {
		t.Fatalf("Algorithm = %d, want SHA1", account.Algorithm)
	}
	if account.Digits != DigitCountSix {
		t.Fatalf("Digits = %d, want SIX", account.Digits)
	}
}

func TestMigrationDecoderParsesOTPAuthURIWithOptionalFields(t *testing.T) {
	t.Parallel()

	// No issuer in query, default algorithm/digits, label issuer used.
	uri := "otpauth://totp/GitHub:trezorg?secret=jbswy3dpehpk3pxp&algorithm=SHA256&digits=8"

	payload, err := MigrationDecoder{}.Decode(uri)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	account := payload.Accounts[0]
	if account.Issuer != "GitHub" {
		t.Fatalf("Issuer = %q, want GitHub from label", account.Issuer)
	}
	if account.Name != "trezorg" {
		t.Fatalf("Name = %q, want trezorg", account.Name)
	}
	if account.Algorithm != AlgorithmSHA256 {
		t.Fatalf("Algorithm = %d, want SHA256", account.Algorithm)
	}
	if account.Digits != DigitCountEight {
		t.Fatalf("Digits = %d, want EIGHT", account.Digits)
	}
}

func TestMigrationDecoderRejectsMalformedURIs(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"https://example.com/GitHub:trezorg":                                                                         "otpauth",
		"otpauth-migration://offline":                                                                                "data",
		"otpauth-migration://offline?data=!":                                                                         "base64",
		"otpauth://totp/GitHub:trezorg":                                                                              "secret",
		"otpauth://totp/GitHub:trezorg?secret=not-base32!":                                                           "secret",
		"otpauth://totp/GitHub:trezorg?secret=JBSWY3DPEHPK3PXP&algorithm=MD5":                                        "algorithm",
		"otpauth://totp/GitHub:trezorg?secret=JBSWY3DPEHPK3PXP&digits=7":                                             "digits",
		"otpauth://hmac/GitHub:trezorg?secret=JBSWY3DPEHPK3PXP":                                                      "type",
		migrationURI(migrationPayload(protowire.AppendVarint(protowire.AppendTag(nil, 2, protowire.VarintType), 1))): "account",
		migrationURI(migrationPayload(protowire.AppendVarint(protowire.AppendTag(nil, 2, protowire.VarintType), 3))): "version",
	}

	for uri, want := range tests {
		t.Run(want, func(t *testing.T) {
			t.Parallel()

			_, err := MigrationDecoder{}.Decode(uri)
			if err == nil {
				t.Fatal("Decode returned nil error")
			}
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %q, want mention %q", err.Error(), want)
			}
		})
	}
}

func TestProviderConverterConvertsSupportedTOTPAccounts(t *testing.T) {
	t.Parallel()

	providers, err := ProviderConverter{}.Convert(MigrationPayload{
		Version: 1,
		Accounts: []Account{{
			Secret:    []byte("hello"),
			Name:      " trezorg ",
			Issuer:    " GitHub ",
			Algorithm: AlgorithmSHA1,
			Digits:    DigitCountSix,
			Type:      AccountTypeTOTP,
		}},
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	want := provider.RawProvider{
		Name:       "trezorg",
		Issuer:     "GitHub",
		Algorithm:  "SHA1",
		Digits:     "SIX",
		Type:       "TOTP",
		TOTPSecret: "NBSWY3DP",
	}
	if providers[0] != want {
		t.Fatalf("provider = %#v, want %#v", providers[0], want)
	}
}

func TestProviderConverterUsesNormalizedDefaults(t *testing.T) {
	t.Parallel()

	providers, err := ProviderConverter{}.Convert(MigrationPayload{
		Version: 1,
		Accounts: []Account{{
			Secret: []byte("hello"),
			Name:   "trezorg",
			Issuer: "GitHub",
			Type:   AccountTypeTOTP,
		}},
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	want := provider.RawProvider{
		Name:       "trezorg",
		Issuer:     "GitHub",
		Algorithm:  "SHA1",
		Digits:     "SIX",
		Type:       "TOTP",
		TOTPSecret: "NBSWY3DP",
	}
	if providers[0] != want {
		t.Fatalf("provider = %#v, want %#v", providers[0], want)
	}
}

func TestProviderConverterRejectsUnsupportedAccounts(t *testing.T) {
	t.Parallel()

	tests := map[string]Account{
		"type":      {Secret: []byte("hello"), Issuer: "GitHub", Algorithm: AlgorithmSHA1, Digits: DigitCountSix, Type: AccountTypeHOTP},
		"algorithm": {Secret: []byte("hello"), Issuer: "GitHub", Algorithm: AlgorithmMD5, Digits: DigitCountSix, Type: AccountTypeTOTP},
		"digits":    {Secret: []byte("hello"), Issuer: "GitHub", Algorithm: AlgorithmSHA1, Digits: 7, Type: AccountTypeTOTP},
		"secret":    {Issuer: "GitHub", Algorithm: AlgorithmSHA1, Digits: DigitCountSix, Type: AccountTypeTOTP},
	}

	for want, account := range tests {
		t.Run(want, func(t *testing.T) {
			t.Parallel()

			_, err := ProviderConverter{}.Convert(MigrationPayload{Version: 1, Accounts: []Account{account}})
			if err == nil {
				t.Fatal("Convert returned nil error")
			}
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %q, want mention %q", err.Error(), want)
			}
		})
	}
}

func TestServiceImportPersistsQRProvidersThroughRepository(t *testing.T) {
	t.Parallel()

	repo := &stubProviderRepository{}
	result, err := Service{
		QRDecoder: testutil.StubQRDecoder{URI: migrationURI(
			migrationPayload(
				otpParameters([]byte("hello"), "trezorg", "GitHub", 1, 1, 2),
				protowire.AppendVarint(protowire.AppendTag(nil, 2, protowire.VarintType), 1),
			),
		)},
	}.Import("/tmp/export.png", repo)
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	if len(repo.imports) != 1 || repo.imports[0].Issuer != "GitHub" {
		t.Fatalf("imports = %#v, want GitHub provider", repo.imports)
	}
	if result.Imported != 1 || result.Added != 1 || result.Overridden != 0 || result.Final != 1 {
		t.Fatalf("result = %#v, want repository result", result)
	}
}

func TestFileServiceImportsProviderFilesThroughRepository(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "providers.json")
	yamlPath := filepath.Join(dir, "extra.yaml")
	writeFile(t, jsonPath, `[
  {
    "name": "trezorg",
    "issuer": "GitHub",
    "algorithm": "SHA1",
    "digits": "SIX",
    "type": "TOTP",
    "totpSecret": "JBSWY3DPEHPK3PXP"
  }
]`)
	writeFile(t, yamlPath, `- name: root
  issuer: AWS
  algorithm: SHA1
  digits: SIX
  type: TOTP
  totpSecret: JBSWY3DPEHPK3PXP
`)

	repo := &stubProviderRepository{}
	result, err := FileService{}.Import([]string{jsonPath, yamlPath}, "", "", repo)
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	if len(repo.imports) != 2 {
		t.Fatalf("len(imports) = %d, want 2", len(repo.imports))
	}
	if repo.imports[0].Issuer != "GitHub" || repo.imports[1].Issuer != "AWS" {
		t.Fatalf("imports = %#v, want GitHub then AWS", repo.imports)
	}
	if result.Imported != 2 || result.Added != 2 || result.Final != 2 {
		t.Fatalf("result = %#v, want imported=2 added=2 final=2", result)
	}
}

func TestFileServiceExplicitFormatAppliesToAllFiles(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "providers.txt")
	writeFile(t, path, `- name: trezorg
  issuer: GitHub
  algorithm: SHA1
  digits: SIX
  type: TOTP
  totpSecret: JBSWY3DPEHPK3PXP
`)

	repo := &stubProviderRepository{}
	if _, err := (FileService{}).Import([]string{path}, "yaml", "", repo); err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	if len(repo.imports) != 1 || repo.imports[0].Issuer != "GitHub" {
		t.Fatalf("imports = %#v, want YAML provider", repo.imports)
	}
}

func migrationURI(payload []byte) string {
	return "otpauth-migration://offline?data=" + base64.StdEncoding.EncodeToString(payload)
}

func migrationPayload(fields ...[]byte) []byte {
	var payload []byte
	for _, field := range fields {
		payload = append(payload, field...)
	}
	return payload
}

func otpParameters(secret []byte, name string, issuer string, algorithm uint64, digits uint64, accountType uint64) []byte {
	var account []byte
	account = protowire.AppendBytes(protowire.AppendTag(account, 1, protowire.BytesType), secret)
	account = protowire.AppendString(protowire.AppendTag(account, 2, protowire.BytesType), name)
	account = protowire.AppendString(protowire.AppendTag(account, 3, protowire.BytesType), issuer)
	account = protowire.AppendVarint(protowire.AppendTag(account, 4, protowire.VarintType), algorithm)
	account = protowire.AppendVarint(protowire.AppendTag(account, 5, protowire.VarintType), digits)
	account = protowire.AppendVarint(protowire.AppendTag(account, 6, protowire.VarintType), accountType)
	return protowire.AppendBytes(protowire.AppendTag(nil, 1, protowire.BytesType), account)
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

type stubProviderRepository struct {
	imports []provider.RawProvider
}

func (s *stubProviderRepository) UpsertProviders(imports []provider.RawProvider) (storage.ImportResult, error) {
	s.imports = append([]provider.RawProvider(nil), imports...)
	return storage.ImportResult{
		OutputPath: "/tmp/providers.sqlite",
		Imported:   len(imports),
		Added:      len(imports),
		Final:      len(imports),
	}, nil
}
