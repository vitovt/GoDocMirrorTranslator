package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"godocmirrortranslator/internal/provider"
	base "godocmirrortranslator/internal/renderer"
)

const (
	EnvPrefix        = "GODOCMIRRORTRANSLATOR_"
	ConfigPathEnv    = EnvPrefix + "CONFIG"
	defaultConfigDir = "handwritten-overlay-translator"
	filesDirEnv      = "FILESDIR"
)

type Config struct {
	OpenAIAPIKey           string                       `json:"openai_api_key,omitempty"`
	GeminiAPIKey           string                       `json:"gemini_api_key,omitempty"`
	Timeout                time.Duration                `json:"timeout"`
	DefaultProvider        string                       `json:"default_provider"`
	DefaultRenderer        string                       `json:"default_renderer"`
	DefaultModel           string                       `json:"default_model,omitempty"`
	DefaultOutputDir       string                       `json:"default_output_dir,omitempty"`
	DefaultFontFamily      string                       `json:"default_font_family"`
	DefaultFontSize        float64                      `json:"default_font_size"`
	DefaultFontWeight      string                       `json:"default_font_weight,omitempty"`
	DefaultPageLayout      string                       `json:"default_page_layout,omitempty"`
	OutputTemplate         string                       `json:"output_template"`
	OverlayColor           string                       `json:"overlay_color"`
	OverlayOpacity         float64                      `json:"overlay_opacity"`
	TextOutlineColor       string                       `json:"text_outline_color,omitempty"`
	TextOutlineWidth       float64                      `json:"text_outline_width"`
	TextBackgroundEnabled  bool                         `json:"text_background_enabled"`
	TextBackgroundColor    string                       `json:"text_background_color,omitempty"`
	TextBackgroundOpacity  float64                      `json:"text_background_opacity"`
	TextBackgroundPaddingX float64                      `json:"text_background_padding_x"`
	TextBackgroundPaddingY float64                      `json:"text_background_padding_y"`
	TextBackgroundRadius   float64                      `json:"text_background_radius"`
	TextShadowEnabled      bool                         `json:"text_shadow_enabled"`
	TextShadowColor        string                       `json:"text_shadow_color,omitempty"`
	TextShadowOpacity      float64                      `json:"text_shadow_opacity"`
	TextShadowBlur         float64                      `json:"text_shadow_blur"`
	TextShadowOffsetX      float64                      `json:"text_shadow_offset_x"`
	TextShadowOffsetY      float64                      `json:"text_shadow_offset_y"`
	PreserveColumns        bool                         `json:"preserve_columns"`
	SourceLanguage         string                       `json:"source_language"`
	TargetLanguage         string                       `json:"target_language"`
	ImageDescription       string                       `json:"image_description,omitempty"`
	ProviderOptions        map[string]map[string]string `json:"provider_options,omitempty"`
	GUIPreferences         map[string]string            `json:"gui_preferences,omitempty"`
}

func Default() Config {
	return Config{
		Timeout:                30 * time.Second,
		DefaultProvider:        "mock",
		DefaultRenderer:        "svg",
		DefaultFontFamily:      "Noto Sans",
		DefaultFontSize:        18,
		DefaultFontWeight:      "normal",
		DefaultPageLayout:      string(base.PageLayoutAuto),
		OutputTemplate:         "{input_basename}_{provider}_{timestamp}.svg",
		OverlayColor:           "#111111",
		OverlayOpacity:         1,
		TextOutlineColor:       "#ffffff",
		TextOutlineWidth:       0,
		TextBackgroundEnabled:  false,
		TextBackgroundColor:    "#ffffff",
		TextBackgroundOpacity:  0.85,
		TextBackgroundPaddingX: 4,
		TextBackgroundPaddingY: 2,
		TextBackgroundRadius:   4,
		TextShadowEnabled:      false,
		TextShadowColor:        "#000000",
		TextShadowOpacity:      0.6,
		TextShadowBlur:         2,
		TextShadowOffsetX:      2,
		TextShadowOffsetY:      2,
		SourceLanguage:         "Ukrainian",
		TargetLanguage:         "German",
		ProviderOptions:        map[string]map[string]string{},
		GUIPreferences:         map[string]string{},
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
	if err == nil {
		return filepath.Join(userConfigDir, defaultConfigDir, "config.json"), nil
	}
	if filesDir := strings.TrimSpace(os.Getenv(filesDirEnv)); filesDir != "" {
		return filepath.Join(filesDir, "fyne", defaultConfigDir, "config.json"), nil
	}
	return "", fmt.Errorf("resolve user config dir: %w", err)
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

func (c Config) Clone() Config {
	cloned := c
	if c.ProviderOptions != nil {
		cloned.ProviderOptions = make(map[string]map[string]string, len(c.ProviderOptions))
		for providerName, options := range c.ProviderOptions {
			if options == nil {
				cloned.ProviderOptions[providerName] = nil
				continue
			}
			copiedOptions := make(map[string]string, len(options))
			for key, value := range options {
				copiedOptions[key] = value
			}
			cloned.ProviderOptions[providerName] = copiedOptions
		}
	}
	if c.GUIPreferences != nil {
		cloned.GUIPreferences = make(map[string]string, len(c.GUIPreferences))
		for key, value := range c.GUIPreferences {
			cloned.GUIPreferences[key] = value
		}
	}
	return cloned
}

func (c *Config) normalize() {
	defaults := Default()
	if c.Timeout <= 0 {
		c.Timeout = defaults.Timeout
	}
	if c.DefaultProvider == "" {
		c.DefaultProvider = defaults.DefaultProvider
	}
	if c.DefaultRenderer == "" {
		c.DefaultRenderer = defaults.DefaultRenderer
	}
	if c.DefaultFontFamily == "" {
		c.DefaultFontFamily = defaults.DefaultFontFamily
	}
	if c.DefaultFontSize <= 0 {
		c.DefaultFontSize = defaults.DefaultFontSize
	}
	if c.DefaultFontWeight == "" {
		c.DefaultFontWeight = defaults.DefaultFontWeight
	}
	pageLayout := strings.ToLower(strings.TrimSpace(c.DefaultPageLayout))
	if !base.IsValidPageLayout(base.PageLayout(pageLayout)) {
		c.DefaultPageLayout = defaults.DefaultPageLayout
	} else {
		c.DefaultPageLayout = pageLayout
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
	if c.TextOutlineColor == "" {
		c.TextOutlineColor = defaults.TextOutlineColor
	}
	if c.TextOutlineWidth < 0 {
		c.TextOutlineWidth = defaults.TextOutlineWidth
	}
	if c.TextBackgroundColor == "" {
		c.TextBackgroundColor = defaults.TextBackgroundColor
	}
	if c.TextBackgroundOpacity < 0 {
		c.TextBackgroundOpacity = defaults.TextBackgroundOpacity
	}
	if c.TextBackgroundPaddingX < 0 {
		c.TextBackgroundPaddingX = defaults.TextBackgroundPaddingX
	}
	if c.TextBackgroundPaddingY < 0 {
		c.TextBackgroundPaddingY = defaults.TextBackgroundPaddingY
	}
	if c.TextBackgroundRadius < 0 {
		c.TextBackgroundRadius = defaults.TextBackgroundRadius
	}
	if c.TextShadowColor == "" {
		c.TextShadowColor = defaults.TextShadowColor
	}
	if c.TextShadowOpacity < 0 {
		c.TextShadowOpacity = defaults.TextShadowOpacity
	}
	if c.TextShadowBlur < 0 {
		c.TextShadowBlur = defaults.TextShadowBlur
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
		EnvPrefix + "OPENAI_API_KEY":        &c.OpenAIAPIKey,
		EnvPrefix + "GEMINI_API_KEY":        &c.GeminiAPIKey,
		EnvPrefix + "DEFAULT_PROVIDER":      &c.DefaultProvider,
		EnvPrefix + "DEFAULT_RENDERER":      &c.DefaultRenderer,
		EnvPrefix + "DEFAULT_MODEL":         &c.DefaultModel,
		EnvPrefix + "DEFAULT_OUTPUT_DIR":    &c.DefaultOutputDir,
		EnvPrefix + "DEFAULT_FONT_FAMILY":   &c.DefaultFontFamily,
		EnvPrefix + "DEFAULT_FONT_WEIGHT":   &c.DefaultFontWeight,
		EnvPrefix + "DEFAULT_PAGE_LAYOUT":   &c.DefaultPageLayout,
		EnvPrefix + "OUTPUT_TEMPLATE":       &c.OutputTemplate,
		EnvPrefix + "OVERLAY_COLOR":         &c.OverlayColor,
		EnvPrefix + "TEXT_OUTLINE_COLOR":    &c.TextOutlineColor,
		EnvPrefix + "TEXT_BACKGROUND_COLOR": &c.TextBackgroundColor,
		EnvPrefix + "TEXT_SHADOW_COLOR":     &c.TextShadowColor,
		EnvPrefix + "SOURCE_LANGUAGE":       &c.SourceLanguage,
		EnvPrefix + "TARGET_LANGUAGE":       &c.TargetLanguage,
		EnvPrefix + "IMAGE_DESCRIPTION":     &c.ImageDescription,
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
	if value, ok := lookup(EnvPrefix + "TEXT_OUTLINE_WIDTH"); ok {
		outlineWidth, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"TEXT_OUTLINE_WIDTH", err)
		}
		c.TextOutlineWidth = outlineWidth
	}
	if value, ok := lookup(EnvPrefix + "TEXT_BACKGROUND_OPACITY"); ok {
		backgroundOpacity, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"TEXT_BACKGROUND_OPACITY", err)
		}
		c.TextBackgroundOpacity = backgroundOpacity
	}
	if value, ok := lookup(EnvPrefix + "TEXT_BACKGROUND_PADDING_X"); ok {
		paddingX, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"TEXT_BACKGROUND_PADDING_X", err)
		}
		c.TextBackgroundPaddingX = paddingX
	}
	if value, ok := lookup(EnvPrefix + "TEXT_BACKGROUND_PADDING_Y"); ok {
		paddingY, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"TEXT_BACKGROUND_PADDING_Y", err)
		}
		c.TextBackgroundPaddingY = paddingY
	}
	if value, ok := lookup(EnvPrefix + "TEXT_BACKGROUND_RADIUS"); ok {
		radius, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"TEXT_BACKGROUND_RADIUS", err)
		}
		c.TextBackgroundRadius = radius
	}
	if value, ok := lookup(EnvPrefix + "TEXT_SHADOW_OPACITY"); ok {
		shadowOpacity, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"TEXT_SHADOW_OPACITY", err)
		}
		c.TextShadowOpacity = shadowOpacity
	}
	if value, ok := lookup(EnvPrefix + "TEXT_SHADOW_BLUR"); ok {
		shadowBlur, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"TEXT_SHADOW_BLUR", err)
		}
		c.TextShadowBlur = shadowBlur
	}
	if value, ok := lookup(EnvPrefix + "TEXT_SHADOW_OFFSET_X"); ok {
		shadowOffsetX, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"TEXT_SHADOW_OFFSET_X", err)
		}
		c.TextShadowOffsetX = shadowOffsetX
	}
	if value, ok := lookup(EnvPrefix + "TEXT_SHADOW_OFFSET_Y"); ok {
		shadowOffsetY, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"TEXT_SHADOW_OFFSET_Y", err)
		}
		c.TextShadowOffsetY = shadowOffsetY
	}
	if value, ok := lookup(EnvPrefix + "PRESERVE_COLUMNS"); ok {
		preserveColumns, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"PRESERVE_COLUMNS", err)
		}
		c.PreserveColumns = preserveColumns
	}
	if value, ok := lookup(EnvPrefix + "TEXT_BACKGROUND_ENABLED"); ok {
		backgroundEnabled, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"TEXT_BACKGROUND_ENABLED", err)
		}
		c.TextBackgroundEnabled = backgroundEnabled
	}
	if value, ok := lookup(EnvPrefix + "TEXT_SHADOW_ENABLED"); ok {
		shadowEnabled, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse %s: %w", EnvPrefix+"TEXT_SHADOW_ENABLED", err)
		}
		c.TextShadowEnabled = shadowEnabled
	}
	return nil
}

func (c *Config) Set(key, value string) error {
	normalizedKey := strings.ToLower(key)
	if strings.HasPrefix(normalizedKey, "provider_options.") {
		return c.setProviderOption(normalizedKey, value)
	}

	switch normalizedKey {
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
	case "default_renderer":
		c.DefaultRenderer = value
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
	case "default_font_weight":
		c.DefaultFontWeight = value
	case "default_page_layout":
		c.DefaultPageLayout = value
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
	case "text_outline_color":
		c.TextOutlineColor = value
	case "text_outline_width":
		outlineWidth, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse text_outline_width: %w", err)
		}
		c.TextOutlineWidth = outlineWidth
	case "text_background_enabled":
		backgroundEnabled, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse text_background_enabled: %w", err)
		}
		c.TextBackgroundEnabled = backgroundEnabled
	case "text_background_color":
		c.TextBackgroundColor = value
	case "text_background_opacity":
		backgroundOpacity, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse text_background_opacity: %w", err)
		}
		c.TextBackgroundOpacity = backgroundOpacity
	case "text_background_padding_x":
		paddingX, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse text_background_padding_x: %w", err)
		}
		c.TextBackgroundPaddingX = paddingX
	case "text_background_padding_y":
		paddingY, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse text_background_padding_y: %w", err)
		}
		c.TextBackgroundPaddingY = paddingY
	case "text_background_radius":
		radius, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse text_background_radius: %w", err)
		}
		c.TextBackgroundRadius = radius
	case "text_shadow_enabled":
		shadowEnabled, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse text_shadow_enabled: %w", err)
		}
		c.TextShadowEnabled = shadowEnabled
	case "text_shadow_color":
		c.TextShadowColor = value
	case "text_shadow_opacity":
		shadowOpacity, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse text_shadow_opacity: %w", err)
		}
		c.TextShadowOpacity = shadowOpacity
	case "text_shadow_blur":
		shadowBlur, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse text_shadow_blur: %w", err)
		}
		c.TextShadowBlur = shadowBlur
	case "text_shadow_offset_x":
		shadowOffsetX, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse text_shadow_offset_x: %w", err)
		}
		c.TextShadowOffsetX = shadowOffsetX
	case "text_shadow_offset_y":
		shadowOffsetY, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse text_shadow_offset_y: %w", err)
		}
		c.TextShadowOffsetY = shadowOffsetY
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
	case "image_description":
		c.ImageDescription = value
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	c.normalize()
	return nil
}

func (c Config) RenderOptions() base.RenderOptions {
	return base.RenderOptions{
		FontFamily:         c.DefaultFontFamily,
		DefaultFontSize:    c.DefaultFontSize,
		PageLayout:         base.PageLayout(c.DefaultPageLayout),
		TextColor:          c.OverlayColor,
		Opacity:            c.OverlayOpacity,
		HasOpacity:         true,
		PreserveColumns:    c.PreserveColumns,
		FontWeight:         c.DefaultFontWeight,
		OutlineColor:       c.TextOutlineColor,
		OutlineWidth:       c.TextOutlineWidth,
		BackgroundEnabled:  c.TextBackgroundEnabled,
		BackgroundColor:    c.TextBackgroundColor,
		BackgroundOpacity:  c.TextBackgroundOpacity,
		BackgroundPaddingX: c.TextBackgroundPaddingX,
		BackgroundPaddingY: c.TextBackgroundPaddingY,
		BackgroundRadius:   c.TextBackgroundRadius,
		ShadowEnabled:      c.TextShadowEnabled,
		ShadowColor:        c.TextShadowColor,
		ShadowOpacity:      c.TextShadowOpacity,
		ShadowBlur:         c.TextShadowBlur,
		ShadowOffsetX:      c.TextShadowOffsetX,
		ShadowOffsetY:      c.TextShadowOffsetY,
	}
}

func (c Config) ProviderConfig(providerName, model string) provider.ProviderConfig {
	providerCfg := provider.ProviderConfig{
		DefaultModel:    model,
		AdvancedOptions: map[string]string{},
	}
	switch providerName {
	case "openai":
		providerCfg.APIKey = c.OpenAIAPIKey
	case "gemini":
		providerCfg.APIKey = c.GeminiAPIKey
	}
	for key, value := range c.ProviderOptions[providerName] {
		providerCfg.AdvancedOptions[key] = value
	}
	return providerCfg
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

func (c *Config) setProviderOption(key, value string) error {
	parts := strings.SplitN(key, ".", 3)
	if len(parts) != 3 || strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[2]) == "" {
		return fmt.Errorf("provider option key must be provider_options.<provider>.<option>")
	}
	c.normalize()
	if c.ProviderOptions[parts[1]] == nil {
		c.ProviderOptions[parts[1]] = map[string]string{}
	}
	c.ProviderOptions[parts[1]][parts[2]] = value
	return nil
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
