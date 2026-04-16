package gemini

import (
	"context"
	"fmt"

	"godocmirrortranslator/internal/domain"
	"godocmirrortranslator/internal/provider"
)

type Provider struct{}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	return "gemini"
}

func (p *Provider) AnalyzePage(context.Context, provider.AnalyzeRequest) (*domain.DocumentPage, error) {
	return nil, fmt.Errorf("gemini provider is not implemented yet")
}

func (p *Provider) ValidateConfig(provider.ProviderConfig) error {
	return nil
}

func (p *Provider) SupportedModels() []string {
	return nil
}
