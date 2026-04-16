package gemini

import (
	"context"
	"fmt"
	"strings"

	"godocmirrortranslator/internal/domain"
	"godocmirrortranslator/internal/provider"
)

type Provider struct {
	cfg provider.ProviderConfig
}

func New(cfg provider.ProviderConfig) *Provider {
	return &Provider{cfg: cfg}
}

func (p *Provider) Name() string {
	return "gemini"
}

func (p *Provider) AnalyzePage(ctx context.Context, req provider.AnalyzeRequest) (*domain.DocumentPage, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return nil, fmt.Errorf("gemini provider is not implemented yet")
}

func (p *Provider) ValidateConfig(cfg provider.ProviderConfig) error {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return fmt.Errorf("gemini API key is required")
	}
	return nil
}

func (p *Provider) SupportedModels() []string {
	return nil
}
