package svg

import (
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"net/http"
	"os"
	"strings"

	"godocmirrortranslator/internal/domain"
	base "godocmirrortranslator/internal/renderer"
)

const (
	portraitWidthMM  = 210.0
	portraitHeightMM = 297.0
)

type Renderer struct{}

func New() *Renderer {
	return &Renderer{}
}

func (r *Renderer) Name() string {
	return "svg"
}

func (r *Renderer) FileExtension() string {
	return ".svg"
}

func (r *Renderer) Render(ctx context.Context, page *domain.DocumentPage, opts base.RenderOptions) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	page.Normalize()
	if err := page.Validate(); err != nil {
		return nil, fmt.Errorf("validate page: %w", err)
	}
	opts = opts.Normalized()

	imageBytes, err := os.ReadFile(page.SourceImagePath)
	if err != nil {
		return nil, fmt.Errorf("read source image: %w", err)
	}

	pageWidth, pageHeight := a4Dimensions(page.Orientation)
	scale := fitScale(pageWidth, pageHeight, float64(page.SourceImageWidth), float64(page.SourceImageHeight))
	imageWidth := float64(page.SourceImageWidth) * scale
	imageHeight := float64(page.SourceImageHeight) * scale
	offsetX := (pageWidth - imageWidth) / 2
	offsetY := (pageHeight - imageHeight) / 2
	mimeType := http.DetectContentType(imageBytes)

	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString(fmt.Sprintf("<svg xmlns=\"http://www.w3.org/2000/svg\" xmlns:xlink=\"http://www.w3.org/1999/xlink\" width=\"%.2fmm\" height=\"%.2fmm\" viewBox=\"0 0 %.2f %.2f\">\n", pageWidth, pageHeight, pageWidth, pageHeight))
	b.WriteString(fmt.Sprintf("  <image x=\"%.4f\" y=\"%.4f\" width=\"%.4f\" height=\"%.4f\" href=\"data:%s;base64,%s\" />\n", offsetX, offsetY, imageWidth, imageHeight, mimeType, base64.StdEncoding.EncodeToString(imageBytes)))

	for _, block := range page.Blocks {
		text := block.TranslatedText
		if text == "" {
			text = block.SourceText
		}
		fontFamily := block.FontFamily
		if fontFamily == "" {
			fontFamily = opts.FontFamily
		}
		fontSize := block.FontSize * scale
		if fontSize <= 0 {
			fontSize = opts.DefaultFontSize * scale
		}
		if fontSize < 2.5 {
			fontSize = 2.5
		}
		color := block.Color
		if color == "" {
			color = opts.TextColor
		}
		opacity := block.Opacity
		if block.Opacity == 0 && opts.HasOpacity {
			opacity = opts.Opacity
		}
		lineHeight := block.LineHeight
		if lineHeight <= 0 {
			lineHeight = domain.DefaultLineHeight
		}
		x := offsetX + (block.X * scale)
		y := offsetY + (block.Y * scale)
		anchor := textAnchor(block.Align)
		transform := ""
		if block.Rotation != 0 {
			transform = fmt.Sprintf(" transform=\"rotate(%.4f %.4f %.4f)\"", block.Rotation, x, y)
		}
		b.WriteString(fmt.Sprintf("  <text x=\"%.4f\" y=\"%.4f\" font-family=\"%s\" font-size=\"%.4f\" fill=\"%s\" fill-opacity=\"%.4f\" text-anchor=\"%s\"%s>\n", x, y, html.EscapeString(fontFamily), fontSize, html.EscapeString(color), opacity, anchor, transform))
		for i, line := range strings.Split(text, "\n") {
			if i == 0 {
				b.WriteString(fmt.Sprintf("    <tspan x=\"%.4f\" dy=\"0\">%s</tspan>\n", x, html.EscapeString(line)))
				continue
			}
			b.WriteString(fmt.Sprintf("    <tspan x=\"%.4f\" dy=\"%.4f\">%s</tspan>\n", x, fontSize*lineHeight, html.EscapeString(line)))
		}
		b.WriteString("  </text>\n")
	}

	b.WriteString("</svg>\n")
	return []byte(b.String()), nil
}

func a4Dimensions(orientation domain.Orientation) (float64, float64) {
	if orientation == domain.OrientationLandscape {
		return portraitHeightMM, portraitWidthMM
	}
	return portraitWidthMM, portraitHeightMM
}

func fitScale(pageWidth, pageHeight, sourceWidth, sourceHeight float64) float64 {
	if sourceWidth <= 0 || sourceHeight <= 0 {
		return 1
	}
	widthScale := pageWidth / sourceWidth
	heightScale := pageHeight / sourceHeight
	if widthScale < heightScale {
		return widthScale
	}
	return heightScale
}

func textAnchor(align domain.TextAlign) string {
	switch align {
	case domain.TextAlignCenter:
		return "middle"
	case domain.TextAlignEnd:
		return "end"
	default:
		return "start"
	}
}
