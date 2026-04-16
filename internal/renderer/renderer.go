package renderer

import (
	"context"

	"godocmirrortranslator/internal/domain"
)

type RenderOptions struct {
	FontFamily      string
	DefaultFontSize float64
	TextColor       string
	Opacity         float64
	PreserveColumns bool
}

type Renderer interface {
	Name() string
	Render(ctx context.Context, page *domain.DocumentPage, opts RenderOptions) ([]byte, error)
	FileExtension() string
}

func DefaultRenderOptions() RenderOptions {
	return RenderOptions{
		FontFamily:      "Noto Sans",
		DefaultFontSize: 18,
		TextColor:       "#111111",
		Opacity:         1,
	}
}

func (o RenderOptions) Normalized() RenderOptions {
	defaults := DefaultRenderOptions()
	if o.FontFamily == "" {
		o.FontFamily = defaults.FontFamily
	}
	if o.DefaultFontSize <= 0 {
		o.DefaultFontSize = defaults.DefaultFontSize
	}
	if o.TextColor == "" {
		o.TextColor = defaults.TextColor
	}
	if o.Opacity == 0 {
		o.Opacity = defaults.Opacity
	}
	return o
}
