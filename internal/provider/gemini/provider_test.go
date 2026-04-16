package gemini

import (
	"testing"

	"godocmirrortranslator/internal/provider"
)

func TestValidateConfigRequiresAPIKey(t *testing.T) {
	p := New(provider.ProviderConfig{})
	if err := p.ValidateConfig(provider.ProviderConfig{}); err == nil {
		t.Fatal("ValidateConfig() error = nil, want missing API key error")
	}
}

func TestValidateConfigAcceptsAPIKey(t *testing.T) {
	p := New(provider.ProviderConfig{})
	if err := p.ValidateConfig(provider.ProviderConfig{APIKey: "test-key"}); err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}
}
