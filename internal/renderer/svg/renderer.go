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

	pageWidth, pageHeight := base.A4Dimensions(page.Orientation)
	scale := base.FitScale(pageWidth, pageHeight, float64(page.SourceImageWidth), float64(page.SourceImageHeight))
	imageWidth := float64(page.SourceImageWidth) * scale
	imageHeight := float64(page.SourceImageHeight) * scale
	offsetX := (pageWidth - imageWidth) / 2
	offsetY := (pageHeight - imageHeight) / 2
	mimeType := http.DetectContentType(imageBytes)
	shadowBlur := opts.ShadowBlur * scale
	shadowOffsetX := opts.ShadowOffsetX * scale
	shadowOffsetY := opts.ShadowOffsetY * scale

	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString(fmt.Sprintf("<svg xmlns=\"http://www.w3.org/2000/svg\" xmlns:xlink=\"http://www.w3.org/1999/xlink\" width=\"%.2fmm\" height=\"%.2fmm\" viewBox=\"0 0 %.2f %.2f\">\n", pageWidth, pageHeight, pageWidth, pageHeight))
	if opts.ShadowEnabled && opts.ShadowOpacity > 0 {
		b.WriteString("  <defs>\n")
		b.WriteString(fmt.Sprintf("    <filter id=\"text-shadow\" x=\"-50%%\" y=\"-50%%\" width=\"200%%\" height=\"200%%\"><feGaussianBlur in=\"SourceGraphic\" stdDeviation=\"%.4f\"/><feOffset dx=\"%.4f\" dy=\"%.4f\"/></filter>\n", shadowBlur, shadowOffsetX, shadowOffsetY))
		b.WriteString("  </defs>\n")
	}
	b.WriteString(fmt.Sprintf("  <image x=\"%.4f\" y=\"%.4f\" width=\"%.4f\" height=\"%.4f\" href=\"data:%s;base64,%s\" />\n", offsetX, offsetY, imageWidth, imageHeight, mimeType, base64.StdEncoding.EncodeToString(imageBytes)))

	for _, block := range page.Blocks {
		metrics := svgMetricsForBlock(block, opts, scale, offsetX, offsetY)
		if opts.BackgroundEnabled && opts.BackgroundOpacity > 0 {
			paddingX := opts.BackgroundPaddingX * scale
			paddingY := opts.BackgroundPaddingY * scale
			radius := opts.BackgroundRadius * scale
			b.WriteString(fmt.Sprintf("  <rect x=\"%.4f\" y=\"%.4f\" width=\"%.4f\" height=\"%.4f\" rx=\"%.4f\" ry=\"%.4f\" fill=\"%s\" fill-opacity=\"%.4f\"%s />\n",
				metrics.BoxX-paddingX,
				metrics.BoxY-paddingY,
				metrics.BoxWidth+(paddingX*2),
				metrics.BoxHeight+(paddingY*2),
				radius,
				radius,
				html.EscapeString(opts.BackgroundColor),
				opts.BackgroundOpacity,
				metrics.Transform,
			))
		}
		if opts.ShadowEnabled && opts.ShadowOpacity > 0 {
			writeTextElement(&b, metrics, fmt.Sprintf(" fill=\"%s\" fill-opacity=\"%.4f\" filter=\"url(#text-shadow)\"", html.EscapeString(opts.ShadowColor), opts.ShadowOpacity))
		}
		textAttrs := fmt.Sprintf(" fill=\"%s\" fill-opacity=\"%.4f\"", html.EscapeString(metrics.Color), metrics.Opacity)
		if opts.FontWeight == "bold" {
			textAttrs += " font-weight=\"bold\""
		}
		if opts.OutlineWidth > 0 {
			textAttrs += fmt.Sprintf(" stroke=\"%s\" stroke-width=\"%.4f\" stroke-linejoin=\"round\" paint-order=\"stroke fill\"", html.EscapeString(opts.OutlineColor), opts.OutlineWidth*scale)
		}
		writeTextElement(&b, metrics, textAttrs)
	}

	b.WriteString("</svg>\n")
	return []byte(b.String()), nil
}

type svgTextMetrics struct {
	Lines      []string
	X          float64
	Y          float64
	BoxX       float64
	BoxY       float64
	BoxWidth   float64
	BoxHeight  float64
	FontFamily string
	FontSize   float64
	Color      string
	Opacity    float64
	LineHeight float64
	Anchor     string
	Transform  string
}

func svgMetricsForBlock(block domain.TextBlock, opts base.RenderOptions, scale, offsetX, offsetY float64) svgTextMetrics {
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
	switch block.Align {
	case domain.TextAlignCenter:
		x += (block.Width * scale) / 2
	case domain.TextAlignEnd:
		x += block.Width * scale
	}
	y := offsetY + (block.Y * scale)
	anchor := textAnchor(block.Align)
	transform := ""
	if block.Rotation != 0 {
		transform = fmt.Sprintf(" transform=\"rotate(%.4f %.4f %.4f)\"", block.Rotation, x, y)
	}

	lines := strings.Split(text, "\n")
	boxWidth := block.Width * scale
	if boxWidth <= 0 {
		boxWidth = estimatedTextWidth(lines, fontSize)
	}
	boxHeight := block.Height * scale
	minHeight := estimatedTextHeight(lines, fontSize, lineHeight)
	if boxHeight < minHeight {
		boxHeight = minHeight
	}
	boxX := x
	switch block.Align {
	case domain.TextAlignCenter:
		boxX -= boxWidth / 2
	case domain.TextAlignEnd:
		boxX -= boxWidth
	}
	boxY := y - fontSize

	return svgTextMetrics{
		Lines:      lines,
		X:          x,
		Y:          y,
		BoxX:       boxX,
		BoxY:       boxY,
		BoxWidth:   boxWidth,
		BoxHeight:  boxHeight,
		FontFamily: fontFamily,
		FontSize:   fontSize,
		Color:      color,
		Opacity:    opacity,
		LineHeight: lineHeight,
		Anchor:     anchor,
		Transform:  transform,
	}
}

func writeTextElement(b *strings.Builder, metrics svgTextMetrics, attrs string) {
	b.WriteString(fmt.Sprintf("  <text x=\"%.4f\" y=\"%.4f\" font-family=\"%s\" font-size=\"%.4f\"%s text-anchor=\"%s\"%s>\n",
		metrics.X,
		metrics.Y,
		html.EscapeString(metrics.FontFamily),
		metrics.FontSize,
		attrs,
		metrics.Anchor,
		metrics.Transform,
	))
	for i, line := range metrics.Lines {
		if i == 0 {
			b.WriteString(fmt.Sprintf("    <tspan x=\"%.4f\" dy=\"0\">%s</tspan>\n", metrics.X, html.EscapeString(line)))
			continue
		}
		b.WriteString(fmt.Sprintf("    <tspan x=\"%.4f\" dy=\"%.4f\">%s</tspan>\n", metrics.X, metrics.FontSize*metrics.LineHeight, html.EscapeString(line)))
	}
	b.WriteString("  </text>\n")
}

func estimatedTextWidth(lines []string, fontSize float64) float64 {
	maxLen := 1
	for _, line := range lines {
		if lineLen := len([]rune(line)); lineLen > maxLen {
			maxLen = lineLen
		}
	}
	return float64(maxLen) * fontSize * 0.62
}

func estimatedTextHeight(lines []string, fontSize, lineHeight float64) float64 {
	lineCount := len(lines)
	if lineCount < 1 {
		lineCount = 1
	}
	if lineCount == 1 {
		return fontSize * 1.15
	}
	return (fontSize * 1.15) + float64(lineCount-1)*(fontSize*lineHeight)
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
