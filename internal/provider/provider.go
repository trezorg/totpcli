package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
	FormatTOML Format = "toml"
)

type Algorithm string

const (
	AlgorithmSHA1   Algorithm = "SHA1"
	AlgorithmSHA256 Algorithm = "SHA256"
	AlgorithmSHA512 Algorithm = "SHA512"
)

type Options struct {
	Path   string
	Format string
	Home   string
}

type ResolvedOptions struct {
	Path   string
	Format Format
}

type RawProvider struct {
	Secret     string `json:"secret" yaml:"secret" toml:"secret"`
	Name       string `json:"name" yaml:"name" toml:"name"`
	Issuer     string `json:"issuer" yaml:"issuer" toml:"issuer"`
	Algorithm  string `json:"algorithm" yaml:"algorithm" toml:"algorithm"`
	Digits     string `json:"digits" yaml:"digits" toml:"digits"`
	Type       string `json:"type" yaml:"type" toml:"type"`
	TOTPSecret string `json:"totpSecret" yaml:"totpSecret" toml:"totpSecret"`
}

type Provider struct {
	Secret     string
	Name       string
	Issuer     string
	Algorithm  Algorithm
	Digits     int
	Type       string
	TOTPSecret string
}

func ResolveOptions(opts Options) (ResolvedOptions, error) {
	format, err := resolveFormat(opts.Path, opts.Format)
	if err != nil {
		return ResolvedOptions{}, err
	}

	path := opts.Path
	if path == "" {
		path = filepath.Join("~", "totp."+string(format))
	}

	expanded, err := ExpandHome(path, opts.Home)
	if err != nil {
		return ResolvedOptions{}, err
	}

	return ResolvedOptions{Path: expanded, Format: format}, nil
}

func ParseFormat(value string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "json":
		return FormatJSON, nil
	case "yaml", "yml":
		return FormatYAML, nil
	case "toml":
		return FormatTOML, nil
	default:
		return "", fmt.Errorf("unsupported format %q: expected json, yaml, or toml", value)
	}
}

func ExpandHome(path string, home string) (string, error) {
	if path == "" {
		return "", errors.New("path is required")
	}
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
	}
	if path == "~" {
		return home, nil
	}
	rest, _ := strings.CutPrefix(path, "~/")
	return filepath.Join(home, rest), nil
}

func Load(path string, format Format) ([]Provider, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read provider file %q: %w", path, err)
	}
	return Parse(data, format)
}

func Parse(data []byte, format Format) ([]Provider, error) {
	raw, err := ParseRaw(data, format)
	if err != nil {
		return nil, err
	}

	providers := make([]Provider, 0, len(raw))
	for i, record := range raw {
		provider, err := Normalize(record)
		if err != nil {
			return nil, fmt.Errorf("provider %d: %w", i+1, err)
		}
		providers = append(providers, provider)
	}
	return providers, nil
}

func LoadRaw(path string, format Format) ([]RawProvider, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read provider file %q: %w", path, err)
	}
	return ParseRaw(data, format)
}

func ParseRaw(data []byte, format Format) ([]RawProvider, error) {
	var raw []RawProvider
	switch format {
	case FormatJSON:
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse json provider file: %w", err)
		}
	case FormatYAML:
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse yaml provider file: %w", err)
		}
	case FormatTOML:
		var doc struct {
			Providers []RawProvider `toml:"providers"`
		}
		if err := toml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse toml provider file: %w", err)
		}
		raw = doc.Providers
	default:
		return nil, fmt.Errorf("unsupported provider format %q", format)
	}
	return raw, nil
}

func Normalize(raw RawProvider) (Provider, error) {
	providerType := strings.ToUpper(strings.TrimSpace(raw.Type))
	if providerType == "" {
		providerType = "TOTP"
	}
	if providerType != "TOTP" {
		return Provider{}, fmt.Errorf("unsupported type %q: expected TOTP", raw.Type)
	}

	secret := strings.TrimSpace(raw.TOTPSecret)
	if secret == "" {
		return Provider{}, errors.New("totpSecret is required")
	}

	name := strings.TrimSpace(raw.Name)
	issuer := strings.TrimSpace(raw.Issuer)
	if name == "" && issuer == "" {
		return Provider{}, errors.New("issuer or name is required")
	}

	algorithm, err := normalizeAlgorithm(raw.Algorithm)
	if err != nil {
		return Provider{}, err
	}
	digits, err := normalizeDigits(raw.Digits)
	if err != nil {
		return Provider{}, err
	}

	return Provider{
		Secret:     raw.Secret,
		Name:       name,
		Issuer:     issuer,
		Algorithm:  algorithm,
		Digits:     digits,
		Type:       providerType,
		TOTPSecret: secret,
	}, nil
}

func (p Provider) DisplayLabel() string {
	switch {
	case p.Issuer != "" && p.Name != "":
		return p.Issuer + "/" + p.Name
	case p.Issuer != "":
		return p.Issuer
	default:
		return p.Name
	}
}

func resolveFormat(path string, explicit string) (Format, error) {
	if explicit != "" {
		return ParseFormat(explicit)
	}
	return DetectFormat(path), nil
}

func DetectFormat(path string) Format {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return FormatYAML
	case ".toml":
		return FormatTOML
	default:
		return FormatJSON
	}
}

func normalizeAlgorithm(value string) (Algorithm, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "", "SHA1":
		return AlgorithmSHA1, nil
	case "SHA256":
		return AlgorithmSHA256, nil
	case "SHA512":
		return AlgorithmSHA512, nil
	default:
		return "", fmt.Errorf("unsupported algorithm %q: expected SHA1, SHA256, or SHA512", value)
	}
}

func normalizeDigits(value string) (int, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	switch normalized {
	case "", "SIX":
		return 6, nil
	case "EIGHT":
		return 8, nil
	}

	digits, err := strconv.Atoi(normalized)
	if err != nil {
		return 0, fmt.Errorf("unsupported digits %q: expected SIX, EIGHT, 6, or 8", value)
	}
	if digits != 6 && digits != 8 {
		return 0, fmt.Errorf("unsupported digits %q: expected SIX, EIGHT, 6, or 8", value)
	}
	return digits, nil
}
