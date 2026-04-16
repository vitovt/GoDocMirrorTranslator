package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadReturnsDefaultsWhenConfigMissing(t *testing.T) {
	cfg, resolvedPath, err := Load(t.TempDir() + "/missing.json")
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
	if cfg.Timeout != defaults.Timeout {
		t.Fatalf("Timeout = %v, want %v", cfg.Timeout, defaults.Timeout)
	}
}

func TestSaveLoadAndMaskSecrets(t *testing.T) {
	path := t.TempDir() + "/config.json"
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
}

func TestSaveLoadPreservesZeroOverlayOpacity(t *testing.T) {
	path := t.TempDir() + "/config.json"
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

func TestApplyEnvOverridesDefaults(t *testing.T) {
	cfg := Default()
	env := map[string]string{
		EnvPrefix + "DEFAULT_PROVIDER":  "gemini",
		EnvPrefix + "DEFAULT_FONT_SIZE": "22.5",
		EnvPrefix + "OVERLAY_OPACITY":   "0.7",
		EnvPrefix + "PRESERVE_COLUMNS":  "true",
		EnvPrefix + "TIMEOUT":           "45s",
		EnvPrefix + "SOURCE_LANGUAGE":   "Polish",
		EnvPrefix + "TARGET_LANGUAGE":   "German",
		EnvPrefix + "OPENAI_API_KEY":    "env-openai",
		EnvPrefix + "GEMINI_API_KEY":    "env-gemini",
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
	if cfg.DefaultFontSize != 22.5 {
		t.Fatalf("DefaultFontSize = %v, want 22.5", cfg.DefaultFontSize)
	}
	if cfg.OverlayOpacity != 0.7 {
		t.Fatalf("OverlayOpacity = %v, want 0.7", cfg.OverlayOpacity)
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

func TestSaveLayoutJSONPreferenceDefaultsToDisabled(t *testing.T) {
	cfg := Default()
	if cfg.SaveLayoutJSONEnabled() {
		t.Fatal("SaveLayoutJSONEnabled() = true, want false by default")
	}
}

func TestSetSaveLayoutJSONEnabledPersistsBoolean(t *testing.T) {
	cfg := Default()
	cfg.SetSaveLayoutJSONEnabled(false)
	if cfg.SaveLayoutJSONEnabled() {
		t.Fatal("SaveLayoutJSONEnabled() = true, want false after update")
	}
}
