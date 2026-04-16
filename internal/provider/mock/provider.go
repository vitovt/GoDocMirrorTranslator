package mock

import (
	"context"
	"fmt"

	"godocmirrortranslator/internal/domain"
	"godocmirrortranslator/internal/provider"
)

type Provider struct{}

func New(provider.ProviderConfig) *Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	return "mock"
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

	page := &domain.DocumentPage{
		SourceImagePath:   req.ImagePath,
		SourceImageWidth:  req.SourceImageWidth,
		SourceImageHeight: req.SourceImageHeight,
		Blocks: []domain.TextBlock{
			{
				ID:             "mock-title",
				SourceText:     fmt.Sprintf("[%s sample]", req.SourceLanguage),
				TranslatedText: fmt.Sprintf("[%s translation]", req.TargetLanguage),
				X:              float64(req.SourceImageWidth) * 0.15,
				Y:              float64(req.SourceImageHeight) * 0.2,
				Width:          float64(req.SourceImageWidth) * 0.7,
				Height:         float64(req.SourceImageHeight) * 0.12,
				FontSize:       32,
				FontFamily:     "Noto Sans",
				Color:          "#111111",
			},
			{
				ID:             "mock-body",
				SourceText:     "Це приклад текстового блоку.",
				TranslatedText: "Dies ist ein Beispieltextblock.",
				X:              float64(req.SourceImageWidth) * 0.18,
				Y:              float64(req.SourceImageHeight) * 0.42,
				Width:          float64(req.SourceImageWidth) * 0.64,
				Height:         float64(req.SourceImageHeight) * 0.16,
				FontSize:       24,
				FontFamily:     "Noto Sans",
				Color:          "#202020",
				LineHeight:     1.25,
			},
		},
		Metadata: map[string]string{
			"provider": p.Name(),
			"model":    req.Model,
		},
	}
	page.Normalize()
	return page, nil
}

func (p *Provider) ValidateConfig(provider.ProviderConfig) error {
	return nil
}

func (p *Provider) SupportedModels() []string {
	return []string{"mock-v1"}
}
