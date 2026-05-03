package renderer

import (
	"context"

	"godocmirrortranslator/internal/domain"
)

const MillimetersPerPoint = 25.4 / 72.0

type PageLayout string

const (
	PageLayoutAuto      PageLayout = "auto"
	PageLayoutPortrait  PageLayout = "portrait"
	PageLayoutLandscape PageLayout = "landscape"
)

type FontSizeMode string

const (
	FontSizeModeUnisizefont  FontSizeMode = "unisizefont"
	FontSizeModeProportional FontSizeMode = "proportional"
)

type RenderOptions struct {
	FontFamily         string
	DefaultFontSize    float64
	FontSizeMode       FontSizeMode
	PageLayout         PageLayout
	TextColor          string
	Opacity            float64
	HasOpacity         bool
	PreserveColumns    bool
	FontWeight         string
	OutlineColor       string
	OutlineWidth       float64
	BackgroundEnabled  bool
	BackgroundColor    string
	BackgroundOpacity  float64
	BackgroundPaddingX float64
	BackgroundPaddingY float64
	BackgroundRadius   float64
	ShadowEnabled      bool
	ShadowColor        string
	ShadowOpacity      float64
	ShadowBlur         float64
	ShadowOffsetX      float64
	ShadowOffsetY      float64
}

type Renderer interface {
	Name() string
	Render(ctx context.Context, page *domain.DocumentPage, opts RenderOptions) ([]byte, error)
	FileExtension() string
}

func DefaultRenderOptions() RenderOptions {
	return RenderOptions{
		FontFamily:         "Noto Sans",
		DefaultFontSize:    18,
		FontSizeMode:       FontSizeModeUnisizefont,
		PageLayout:         PageLayoutAuto,
		TextColor:          "#111111",
		Opacity:            1,
		HasOpacity:         true,
		FontWeight:         "normal",
		OutlineColor:       "#ffffff",
		OutlineWidth:       0,
		BackgroundEnabled:  false,
		BackgroundColor:    "#ffffff",
		BackgroundOpacity:  0.85,
		BackgroundPaddingX: 4,
		BackgroundPaddingY: 2,
		BackgroundRadius:   4,
		ShadowEnabled:      false,
		ShadowColor:        "#000000",
		ShadowOpacity:      0.6,
		ShadowBlur:         2,
		ShadowOffsetX:      2,
		ShadowOffsetY:      2,
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
	if !IsValidFontSizeMode(o.FontSizeMode) {
		o.FontSizeMode = defaults.FontSizeMode
	}
	if !IsValidPageLayout(o.PageLayout) {
		o.PageLayout = defaults.PageLayout
	}
	if o.TextColor == "" {
		o.TextColor = defaults.TextColor
	}
	if !o.HasOpacity {
		o.Opacity = defaults.Opacity
		o.HasOpacity = defaults.HasOpacity
	}
	if o.FontWeight != "normal" && o.FontWeight != "bold" {
		o.FontWeight = defaults.FontWeight
	}
	if o.OutlineColor == "" {
		o.OutlineColor = defaults.OutlineColor
	}
	if o.OutlineWidth < 0 {
		o.OutlineWidth = defaults.OutlineWidth
	}
	if o.BackgroundColor == "" {
		o.BackgroundColor = defaults.BackgroundColor
	}
	if o.BackgroundOpacity < 0 {
		o.BackgroundOpacity = defaults.BackgroundOpacity
	}
	if o.BackgroundPaddingX < 0 {
		o.BackgroundPaddingX = defaults.BackgroundPaddingX
	}
	if o.BackgroundPaddingY < 0 {
		o.BackgroundPaddingY = defaults.BackgroundPaddingY
	}
	if o.BackgroundRadius < 0 {
		o.BackgroundRadius = defaults.BackgroundRadius
	}
	if o.ShadowColor == "" {
		o.ShadowColor = defaults.ShadowColor
	}
	if o.ShadowOpacity < 0 {
		o.ShadowOpacity = defaults.ShadowOpacity
	}
	if o.ShadowBlur < 0 {
		o.ShadowBlur = defaults.ShadowBlur
	}
	return o
}

func IsValidPageLayout(layout PageLayout) bool {
	switch layout {
	case PageLayoutAuto, PageLayoutPortrait, PageLayoutLandscape:
		return true
	default:
		return false
	}
}

func IsValidFontSizeMode(mode FontSizeMode) bool {
	switch mode {
	case FontSizeModeUnisizefont, FontSizeModeProportional:
		return true
	default:
		return false
	}
}
