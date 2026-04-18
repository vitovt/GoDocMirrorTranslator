package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	base "godocmirrortranslator/internal/renderer"
)

func TestResolvePathPrefersExplicitAndEnvThenUserConfigDir(t *testing.T) {
	explicit := filepath.Join(t.TempDir(), "explicit.json")
	if resolved, err := ResolvePath(explicit); err != nil {
		t.Fatalf("ResolvePath(explicit) error = %v", err)
	} else if resolved != explicit {
		t.Fatalf("ResolvePath(explicit) = %q, want %q", resolved, explicit)
	}

	envPath := filepath.Join(t.TempDir(), "from-env.json")
	t.Setenv(ConfigPathEnv, envPath)
	if resolved, err := ResolvePath(""); err != nil {
		t.Fatalf("ResolvePath(from env) error = %v", err)
	} else if resolved != envPath {
		t.Fatalf("ResolvePath(from env) = %q, want %q", resolved, envPath)
	}

	t.Setenv(ConfigPathEnv, "")
	configHome := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("AppData", configHome)
	} else {
		t.Setenv("XDG_CONFIG_HOME", configHome)
	}
	resolved, err := ResolvePath("")
	if err != nil {
		t.Fatalf("ResolvePath(default) error = %v", err)
	}
	want := filepath.Join(configHome, defaultConfigDir, "config.json")
	if resolved != want {
		t.Fatalf("ResolvePath(default) = %q, want %q", resolved, want)
	}
}

func TestResolvePathFallsBackToFilesDirWhenUserConfigDirUnavailable(t *testing.T) {
	t.Setenv(ConfigPathEnv, "")
	if runtime.GOOS == "windows" {
		t.Setenv("AppData", "")
	} else {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "")
	}

	filesDir := filepath.Join(t.TempDir(), "files")
	t.Setenv(filesDirEnv, filesDir)

	resolved, err := ResolvePath("")
	if err != nil {
		t.Fatalf("ResolvePath(files dir fallback) error = %v", err)
	}

	want := filepath.Join(filesDir, "fyne", defaultConfigDir, "config.json")
	if resolved != want {
		t.Fatalf("ResolvePath(files dir fallback) = %q, want %q", resolved, want)
	}
}

func TestLoadReturnsDefaultsWhenConfigMissing(t *testing.T) {
	cfg, resolvedPath, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if resolvedPath == "" {
		t.Fatal("Load() returned empty resolved path")
	}
	defaults := Default()
	if cfg.DefaultProvider != defaults.DefaultProvider {
		t.Fatalf("DefaultProvider = %q, want %q", cfg.DefaultProvider, defaults.DefaultProvider)
	}
	if cfg.DefaultRenderer != defaults.DefaultRenderer {
		t.Fatalf("DefaultRenderer = %q, want %q", cfg.DefaultRenderer, defaults.DefaultRenderer)
	}
	if cfg.Timeout != defaults.Timeout {
		t.Fatalf("Timeout = %v, want %v", cfg.Timeout, defaults.Timeout)
	}
}

func TestLoadReturnsDefaultsWhenConfigIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	cfg, resolvedPath, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if resolvedPath != path {
		t.Fatalf("resolvedPath = %q, want %q", resolvedPath, path)
	}
	if cfg.DefaultProvider != Default().DefaultProvider {
		t.Fatalf("DefaultProvider = %q, want %q", cfg.DefaultProvider, Default().DefaultProvider)
	}
}

func TestLoadRejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	if _, _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want decode config error")
	} else if !strings.Contains(err.Error(), "decode config") {
		t.Fatalf("Load() error = %v, want decode config error", err)
	}
}

func TestSaveLoadAndMaskSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := Default()
	cfg.OpenAIAPIKey = "openai-secret"
	cfg.GeminiAPIKey = "gemini-secret"
	cfg.DefaultProvider = "openai"

	if _, err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, _, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.DefaultProvider != "openai" {
		t.Fatalf("DefaultProvider = %q, want %q", loaded.DefaultProvider, "openai")
	}
	masked := loaded.Masked()
	if masked.OpenAIAPIKey == loaded.OpenAIAPIKey || !strings.Contains(masked.OpenAIAPIKey, "*") {
		t.Fatalf("Masked OpenAI key = %q, want masked value", masked.OpenAIAPIKey)
	}
	if masked.GeminiAPIKey == loaded.GeminiAPIKey || !strings.Contains(masked.GeminiAPIKey, "*") {
		t.Fatalf("Masked Gemini key = %q, want masked value", masked.GeminiAPIKey)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %o, want 600", info.Mode().Perm())
	}
}

func TestSaveLoadPreservesZeroOverlayOpacity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := Default()
	cfg.OverlayOpacity = 0

	if _, err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, _, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.OverlayOpacity != 0 {
		t.Fatalf("OverlayOpacity = %v, want 0", loaded.OverlayOpacity)
	}
}

func TestLoadEffectiveAppliesEnvOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := Default()
	cfg.DefaultProvider = "openai"
	cfg.DefaultRenderer = "fodg"
	cfg.DefaultModel = "from-config"
	cfg.OutputTemplate = "from-config.svg"
	if _, err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	t.Setenv(EnvPrefix+"DEFAULT_PROVIDER", "gemini")
	t.Setenv(EnvPrefix+"DEFAULT_RENDERER", "svg")
	t.Setenv(EnvPrefix+"DEFAULT_MODEL", "from-env")
	t.Setenv(EnvPrefix+"OUTPUT_TEMPLATE", "from-env.svg")

	loaded, resolvedPath, err := LoadEffective(path)
	if err != nil {
		t.Fatalf("LoadEffective() error = %v", err)
	}
	if resolvedPath != path {
		t.Fatalf("resolvedPath = %q, want %q", resolvedPath, path)
	}
	if loaded.DefaultProvider != "gemini" || loaded.DefaultRenderer != "svg" || loaded.DefaultModel != "from-env" || loaded.OutputTemplate != "from-env.svg" {
		t.Fatalf("LoadEffective() = %#v, want env overrides applied", loaded)
	}
}

func TestApplyEnvOverridesDefaults(t *testing.T) {
	cfg := Default()
	env := map[string]string{
		EnvPrefix + "DEFAULT_PROVIDER":        "gemini",
		EnvPrefix + "DEFAULT_RENDERER":        "fodg",
		EnvPrefix + "DEFAULT_FONT_SIZE":       "22.5",
		EnvPrefix + "DEFAULT_FONT_WEIGHT":     "bold",
		EnvPrefix + "DEFAULT_PAGE_LAYOUT":     "landscape",
		EnvPrefix + "OVERLAY_OPACITY":         "0.7",
		EnvPrefix + "TEXT_BACKGROUND_ENABLED": "true",
		EnvPrefix + "TEXT_SHADOW_ENABLED":     "true",
		EnvPrefix + "PRESERVE_COLUMNS":        "true",
		EnvPrefix + "TIMEOUT":                 "45s",
		EnvPrefix + "SOURCE_LANGUAGE":         "Polish",
		EnvPrefix + "TARGET_LANGUAGE":         "German",
		EnvPrefix + "OPENAI_API_KEY":          "env-openai",
		EnvPrefix + "GEMINI_API_KEY":          "env-gemini",
	}
	lookup := func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	}
	if err := cfg.ApplyEnv(lookup); err != nil {
		t.Fatalf("ApplyEnv() error = %v", err)
	}
	if cfg.DefaultProvider != "gemini" {
		t.Fatalf("DefaultProvider = %q, want gemini", cfg.DefaultProvider)
	}
	if cfg.DefaultRenderer != "fodg" {
		t.Fatalf("DefaultRenderer = %q, want fodg", cfg.DefaultRenderer)
	}
	if cfg.DefaultFontSize != 22.5 {
		t.Fatalf("DefaultFontSize = %v, want 22.5", cfg.DefaultFontSize)
	}
	if cfg.DefaultFontWeight != "bold" {
		t.Fatalf("DefaultFontWeight = %q, want bold", cfg.DefaultFontWeight)
	}
	if cfg.DefaultPageLayout != "landscape" {
		t.Fatalf("DefaultPageLayout = %q, want landscape", cfg.DefaultPageLayout)
	}
	if cfg.OverlayOpacity != 0.7 {
		t.Fatalf("OverlayOpacity = %v, want 0.7", cfg.OverlayOpacity)
	}
	if !cfg.TextBackgroundEnabled || !cfg.TextShadowEnabled {
		t.Fatalf("text readability toggles = background:%v shadow:%v, want both true", cfg.TextBackgroundEnabled, cfg.TextShadowEnabled)
	}
	if !cfg.PreserveColumns {
		t.Fatal("PreserveColumns = false, want true")
	}
	if cfg.Timeout != 45*time.Second {
		t.Fatalf("Timeout = %v, want 45s", cfg.Timeout)
	}
	if cfg.SourceLanguage != "Polish" || cfg.TargetLanguage != "German" {
		t.Fatalf("languages = %q -> %q, want Polish -> German", cfg.SourceLanguage, cfg.TargetLanguage)
	}
}

func TestApplyEnvRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: EnvPrefix + "TIMEOUT",
			env:  map[string]string{EnvPrefix + "TIMEOUT": "not-a-duration"},
			want: "parse " + EnvPrefix + "TIMEOUT",
		},
		{
			name: EnvPrefix + "DEFAULT_FONT_SIZE",
			env:  map[string]string{EnvPrefix + "DEFAULT_FONT_SIZE": "oops"},
			want: "parse " + EnvPrefix + "DEFAULT_FONT_SIZE",
		},
		{
			name: EnvPrefix + "OVERLAY_OPACITY",
			env:  map[string]string{EnvPrefix + "OVERLAY_OPACITY": "oops"},
			want: "parse " + EnvPrefix + "OVERLAY_OPACITY",
		},
		{
			name: EnvPrefix + "PRESERVE_COLUMNS",
			env:  map[string]string{EnvPrefix + "PRESERVE_COLUMNS": "oops"},
			want: "parse " + EnvPrefix + "PRESERVE_COLUMNS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			lookup := func(key string) (string, bool) {
				value, ok := tt.env[key]
				return value, ok
			}
			err := cfg.ApplyEnv(lookup)
			if err == nil {
				t.Fatal("ApplyEnv() error = nil, want parse failure")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ApplyEnv() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestSetSupportsKnownKeysAndRejectsUnknownKeys(t *testing.T) {
	tests := []struct {
		key   string
		value string
		check func(t *testing.T, cfg Config)
	}{
		{
			key:   "openai_api_key",
			value: "sk-test",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.OpenAIAPIKey != "sk-test" {
					t.Fatalf("OpenAIAPIKey = %q, want sk-test", cfg.OpenAIAPIKey)
				}
			},
		},
		{
			key:   "gemini_api_key",
			value: "gm-test",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.GeminiAPIKey != "gm-test" {
					t.Fatalf("GeminiAPIKey = %q, want gm-test", cfg.GeminiAPIKey)
				}
			},
		},
		{
			key:   "timeout",
			value: "90s",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.Timeout != 90*time.Second {
					t.Fatalf("Timeout = %v, want 90s", cfg.Timeout)
				}
			},
		},
		{
			key:   "default_provider",
			value: "gemini",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.DefaultProvider != "gemini" {
					t.Fatalf("DefaultProvider = %q, want gemini", cfg.DefaultProvider)
				}
			},
		},
		{
			key:   "default_renderer",
			value: "fodg",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.DefaultRenderer != "fodg" {
					t.Fatalf("DefaultRenderer = %q, want fodg", cfg.DefaultRenderer)
				}
			},
		},
		{
			key:   "default_model",
			value: "gemini-2.5-flash",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.DefaultModel != "gemini-2.5-flash" {
					t.Fatalf("DefaultModel = %q, want gemini-2.5-flash", cfg.DefaultModel)
				}
			},
		},
		{
			key:   "default_output_dir",
			value: "/tmp/out",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.DefaultOutputDir != "/tmp/out" {
					t.Fatalf("DefaultOutputDir = %q, want /tmp/out", cfg.DefaultOutputDir)
				}
			},
		},
		{
			key:   "default_font_family",
			value: "Fira Sans",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.DefaultFontFamily != "Fira Sans" {
					t.Fatalf("DefaultFontFamily = %q, want Fira Sans", cfg.DefaultFontFamily)
				}
			},
		},
		{
			key:   "default_font_size",
			value: "23.5",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.DefaultFontSize != 23.5 {
					t.Fatalf("DefaultFontSize = %v, want 23.5", cfg.DefaultFontSize)
				}
			},
		},
		{
			key:   "default_font_weight",
			value: "bold",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.DefaultFontWeight != "bold" {
					t.Fatalf("DefaultFontWeight = %q, want bold", cfg.DefaultFontWeight)
				}
			},
		},
		{
			key:   "default_page_layout",
			value: "portrait",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.DefaultPageLayout != "portrait" {
					t.Fatalf("DefaultPageLayout = %q, want portrait", cfg.DefaultPageLayout)
				}
			},
		},
		{
			key:   "output_template",
			value: "custom.svg",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.OutputTemplate != "custom.svg" {
					t.Fatalf("OutputTemplate = %q, want custom.svg", cfg.OutputTemplate)
				}
			},
		},
		{
			key:   "overlay_color",
			value: "#abcdef",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.OverlayColor != "#abcdef" {
					t.Fatalf("OverlayColor = %q, want #abcdef", cfg.OverlayColor)
				}
			},
		},
		{
			key:   "overlay_opacity",
			value: "0.25",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.OverlayOpacity != 0.25 {
					t.Fatalf("OverlayOpacity = %v, want 0.25", cfg.OverlayOpacity)
				}
			},
		},
		{
			key:   "text_outline_width",
			value: "1.5",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.TextOutlineWidth != 1.5 {
					t.Fatalf("TextOutlineWidth = %v, want 1.5", cfg.TextOutlineWidth)
				}
			},
		},
		{
			key:   "text_background_enabled",
			value: "true",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if !cfg.TextBackgroundEnabled {
					t.Fatal("TextBackgroundEnabled = false, want true")
				}
			},
		},
		{
			key:   "text_shadow_enabled",
			value: "true",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if !cfg.TextShadowEnabled {
					t.Fatal("TextShadowEnabled = false, want true")
				}
			},
		},
		{
			key:   "preserve_columns",
			value: "true",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if !cfg.PreserveColumns {
					t.Fatal("PreserveColumns = false, want true")
				}
			},
		},
		{
			key:   "source_language",
			value: "Polish",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.SourceLanguage != "Polish" {
					t.Fatalf("SourceLanguage = %q, want Polish", cfg.SourceLanguage)
				}
			},
		},
		{
			key:   "target_language",
			value: "French",
			check: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.TargetLanguage != "French" {
					t.Fatalf("TargetLanguage = %q, want French", cfg.TargetLanguage)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			cfg := Default()
			if err := cfg.Set(tt.key, tt.value); err != nil {
				t.Fatalf("Set(%q) error = %v", tt.key, err)
			}
			tt.check(t, cfg)
		})
	}

	for _, tt := range []struct {
		key   string
		value string
		want  string
	}{
		{key: "timeout", value: "bad", want: "parse timeout"},
		{key: "default_font_size", value: "bad", want: "parse default_font_size"},
		{key: "overlay_opacity", value: "bad", want: "parse overlay_opacity"},
		{key: "text_outline_width", value: "bad", want: "parse text_outline_width"},
		{key: "text_background_enabled", value: "bad", want: "parse text_background_enabled"},
		{key: "text_shadow_enabled", value: "bad", want: "parse text_shadow_enabled"},
		{key: "preserve_columns", value: "bad", want: "parse preserve_columns"},
		{key: "unknown_key", value: "value", want: "unknown config key"},
	} {
		t.Run("error_"+tt.key, func(t *testing.T) {
			cfg := Default()
			err := cfg.Set(tt.key, tt.value)
			if err == nil {
				t.Fatal("Set() error = nil, want failure")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Set() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestSetProviderOptionCreatesNestedMap(t *testing.T) {
	cfg := Default()
	if err := cfg.Set("provider_options.openai.image_detail", "high"); err != nil {
		t.Fatalf("Set(provider option) error = %v", err)
	}
	if cfg.ProviderOptions["openai"]["image_detail"] != "high" {
		t.Fatalf("ProviderOptions = %#v, want openai image_detail=high", cfg.ProviderOptions)
	}
}

func TestCloneDeepCopiesNestedMaps(t *testing.T) {
	cfg := Default()
	cfg.ProviderOptions["openai"] = map[string]string{"image_detail": "high"}
	cfg.GUIPreferences["menu"] = "visible"

	cloned := cfg.Clone()
	cloned.ProviderOptions["openai"]["image_detail"] = "low"
	cloned.GUIPreferences["menu"] = "hidden"

	if cfg.ProviderOptions["openai"]["image_detail"] != "high" {
		t.Fatalf("original ProviderOptions changed to %#v, want preserved value", cfg.ProviderOptions)
	}
	if cfg.GUIPreferences["menu"] != "visible" {
		t.Fatalf("original GUIPreferences changed to %#v, want preserved value", cfg.GUIPreferences)
	}
}

func TestSetProviderOptionRejectsInvalidKey(t *testing.T) {
	cfg := Default()
	if err := cfg.Set("provider_options.openai", "high"); err == nil {
		t.Fatal("Set(invalid provider option) error = nil, want validation error")
	} else if !strings.Contains(err.Error(), "provider option key must be provider_options.<provider>.<option>") {
		t.Fatalf("Set(invalid provider option) error = %v", err)
	}
}

func TestRenderOptionsAndProviderConfigMirrorConfig(t *testing.T) {
	cfg := Default()
	cfg.DefaultFontFamily = "Fira Sans"
	cfg.DefaultFontSize = 19
	cfg.DefaultFontWeight = "bold"
	cfg.DefaultPageLayout = "portrait"
	cfg.OverlayColor = "#334455"
	cfg.OverlayOpacity = 0.6
	cfg.TextOutlineColor = "#ffffff"
	cfg.TextOutlineWidth = 2
	cfg.TextBackgroundEnabled = true
	cfg.TextBackgroundColor = "#fefefe"
	cfg.TextBackgroundOpacity = 0.8
	cfg.TextBackgroundPaddingX = 5
	cfg.TextBackgroundPaddingY = 3
	cfg.TextBackgroundRadius = 6
	cfg.TextShadowEnabled = true
	cfg.TextShadowColor = "#101010"
	cfg.TextShadowOpacity = 0.4
	cfg.TextShadowBlur = 3
	cfg.TextShadowOffsetX = 2
	cfg.TextShadowOffsetY = 1
	cfg.PreserveColumns = true
	cfg.OpenAIAPIKey = "sk-openai"
	cfg.GeminiAPIKey = "gm-gemini"
	cfg.ProviderOptions["openai"] = map[string]string{"image_detail": "high"}
	cfg.ProviderOptions["gemini"] = map[string]string{"temperature": "0"}

	renderOpts := cfg.RenderOptions()
	if renderOpts.FontFamily != "Fira Sans" ||
		renderOpts.DefaultFontSize != 19 ||
		renderOpts.FontWeight != "bold" ||
		renderOpts.PageLayout != base.PageLayoutPortrait ||
		renderOpts.TextColor != "#334455" ||
		renderOpts.Opacity != 0.6 ||
		!renderOpts.HasOpacity ||
		renderOpts.OutlineColor != "#ffffff" ||
		renderOpts.OutlineWidth != 2 ||
		!renderOpts.BackgroundEnabled ||
		renderOpts.BackgroundColor != "#fefefe" ||
		renderOpts.BackgroundOpacity != 0.8 ||
		renderOpts.BackgroundPaddingX != 5 ||
		renderOpts.BackgroundPaddingY != 3 ||
		renderOpts.BackgroundRadius != 6 ||
		!renderOpts.ShadowEnabled ||
		renderOpts.ShadowColor != "#101010" ||
		renderOpts.ShadowOpacity != 0.4 ||
		renderOpts.ShadowBlur != 3 ||
		renderOpts.ShadowOffsetX != 2 ||
		renderOpts.ShadowOffsetY != 1 ||
		!renderOpts.PreserveColumns {
		t.Fatalf("RenderOptions() = %#v, want mirrored render settings", renderOpts)
	}

	openaiCfg := cfg.ProviderConfig("openai", "gpt-4.1-mini")
	if openaiCfg.APIKey != "sk-openai" || openaiCfg.DefaultModel != "gpt-4.1-mini" || openaiCfg.AdvancedOptions["image_detail"] != "high" {
		t.Fatalf("ProviderConfig(openai) = %#v, want API key/model/options", openaiCfg)
	}
	geminiCfg := cfg.ProviderConfig("gemini", "gemini-2.5-flash")
	if geminiCfg.APIKey != "gm-gemini" || geminiCfg.AdvancedOptions["temperature"] != "0" {
		t.Fatalf("ProviderConfig(gemini) = %#v, want API key/options", geminiCfg)
	}
	unknownCfg := cfg.ProviderConfig("mock", "mock-v1")
	if unknownCfg.APIKey != "" || unknownCfg.DefaultModel != "mock-v1" || len(unknownCfg.AdvancedOptions) != 0 {
		t.Fatalf("ProviderConfig(mock) = %#v, want empty API key and options", unknownCfg)
	}
}

func TestMaskSecretHandlesShortValues(t *testing.T) {
	if got := maskSecret(""); got != "" {
		t.Fatalf("maskSecret(\"\") = %q, want empty string", got)
	}
	if got := maskSecret("abcd"); got != "****" {
		t.Fatalf("maskSecret(\"abcd\") = %q, want ****", got)
	}
}

func TestNormalizeAndPreferenceHelpersOnZeroConfig(t *testing.T) {
	var cfg Config
	cfg.normalize()

	defaults := Default()
	if cfg.Timeout != defaults.Timeout || cfg.DefaultProvider != defaults.DefaultProvider || cfg.DefaultRenderer != defaults.DefaultRenderer || cfg.DefaultFontFamily != defaults.DefaultFontFamily || cfg.DefaultFontSize != defaults.DefaultFontSize {
		t.Fatalf("normalize() did not apply defaults: %#v", cfg)
	}
	if cfg.DefaultPageLayout != defaults.DefaultPageLayout {
		t.Fatalf("DefaultPageLayout = %q, want %q", cfg.DefaultPageLayout, defaults.DefaultPageLayout)
	}
	if cfg.OutputTemplate != defaults.OutputTemplate || cfg.OverlayColor != defaults.OverlayColor {
		t.Fatalf("normalize() did not apply output defaults: %#v", cfg)
	}
	if cfg.OverlayOpacity != 0 {
		t.Fatalf("OverlayOpacity = %v, want preserved zero value", cfg.OverlayOpacity)
	}
	if cfg.SourceLanguage != defaults.SourceLanguage || cfg.TargetLanguage != defaults.TargetLanguage {
		t.Fatalf("normalize() did not apply language defaults: %#v", cfg)
	}
	if cfg.ProviderOptions == nil || cfg.GUIPreferences == nil {
		t.Fatalf("normalize() did not initialize maps: %#v", cfg)
	}
}
