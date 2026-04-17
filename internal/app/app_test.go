package app

import (
	"strings"
	"testing"

	"godocmirrortranslator/internal/provider"
)

func TestValidateProviderConfigRejectsUnknownProvider(t *testing.T) {
	application := New("test")
	err := application.ValidateProviderConfig("missing", provider.ProviderConfig{})
	if err == nil {
		t.Fatal("ValidateProviderConfig() error = nil, want unknown provider error")
	}
	if !strings.Contains(err.Error(), `unknown provider "missing"`) {
		t.Fatalf("ValidateProviderConfig() error = %v, want unknown provider error", err)
	}
}

func TestValidateProviderConfigUsesProviderValidation(t *testing.T) {
	application := New("test")
	err := application.ValidateProviderConfig("openai", provider.ProviderConfig{})
	if err == nil {
		t.Fatal("ValidateProviderConfig() error = nil, want provider validation error")
	}
	if !strings.Contains(err.Error(), "openai API key is required") {
		t.Fatalf("ValidateProviderConfig() error = %v, want openai API key validation", err)
	}
}

func TestProviderNamesReturnsSortedNames(t *testing.T) {
	application := New("test")
	names := application.ProviderNames()
	want := []string{"gemini", "mock", "openai"}
	if len(names) != len(want) {
		t.Fatalf("ProviderNames() = %#v, want %#v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("ProviderNames()[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestSupportedModelsUsesProviderFactoryAndHandlesUnknownProvider(t *testing.T) {
	application := New("test")
	models := application.SupportedModels("openai", provider.ProviderConfig{})
	if len(models) == 0 {
		t.Fatal("SupportedModels(openai) returned no models")
	}
	if models[0] != "gpt-4.1" {
		t.Fatalf("SupportedModels(openai) = %#v, want sorted OpenAI models", models)
	}
	if got := application.SupportedModels("missing", provider.ProviderConfig{}); got != nil {
		t.Fatalf("SupportedModels(missing) = %#v, want nil", got)
	}
}
