package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	EnvPrefix        = "GODOCMIRRORTRANSLATOR_"
	ConfigPathEnv    = EnvPrefix + "CONFIG"
	defaultConfigDir = "handwritten-overlay-translator"
)

type Config struct {
	OpenAIAPIKey      string                       `json:"openai_api_key,omitempty"`
	GeminiAPIKey      string                       `json:"gemini_api_key,omitempty"`
	Timeout           time.Duration                `json:"timeout"`
	DefaultProvider   string                       `json:"default_provider"`
	DefaultModel      string                       `json:"default_model,omitempty"`
	DefaultOutputDir  string                       `json:"default_output_dir,omitempty"`
	DefaultFontFamily string                       `json:"default_font_family"`
	DefaultFontSize   float64                      `json:"default_font_size"`
	OutputTemplate    string                       `json:"output_template"`
	OverlayColor      string                       `json:"overlay_color"`
	OverlayOpacity    float64                      `json:"overlay_opacity"`
	PreserveColumns   bool                         `json:"preserve_columns"`
	SourceLanguage    string                       `json:"source_language"`
	TargetLanguage    string                       `json:"target_language"`
	ProviderOptions   map[string]map[string]string `json:"provider_options,omitempty"`
	GUIPreferences    map[string]string            `json:"gui_preferences,omitempty"`
}

func Default() Config {
	return Config{
		Timeout:           30 * time.Second,
		DefaultProvider:   "mock",
		DefaultFontFamily: "Noto Sans",
		DefaultFontSize:   18,
		OutputTemplate:    "{input_basename}_{provider}_{timestamp}.svg",
		OverlayColor:      "#111111",
		OverlayOpacity:    1,
		SourceLanguage:    "Ukrainian",
		TargetLanguage:    "German",
		ProviderOptions:   map[string]map[string]string{},
		GUIPreferences:    map[string]string{},
	}
}

func ResolvePath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if fromEnv := os.Getenv(ConfigPathEnv); fromEnv != "" {
		return fromEnv, nil
	}
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(userConfigDir, defaultConfigDir, "config.json"), nil
}

func Load(path string) (Config, string, error) {
	resolvedPath, err := ResolvePath(path)
	if err != nil {
		return Config{}, "", err
	}

	cfg := Default()
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, resolvedPath, nil
		}
		return Config{}, "", fmt.Errorf("read config: %w", err)
	}
	if len(data) == 0 {
		return cfg, resolvedPath, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, "", fmt.Errorf("decode config: %w", err)
	}
	cfg.normalize()
	return cfg, resolvedPath, nil
}

func LoadEffective(path string) (Config, string, error) {
	cfg, resolvedPath, err := Load(path)
	if err != nil {
		return Config{}, "", err
	}
	if err := cfg.ApplyEnv(os.LookupEnv); err != nil {
		return Config{}, "", err
	}
	cfg.normalize()
	return cfg, resolvedPath, nil
}

func Save(path string, cfg Config) (string, error) {
	resolvedPath, err := ResolvePath(path)
	if err != nil {
		return "", err
	}
	cfg.normalize()
	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode config: %w", err)
	}
	if err := writeFileAtomically(resolvedPath, append(data, '\n'), 0o600); err != nil {
		return "", err
	}
	return resolvedPath, nil
}

func (c Config) Masked() Config {
	c.OpenAIAPIKey = maskSecret(c.OpenAIAPIKey)
	c.GeminiAPIKey = maskSecret(c.GeminiAPIKey)
	return c
}

func (c *Config) normalize() {
	defaults := Default()
	if c.Timeout <= 0 {
		c.Timeout = defaults.Timeout
	}
	if c.DefaultProvider == "" {
		c.DefaultProvider = defaults.DefaultProvider
	}
	if c.DefaultFontFamily == "" {
		c.DefaultFontFamily = defaults.DefaultFontFamily
	}
	if c.DefaultFontSize <= 0 {
		c.DefaultFontSize = defaults.DefaultFontSize
	}
	if c.OutputTemplate == "" {
		c.OutputTemplate = defaults.OutputTemplate
	}
	if c.OverlayColor == "" {
		c.OverlayColor = defaults.OverlayColor
	}
	if c.OverlayOpacity < 0 {
		c.OverlayOpacity = defaults.OverlayOpacity
	}
	if c.SourceLanguage == "" {
		c.SourceLanguage = defaults.SourceLanguage
	}
	if c.TargetLanguage == "" {
		c.TargetLanguage = defaults.TargetLanguage
	}
	if c.ProviderOptions == nil {
		c.ProviderOptions = map[string]map[string]string{}
	}
	if c.GUIPreferences == nil {
		c.GUIPreferences = map[string]string{}
	}
}

func (c *Config) ApplyEnv(lookup func(string) (string, bool)) error {
	stringMappings := map[string]*string{
		EnvPrefix + "OPENAI_API_KEY":      &c.OpenAIAPIKey,
		EnvPrefix + "GEMINI_API_KEY":      &c.GeminiAPIKey,
		EnvPrefix + "DEFAULT_PROVIDER":    &c.DefaultProvider,
		EnvPrefix + "DEFAULT_MODEL":       &c.DefaultModel,
		EnvPrefix + "DEFAULT_OUTPUT_DIR":  &c.DefaultOutputDir,
		EnvPrefix + "DEFAULT_FONT_FAMILY": &c.DefaultFontFamily,
		EnvPrefix + "OUTPUT_TEMPLATE":     &c.OutputTemplate,
		EnvPrefix + "OVERLAY_COLOR":       &c.OverlayColor,
		EnvPrefix + "SOURCE_LANGUAGE":     &c.SourceLanguage,
		EnvPrefix + "TARGET_LANGUAGE":     &c.TargetLanguage,
	}
	for envKey, target := range stringMappings {
		if value, ok := lookup(envKey); ok {
			*target = value
		}
	}

	if value, ok := lookup(EnvPrefix + "TIMEOUT"); ok {
		duration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"TIMEOUT", err)
		}
		c.Timeout = duration
	}
	if value, ok := lookup(EnvPrefix + "DEFAULT_FONT_SIZE"); ok {
		fontSize, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"DEFAULT_FONT_SIZE", err)
		}
		c.DefaultFontSize = fontSize
	}
	if value, ok := lookup(EnvPrefix + "OVERLAY_OPACITY"); ok {
		opacity, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"OVERLAY_OPACITY", err)
		}
		c.OverlayOpacity = opacity
	}
	if value, ok := lookup(EnvPrefix + "PRESERVE_COLUMNS"); ok {
		preserveColumns, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"PRESERVE_COLUMNS", err)
		}
		c.PreserveColumns = preserveColumns
	}
	return nil
}

func (c *Config) Set(key, value string) error {
	switch strings.ToLower(key) {
	case "openai_api_key":
		c.OpenAIAPIKey = value
	case "gemini_api_key":
		c.GeminiAPIKey = value
	case "timeout":
		duration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("parse timeout: %w", err)
		}
		c.Timeout = duration
	case "default_provider":
		c.DefaultProvider = value
	case "default_model":
		c.DefaultModel = value
	case "default_output_dir":
		c.DefaultOutputDir = value
	case "default_font_family":
		c.DefaultFontFamily = value
	case "default_font_size":
		fontSize, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse default_font_size: %w", err)
		}
		c.DefaultFontSize = fontSize
	case "output_template":
		c.OutputTemplate = value
	case "overlay_color":
		c.OverlayColor = value
	case "overlay_opacity":
		opacity, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse overlay_opacity: %w", err)
		}
		c.OverlayOpacity = opacity
	case "preserve_columns":
		preserveColumns, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse preserve_columns: %w", err)
		}
		c.PreserveColumns = preserveColumns
	case "source_language":
		c.SourceLanguage = value
	case "target_language":
		c.TargetLanguage = value
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	c.normalize()
	return nil
}

func maskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

func writeFileAtomically(path string, data []byte, perm os.FileMode) error {
	tempFile, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tempName := tempFile.Name()
	defer os.Remove(tempName)

	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tempFile.Chmod(perm); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("chmod temp config: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("rename temp config: %w", err)
	}
	return nil
}
