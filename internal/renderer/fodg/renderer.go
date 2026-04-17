package fodg

import (
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"math"
	"net/http"
	"os"
	"slices"
	"strings"

	"godocmirrortranslator/internal/domain"
	base "godocmirrortranslator/internal/renderer"
)

const mmPerPoint = 25.4 / 72.0

type Renderer struct{}

func New() *Renderer {
	return &Renderer{}
}

func (r *Renderer) Name() string {
	return "fodg"
}

func (r *Renderer) FileExtension() string {
	return ".fodg"
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

	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<office:document")
	b.WriteString(" xmlns:office=\"urn:oasis:names:tc:opendocument:xmlns:office:1.0\"")
	b.WriteString(" xmlns:draw=\"urn:oasis:names:tc:opendocument:xmlns:drawing:1.0\"")
	b.WriteString(" xmlns:text=\"urn:oasis:names:tc:opendocument:xmlns:text:1.0\"")
	b.WriteString(" xmlns:style=\"urn:oasis:names:tc:opendocument:xmlns:style:1.0\"")
	b.WriteString(" xmlns:svg=\"urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0\"")
	b.WriteString(" xmlns:fo=\"urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0\"")
	b.WriteString(" xmlns:xlink=\"http://www.w3.org/1999/xlink\"")
	b.WriteString(" xmlns:loext=\"urn:org:documentfoundation:names:experimental:office:xmlns:loext:1.0\"")
	b.WriteString(" office:version=\"1.3\"")
	b.WriteString(" office:mimetype=\"application/vnd.oasis.opendocument.graphics\">\n")

	writeFontFaceDecls(&b, page, opts)
	b.WriteString(" <office:automatic-styles>\n")
	b.WriteString(fmt.Sprintf("  <style:page-layout style:name=\"pm1\"><style:page-layout-properties fo:margin-top=\"0mm\" fo:margin-bottom=\"0mm\" fo:margin-left=\"0mm\" fo:margin-right=\"0mm\" fo:page-width=\"%s\" fo:page-height=\"%s\" style:print-orientation=\"%s\"/></style:page-layout>\n", odfLength(pageWidth), odfLength(pageHeight), page.Orientation))
	b.WriteString("  <style:style style:name=\"dp1\" style:family=\"drawing-page\"/>\n")
	b.WriteString("  <style:style style:name=\"grImage\" style:family=\"graphic\"><style:graphic-properties draw:stroke=\"none\" draw:fill=\"none\" fo:padding-top=\"0mm\" fo:padding-bottom=\"0mm\" fo:padding-left=\"0mm\" fo:padding-right=\"0mm\"/></style:style>\n")
	for i, block := range page.Blocks {
		metrics := fodgMetricsForBlock(block, opts, scale, offsetX, offsetY)
		if strings.TrimSpace(metrics.Text) == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("  <style:style style:name=\"gr%d\" style:family=\"graphic\"><style:graphic-properties draw:stroke=\"none\" draw:fill=\"none\" draw:auto-grow-height=\"false\" draw:auto-grow-width=\"false\" draw:textarea-horizontal-align=\"%s\" draw:textarea-vertical-align=\"top\" fo:padding-top=\"0mm\" fo:padding-bottom=\"0mm\" fo:padding-left=\"0mm\" fo:padding-right=\"0mm\" fo:min-height=\"0mm\" fo:min-width=\"0mm\"%s/><style:paragraph-properties style:writing-mode=\"lr-tb\"/></style:style>\n", i+1, metrics.Align, graphicReadabilityAttrs(opts, scale)))
		b.WriteString(fmt.Sprintf("  <style:style style:name=\"P%d\" style:family=\"paragraph\"><style:paragraph-properties fo:text-align=\"%s\" fo:margin-top=\"0mm\" fo:margin-bottom=\"0mm\" fo:line-height=\"%s\"/></style:style>\n", i+1, metrics.Align, odfPercent(metrics.LineHeight)))
		b.WriteString(fmt.Sprintf("  <style:style style:name=\"T%d\" style:family=\"text\"><style:text-properties fo:color=\"%s\" loext:opacity=\"%s\"%s style:font-name=\"%s\" fo:font-family=\"%s\" fo:font-size=\"%s\"%s%s/></style:style>\n",
			i+1,
			html.EscapeString(metrics.Color),
			odfPercent(metrics.Opacity),
			textOutlineAttr(opts),
			html.EscapeString(metrics.FontFamily),
			html.EscapeString(odfFontFamily(metrics.FontFamily)),
			odfPoint(metrics.FontSizeMM),
			fontWeightAttr(opts.FontWeight),
			textBackgroundAttr(opts),
		))
	}
	b.WriteString(" </office:automatic-styles>\n")
	b.WriteString(" <office:master-styles><style:master-page style:name=\"Default\" style:page-layout-name=\"pm1\"/></office:master-styles>\n")
	b.WriteString(" <office:body><office:drawing><draw:page draw:name=\"page1\" draw:style-name=\"dp1\" draw:master-page-name=\"Default\">\n")
	b.WriteString(fmt.Sprintf("  <draw:frame draw:style-name=\"grImage\" draw:layer=\"layout\" svg:x=\"%s\" svg:y=\"%s\" svg:width=\"%s\" svg:height=\"%s\" draw:z-index=\"0\">\n", odfLength(offsetX), odfLength(offsetY), odfLength(imageWidth), odfLength(imageHeight)))
	b.WriteString(fmt.Sprintf("   <draw:image draw:mime-type=\"%s\"><office:binary-data>%s</office:binary-data><text:p/></draw:image>\n", html.EscapeString(mimeType), base64.StdEncoding.EncodeToString(imageBytes)))
	b.WriteString("  </draw:frame>\n")

	for i, block := range page.Blocks {
		metrics := fodgMetricsForBlock(block, opts, scale, offsetX, offsetY)
		if strings.TrimSpace(metrics.Text) == "" {
			continue
		}
		writeTextFrame(&b, fmt.Sprintf("gr%d", i+1), fmt.Sprintf("P%d", i+1), fmt.Sprintf("T%d", i+1), metrics, metrics.FrameX, metrics.FrameY, i+1)
	}

	b.WriteString(" </draw:page></office:drawing></office:body>\n")
	b.WriteString("</office:document>\n")
	return []byte(b.String()), nil
}

func renderedText(block domain.TextBlock) string {
	if block.TranslatedText != "" {
		return block.TranslatedText
	}
	return block.SourceText
}

func fodgAlign(align domain.TextAlign) string {
	switch align {
	case domain.TextAlignCenter:
		return "center"
	case domain.TextAlignEnd:
		return "right"
	default:
		return "left"
	}
}

func frameMinHeight(text string, fontSizeMM, lineHeight float64) float64 {
	lineCount := len(strings.Split(text, "\n"))
	if lineCount < 1 {
		lineCount = 1
	}
	height := fontSizeMM
	if lineCount == 1 {
		return height
	}
	return height + float64(lineCount-1)*(fontSizeMM*lineHeight)
}

type fodgTextMetrics struct {
	Text          string
	Lines         []string
	FontFamily    string
	FontSizeMM    float64
	Color         string
	Opacity       float64
	LineHeight    float64
	Align         string
	FrameX        float64
	FrameY        float64
	FrameWidth    float64
	FrameHeight   float64
	TransformAttr string
}

func fodgMetricsForBlock(block domain.TextBlock, opts base.RenderOptions, scale, offsetX, offsetY float64) fodgTextMetrics {
	text := renderedText(block)
	fontFamily := block.FontFamily
	if fontFamily == "" {
		fontFamily = opts.FontFamily
	}
	fontSizeMM := block.FontSize * scale
	if fontSizeMM <= 0 {
		fontSizeMM = opts.DefaultFontSize * scale
	}
	if fontSizeMM < 0.9 {
		fontSizeMM = 0.9
	}
	color := block.Color
	if opts.OutlineWidth > 0 && opts.OutlineColor != "" {
		color = opts.OutlineColor
	} else if color == "" {
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

	frameX := offsetX + (block.X * scale)
	frameY := offsetY + (block.Y * scale)
	frameWidth := block.Width * scale
	if frameWidth <= 0 {
		frameWidth = estimatedFrameWidth(text, fontSizeMM)
	}
	frameHeight := block.Height * scale
	minHeight := frameMinHeight(text, fontSizeMM, lineHeight)
	if frameHeight < minHeight {
		frameHeight = minHeight
	}
	transformAttr := ""
	if block.Rotation != 0 {
		transformAttr = fmt.Sprintf(" draw:transform=\"rotate (%0.10f)\"", block.Rotation*(math.Pi/180))
	}

	return fodgTextMetrics{
		Text:          text,
		Lines:         strings.Split(text, "\n"),
		FontFamily:    fontFamily,
		FontSizeMM:    fontSizeMM,
		Color:         color,
		Opacity:       opacity,
		LineHeight:    lineHeight,
		Align:         fodgAlign(block.Align),
		FrameX:        frameX,
		FrameY:        frameY,
		FrameWidth:    frameWidth,
		FrameHeight:   frameHeight,
		TransformAttr: transformAttr,
	}
}

func estimatedFrameWidth(text string, fontSizeMM float64) float64 {
	maxLen := 1
	for _, line := range strings.Split(text, "\n") {
		if lineLen := len([]rune(line)); lineLen > maxLen {
			maxLen = lineLen
		}
	}
	return float64(maxLen) * fontSizeMM * 0.62
}

func fontWeightAttr(weight string) string {
	if weight == "bold" {
		return " fo:font-weight=\"bold\""
	}
	return ""
}

func writeFontFaceDecls(b *strings.Builder, page *domain.DocumentPage, opts base.RenderOptions) {
	fonts := []string{opts.FontFamily}
	for _, block := range page.Blocks {
		if block.FontFamily != "" {
			fonts = append(fonts, block.FontFamily)
		}
	}
	slices.Sort(fonts)
	fonts = slices.Compact(fonts)

	b.WriteString(" <office:font-face-decls>\n")
	for _, font := range fonts {
		if strings.TrimSpace(font) == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("  <style:font-face style:name=\"%s\" svg:font-family=\"%s\" style:font-family-generic=\"roman\" style:font-pitch=\"variable\"/>\n",
			html.EscapeString(font),
			html.EscapeString(odfFontFamily(font)),
		))
	}
	b.WriteString(" </office:font-face-decls>\n")
}

func graphicReadabilityAttrs(opts base.RenderOptions, scale float64) string {
	if !opts.ShadowEnabled || opts.ShadowOpacity <= 0 {
		return ""
	}

	attrs := fmt.Sprintf(" draw:shadow=\"visible\" draw:shadow-offset-x=\"%s\" draw:shadow-offset-y=\"%s\" draw:shadow-color=\"%s\" draw:shadow-opacity=\"%s\"",
		odfLength(opts.ShadowOffsetX*scale),
		odfLength(opts.ShadowOffsetY*scale),
		html.EscapeString(opts.ShadowColor),
		odfPercent(opts.ShadowOpacity),
	)
	if opts.ShadowBlur > 0 {
		attrs += fmt.Sprintf(" loext:shadow-blur=\"%s\"", odfLength(opts.ShadowBlur*scale))
	}
	return attrs
}

func textBackgroundAttr(opts base.RenderOptions) string {
	if !opts.BackgroundEnabled {
		return ""
	}
	return fmt.Sprintf(" fo:background-color=\"%s\"", html.EscapeString(opts.BackgroundColor))
}

func textOutlineAttr(opts base.RenderOptions) string {
	if opts.OutlineWidth > 0 {
		return " style:text-outline=\"true\""
	}
	return ""
}

func odfFontFamily(name string) string {
	return fmt.Sprintf("'%s'", name)
}

func writeTextFrame(b *strings.Builder, graphicStyleName, paragraphStyleName, textStyleName string, metrics fodgTextMetrics, frameX, frameY float64, zIndex int) {
	b.WriteString(fmt.Sprintf("  <draw:frame draw:style-name=\"%s\" draw:text-style-name=\"%s\" draw:layer=\"layout\" svg:x=\"%s\" svg:y=\"%s\" svg:width=\"%s\" svg:height=\"%s\" draw:z-index=\"%d\"%s>\n",
		graphicStyleName,
		paragraphStyleName,
		odfLength(frameX),
		odfLength(frameY),
		odfLength(metrics.FrameWidth),
		odfLength(metrics.FrameHeight),
		zIndex,
		metrics.TransformAttr,
	))
	b.WriteString("   <draw:text-box>\n")
	for _, line := range metrics.Lines {
		if line == "" {
			b.WriteString("    <text:p/>\n")
			continue
		}
		b.WriteString(fmt.Sprintf("    <text:p><text:span text:style-name=\"%s\">%s</text:span></text:p>\n", textStyleName, html.EscapeString(line)))
	}
	b.WriteString("   </draw:text-box>\n")
	b.WriteString("  </draw:frame>\n")
}

func odfLength(valueMM float64) string {
	return fmt.Sprintf("%.4fmm", valueMM)
}

func odfPoint(valueMM float64) string {
	return fmt.Sprintf("%.4fpt", valueMM/mmPerPoint)
}

func odfPercent(value float64) string {
	return fmt.Sprintf("%.2f%%", value*100)
}
