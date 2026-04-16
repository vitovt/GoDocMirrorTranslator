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
