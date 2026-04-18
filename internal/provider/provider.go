package provider

import (
	"context"
	"fmt"
	"time"

	"godocmirrortranslator/internal/domain"
)

type AnalyzeRequest struct {
	ImagePath         string
	ImageBytes        []byte
	SourceImageWidth  int
	SourceImageHeight int
	SourceLanguage    string
	TargetLanguage    string
	ImageDescription  string
	Model             string
	Timeout           time.Duration
}

type ProviderConfig struct {
	APIKey          string
	DefaultModel    string
	AdvancedOptions map[string]string
}

type Provider interface {
	Name() string
	AnalyzePage(ctx context.Context, req AnalyzeRequest) (*domain.DocumentPage, error)
	ValidateConfig(cfg ProviderConfig) error
	SupportedModels() []string
}

func (r AnalyzeRequest) Validate() error {
	if r.ImagePath == "" && len(r.ImageBytes) == 0 {
		return fmt.Errorf("input image is required")
	}
	if r.SourceImageWidth <= 0 || r.SourceImageHeight <= 0 {
		return fmt.Errorf("source image dimensions must be positive")
	}
	if r.SourceLanguage == "" {
		return fmt.Errorf("source language is required")
	}
	if r.TargetLanguage == "" {
		return fmt.Errorf("target language is required")
	}
	return nil
}
