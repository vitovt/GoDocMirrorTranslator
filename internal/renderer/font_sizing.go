package renderer

import (
	"sort"

	"godocmirrortranslator/internal/domain"
)

type FontSizer struct {
	mode          FontSizeMode
	defaultSizePt float64
	referenceSize float64
}

func NewFontSizer(page *domain.DocumentPage, opts RenderOptions) FontSizer {
	opts = opts.Normalized()
	return FontSizer{
		mode:          opts.FontSizeMode,
		defaultSizePt: opts.DefaultFontSize,
		referenceSize: medianBlockFontSize(page),
	}
}

func (s FontSizer) PointSize(block domain.TextBlock) float64 {
	sizePt := s.defaultSizePt
	if s.mode == FontSizeModeProportional &&
		s.referenceSize > 0 &&
		block.FontSize > 0 {
		sizePt = block.FontSize * (s.defaultSizePt / s.referenceSize)
	}
	if sizePt <= 0 {
		return DefaultRenderOptions().DefaultFontSize
	}
	return sizePt
}

func (s FontSizer) MillimeterSize(block domain.TextBlock) float64 {
	return s.PointSize(block) * MillimetersPerPoint
}

func medianBlockFontSize(page *domain.DocumentPage) float64 {
	if page == nil || len(page.Blocks) == 0 {
		return 0
	}

	values := make([]float64, 0, len(page.Blocks))
	for _, block := range page.Blocks {
		if block.FontSize > 0 {
			values = append(values, block.FontSize)
		}
	}
	if len(values) == 0 {
		return 0
	}

	sort.Float64s(values)
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return (values[mid-1] + values[mid]) / 2
}
