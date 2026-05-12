package importer

import (
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/trezorg/totpcli/internal/provider"
	"github.com/trezorg/totpcli/internal/storage"
	"google.golang.org/protobuf/encoding/protowire"
)

type AccountAlgorithm uint64

const (
	AlgorithmUnspecified AccountAlgorithm = 0
	AlgorithmSHA1        AccountAlgorithm = 1
	AlgorithmSHA256      AccountAlgorithm = 2
	AlgorithmSHA512      AccountAlgorithm = 3
	AlgorithmMD5         AccountAlgorithm = 4
)

type DigitCount uint64

const (
	DigitCountUnspecified DigitCount = 0
	DigitCountSix         DigitCount = 1
	DigitCountEight       DigitCount = 2
)

type AccountType uint64

const (
	AccountTypeUnspecified AccountType = 0
	AccountTypeHOTP        AccountType = 1
	AccountTypeTOTP        AccountType = 2
)

type Account struct {
	Secret    []byte
	Name      string
	Issuer    string
	Algorithm AccountAlgorithm
	Digits    DigitCount
	Type      AccountType
}

type MigrationPayload struct {
	Accounts   []Account
	Version    int
	BatchSize  int
	BatchIndex int
	BatchID    int
}

type QRDecoder interface {
	Decode(path string) (string, error)
}

type PayloadDecoder interface {
	Decode(uri string) (MigrationPayload, error)
}

type Converter interface {
	Convert(payload MigrationPayload) ([]provider.RawProvider, error)
}

type ProviderRepository interface {
	UpsertProviders([]provider.RawProvider) (storage.ImportResult, error)
}

type Service struct {
	QRDecoder QRDecoder
	Decoder   PayloadDecoder
	Converter Converter
}

func (s Service) Import(imagePath string, repo ProviderRepository) (storage.ImportResult, error) {
	imports, err := s.Providers(imagePath)
	if err != nil {
		return storage.ImportResult{}, err
	}
	if repo == nil {
		return storage.ImportResult{}, errors.New("provider repository is required")
	}
	return repo.UpsertProviders(imports)
}

type FileService struct{}

func (FileService) Import(paths []string, explicitFormat string, home string, repo ProviderRepository) (storage.ImportResult, error) {
	if repo == nil {
		return storage.ImportResult{}, errors.New("provider repository is required")
	}
	if len(paths) == 0 {
		return storage.ImportResult{}, errors.New("provider file path is required")
	}
	var imports []provider.RawProvider
	for _, path := range paths {
		expanded, err := provider.ExpandHome(path, home)
		if err != nil {
			return storage.ImportResult{}, err
		}
		var format provider.Format
		if strings.TrimSpace(explicitFormat) == "" {
			format = provider.DetectFormat(expanded)
		} else {
			parsed, err := provider.ParseFormat(explicitFormat)
			if err != nil {
				return storage.ImportResult{}, err
			}
			format = parsed
		}
		raw, err := provider.LoadRaw(expanded, format)
		if err != nil {
			return storage.ImportResult{}, err
		}
		imports = append(imports, raw...)
	}
	result, err := repo.UpsertProviders(imports)
	if err != nil {
		return storage.ImportResult{}, err
	}
	return result, nil
}

func (s Service) Providers(imagePath string) ([]provider.RawProvider, error) {
	qrDecoder := s.QRDecoder
	if qrDecoder == nil {
		qrDecoder = ImageQRDecoder{}
	}
	decoder := s.Decoder
	if decoder == nil {
		decoder = MigrationDecoder{}
	}
	converter := s.Converter
	if converter == nil {
		converter = ProviderConverter{}
	}

	uri, err := qrDecoder.Decode(imagePath)
	if err != nil {
		return nil, fmt.Errorf("decode QR image %q: %w", imagePath, err)
	}
	payload, err := decoder.Decode(uri)
	if err != nil {
		return nil, err
	}
	imports, err := converter.Convert(payload)
	if err != nil {
		return nil, err
	}
	return imports, nil
}

type ImageQRDecoder struct{}
type commandRunner func(string, ...string) *exec.Cmd

func (ImageQRDecoder) Decode(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}
	bitmap, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", fmt.Errorf("prepare image for QR decoding: %w", err)
	}
	result, err := qrcode.NewQRCodeReader().Decode(bitmap, nil)
	if err != nil {
		zbarResult, zbarErr := decodeWithZbarimg(path, exec.Command)
		if zbarErr == nil {
			return zbarResult, nil
		}
		return "", fmt.Errorf("decode QR code: %w (zbarimg fallback: %v)", err, zbarErr)
	}
	return result.GetText(), nil
}

type MigrationDecoder struct{}

func (MigrationDecoder) Decode(rawURI string) (MigrationPayload, error) {
	parsed, err := url.Parse(rawURI)
	if err != nil {
		return MigrationPayload{}, fmt.Errorf("parse migration URI: %w", err)
	}
	if parsed.Scheme != "otpauth-migration" {
		return MigrationPayload{}, fmt.Errorf("unsupported QR URI %q: expected otpauth-migration URI", parsed.Scheme)
	}

	data := parsed.Query().Get("data")
	if data == "" {
		return MigrationPayload{}, errors.New("migration URI data parameter is required")
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return MigrationPayload{}, fmt.Errorf("base64 decode migration data: %w", err)
	}

	payload, err := decodePayload(decoded)
	if err != nil {
		return MigrationPayload{}, err
	}
	if payload.Version != 1 && payload.Version != 2 {
		return MigrationPayload{}, fmt.Errorf("unsupported migration payload version %d: expected 1 or 2", payload.Version)
	}
	if len(payload.Accounts) == 0 {
		return MigrationPayload{}, errors.New("migration payload account list is empty")
	}
	return payload, nil
}

type ProviderConverter struct{}

func (ProviderConverter) Convert(payload MigrationPayload) ([]provider.RawProvider, error) {
	providers := make([]provider.RawProvider, 0, len(payload.Accounts))
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)

	for i, account := range payload.Accounts {
		if len(account.Secret) == 0 {
			return nil, fmt.Errorf("account %d secret is required", i+1)
		}
		if account.Type != AccountTypeTOTP {
			return nil, fmt.Errorf("account %d unsupported type %d: expected TOTP", i+1, account.Type)
		}

		algorithm, err := convertAlgorithm(account.Algorithm)
		if err != nil {
			return nil, fmt.Errorf("account %d %w", i+1, err)
		}
		digits, err := convertDigits(account.Digits)
		if err != nil {
			return nil, fmt.Errorf("account %d %w", i+1, err)
		}

		raw := provider.RawProvider{
			Name:       strings.TrimSpace(account.Name),
			Issuer:     strings.TrimSpace(account.Issuer),
			Algorithm:  algorithm,
			Digits:     digits,
			Type:       "TOTP",
			TOTPSecret: encoder.EncodeToString(account.Secret),
		}
		normalized, err := provider.Normalize(raw)
		if err != nil {
			return nil, fmt.Errorf("account %d provider conversion: %w", i+1, err)
		}
		providers = append(providers, rawProviderFromNormalized(normalized))
	}
	return providers, nil
}

func rawProviderFromNormalized(normalized provider.Provider) provider.RawProvider {
	digits := fmt.Sprint(normalized.Digits)
	switch normalized.Digits {
	case 6:
		digits = "SIX"
	case 8:
		digits = "EIGHT"
	}
	return provider.RawProvider{
		Secret:     normalized.Secret,
		Name:       normalized.Name,
		Issuer:     normalized.Issuer,
		Algorithm:  string(normalized.Algorithm),
		Digits:     digits,
		Type:       normalized.Type,
		TOTPSecret: normalized.TOTPSecret,
	}
}

func decodePayload(data []byte) (MigrationPayload, error) {
	var payload MigrationPayload
	for len(data) > 0 {
		num, typ, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return MigrationPayload{}, fmt.Errorf("parse migration payload tag: %w", protowire.ParseError(tagLen))
		}
		data = data[tagLen:]

		switch num {
		case 1:
			value, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return MigrationPayload{}, fmt.Errorf("parse migration payload account: %w", protowire.ParseError(n))
			}
			account, err := decodeAccount(value)
			if err != nil {
				return MigrationPayload{}, err
			}
			payload.Accounts = append(payload.Accounts, account)
			data = data[n:]
		case 2:
			value, n, err := consumeVarintField(num, typ, data)
			if err != nil {
				return MigrationPayload{}, err
			}
			payload.Version = int(value)
			data = data[n:]
		case 3:
			value, n, err := consumeVarintField(num, typ, data)
			if err != nil {
				return MigrationPayload{}, err
			}
			payload.BatchSize = int(value)
			data = data[n:]
		case 4:
			value, n, err := consumeVarintField(num, typ, data)
			if err != nil {
				return MigrationPayload{}, err
			}
			payload.BatchIndex = int(value)
			data = data[n:]
		case 5:
			value, n, err := consumeVarintField(num, typ, data)
			if err != nil {
				return MigrationPayload{}, err
			}
			payload.BatchID = int(value)
			data = data[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return MigrationPayload{}, fmt.Errorf("parse unknown migration payload field %d: %w", num, protowire.ParseError(n))
			}
			data = data[n:]
		}
	}
	return payload, nil
}

func decodeAccount(data []byte) (Account, error) {
	var account Account
	for len(data) > 0 {
		num, typ, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return Account{}, fmt.Errorf("parse account tag: %w", protowire.ParseError(tagLen))
		}
		data = data[tagLen:]

		switch num {
		case 1:
			value, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return Account{}, fmt.Errorf("parse account secret: %w", protowire.ParseError(n))
			}
			account.Secret = append([]byte(nil), value...)
			data = data[n:]
		case 2:
			value, n, err := consumeStringField(num, typ, data)
			if err != nil {
				return Account{}, err
			}
			account.Name = value
			data = data[n:]
		case 3:
			value, n, err := consumeStringField(num, typ, data)
			if err != nil {
				return Account{}, err
			}
			account.Issuer = value
			data = data[n:]
		case 4:
			value, n, err := consumeVarintField(num, typ, data)
			if err != nil {
				return Account{}, err
			}
			account.Algorithm = AccountAlgorithm(value)
			data = data[n:]
		case 5:
			value, n, err := consumeVarintField(num, typ, data)
			if err != nil {
				return Account{}, err
			}
			account.Digits = DigitCount(value)
			data = data[n:]
		case 6:
			value, n, err := consumeVarintField(num, typ, data)
			if err != nil {
				return Account{}, err
			}
			account.Type = AccountType(value)
			data = data[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return Account{}, fmt.Errorf("parse unknown account field %d: %w", num, protowire.ParseError(n))
			}
			data = data[n:]
		}
	}
	return account, nil
}

func consumeVarintField(num protowire.Number, typ protowire.Type, data []byte) (uint64, int, error) {
	if typ != protowire.VarintType {
		return 0, 0, fmt.Errorf("field %d has wire type %d, expected varint", num, typ)
	}
	value, n := protowire.ConsumeVarint(data)
	if n < 0 {
		return 0, 0, fmt.Errorf("parse field %d varint: %w", num, protowire.ParseError(n))
	}
	return value, n, nil
}

func consumeStringField(num protowire.Number, typ protowire.Type, data []byte) (string, int, error) {
	if typ != protowire.BytesType {
		return "", 0, fmt.Errorf("field %d has wire type %d, expected bytes", num, typ)
	}
	value, n := protowire.ConsumeString(data)
	if n < 0 {
		return "", 0, fmt.Errorf("parse field %d string: %w", num, protowire.ParseError(n))
	}
	return value, n, nil
}

func convertAlgorithm(value AccountAlgorithm) (string, error) {
	switch value {
	case AlgorithmUnspecified, AlgorithmSHA1:
		return "SHA1", nil
	case AlgorithmSHA256:
		return "SHA256", nil
	case AlgorithmSHA512:
		return "SHA512", nil
	default:
		return "", fmt.Errorf("unsupported algorithm %d: expected SHA1, SHA256, or SHA512", value)
	}
}

func convertDigits(value DigitCount) (string, error) {
	switch value {
	case DigitCountUnspecified, DigitCountSix:
		return "SIX", nil
	case DigitCountEight:
		return "EIGHT", nil
	default:
		return "", fmt.Errorf("unsupported digits %d: expected SIX or EIGHT", value)
	}
}

func decodeWithZbarimg(path string, run commandRunner) (string, error) {
	output, err := run("zbarimg", "-q", path).Output()
	if err != nil {
		return "", err
	}
	for line := range strings.Lines(string(output)) {
		line = strings.TrimSpace(line)
		if value, ok := strings.CutPrefix(line, "QR-Code:"); ok {
			return strings.TrimSpace(value), nil
		}
		if strings.HasPrefix(line, "otpauth-migration://") {
			return line, nil
		}
	}
	return "", errors.New("zbarimg did not return a QR code")
}
