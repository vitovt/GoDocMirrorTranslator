package provider

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"godocmirrortranslator/internal/domain"
)

const UnreadableText = "[unreadable]"

const maxInputImageBytes = 20 * 1024 * 1024

type AnalysisResponse struct {
	Blocks []AnalysisBlock `json:"blocks"`
}

type AnalysisBlock struct {
	ID             string           `json:"id,omitempty"`
	SourceText     string           `json:"source_text,omitempty"`
	TranslatedText string           `json:"translated_text,omitempty"`
	X              float64          `json:"x"`
	Y              float64          `json:"y"`
	Width          float64          `json:"width"`
	Height         float64          `json:"height"`
	Rotation       float64          `json:"rotation,omitempty"`
	FontSize       float64          `json:"font_size,omitempty"`
	FontFamily     string           `json:"font_family,omitempty"`
	Align          domain.TextAlign `json:"align,omitempty"`
	Color          string           `json:"color,omitempty"`
	Opacity        float64          `json:"opacity,omitempty"`
	LineHeight     float64          `json:"line_height,omitempty"`
	Confidence     *float64         `json:"confidence,omitempty"`
	Notes          string           `json:"notes,omitempty"`
}

type InputImage struct {
	Bytes    []byte
	MIMEType string
}

func AnalysisJSONSchema() map[string]any {
	blockProperties := map[string]any{
		"id":              nullableSchema("string", "Optional stable identifier for the block."),
		"source_text":     map[string]any{"type": "string", "description": "Original text from the image. Use [unreadable] when the text cannot be read."},
		"translated_text": map[string]any{"type": "string", "description": "Translated text for the block."},
		"x":               map[string]any{"type": "number", "description": "Top-left X coordinate in original image pixels."},
		"y":               map[string]any{"type": "number", "description": "Top-left Y coordinate in original image pixels."},
		"width":           map[string]any{"type": "number", "minimum": 0, "description": "Block width in original image pixels."},
		"height":          map[string]any{"type": "number", "minimum": 0, "description": "Block height in original image pixels."},
		"rotation":        nullableNumberSchema("Optional clockwise rotation in degrees."),
		"font_size":       nullableNumberSchema("Optional approximate font size in original image pixels.", "minimum", 0),
		"font_family":     nullableSchema("string", "Optional suggested font family."),
		"align":           nullableEnumSchema([]string{string(domain.TextAlignStart), string(domain.TextAlignCenter), string(domain.TextAlignEnd)}, "Optional text alignment."),
		"color":           nullableSchema("string", "Optional suggested text color."),
		"opacity":         nullableNumberSchema("Optional text opacity in the range [0,1].", "minimum", 0, "maximum", 1),
		"line_height":     nullableNumberSchema("Optional line-height multiplier.", "minimum", 0),
		"confidence":      nullableNumberSchema("Confidence estimate in the range [0,1].", "minimum", 0, "maximum", 1),
		"notes":           nullableSchema("string", "Optional notes about uncertainty or layout decisions."),
	}

	blockRequired := []string{
		"id",
		"source_text",
		"translated_text",
		"x",
		"y",
		"width",
		"height",
		"rotation",
		"font_size",
		"font_family",
		"align",
		"color",
		"opacity",
		"line_height",
		"confidence",
		"notes",
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"blocks": map[string]any{
				"type":        "array",
				"description": "Detected and translated text blocks in original image pixel space.",
				"items": map[string]any{
					"type":                 "object",
					"properties":           blockProperties,
					"required":             blockRequired,
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"blocks"},
		"additionalProperties": false,
	}
}

func nullableSchema(valueType, description string) map[string]any {
	return map[string]any{
		"type":        []string{valueType, "null"},
		"description": description,
	}
}

func nullableNumberSchema(description string, constraints ...any) map[string]any {
	schema := nullableSchema("number", description)
	for i := 0; i+1 < len(constraints); i += 2 {
		key, ok := constraints[i].(string)
		if !ok {
			continue
		}
		schema[key] = constraints[i+1]
	}
	return schema
}

func nullableEnumSchema(values []string, description string) map[string]any {
	enumValues := make([]any, 0, len(values)+1)
	for _, value := range values {
		enumValues = append(enumValues, value)
	}
	enumValues = append(enumValues, nil)
	schema := nullableSchema("string", description)
	schema["enum"] = enumValues
	return schema
}

func LoadInputImage(req AnalyzeRequest) (InputImage, error) {
	var data []byte
	switch {
	case len(req.ImageBytes) > 0:
		data = append([]byte(nil), req.ImageBytes...)
	case req.ImagePath != "":
		fileBytes, err := os.ReadFile(req.ImagePath)
		if err != nil {
			return InputImage{}, fmt.Errorf("read input image: %w", err)
		}
		data = fileBytes
	default:
		return InputImage{}, fmt.Errorf("input image is required")
	}
	if len(data) == 0 {
		return InputImage{}, fmt.Errorf("input image is empty")
	}
	if len(data) > maxInputImageBytes {
		return InputImage{}, fmt.Errorf("input image exceeds 20 MiB")
	}

	mimeType := ""
	if req.ImagePath != "" {
		mimeType = mime.TypeByExtension(strings.ToLower(filepath.Ext(req.ImagePath)))
	}
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	if mimeType == "" || mimeType == "application/octet-stream" {
		return InputImage{}, fmt.Errorf("unsupported input image mime type")
	}
	if !isSupportedInputMIMEType(mimeType) {
		return InputImage{}, fmt.Errorf("unsupported input image mime type %q", mimeType)
	}

	return InputImage{
		Bytes:    data,
		MIMEType: mimeType,
	}, nil
}

func ResolveModel(req AnalyzeRequest, cfg ProviderConfig, fallback string) string {
	if strings.TrimSpace(req.Model) != "" {
		return req.Model
	}
	if strings.TrimSpace(cfg.DefaultModel) != "" {
		return cfg.DefaultModel
	}
	return fallback
}

func isSupportedInputMIMEType(mimeType string) bool {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func (r AnalysisResponse) ToDocumentPage(providerName, model string, req AnalyzeRequest) *domain.DocumentPage {
	page := &domain.DocumentPage{
		SourceImagePath:   req.ImagePath,
		SourceImageWidth:  req.SourceImageWidth,
		SourceImageHeight: req.SourceImageHeight,
		Blocks:            make([]domain.TextBlock, 0, len(r.Blocks)),
		Metadata: map[string]string{
			"provider": providerName,
			"model":    model,
		},
	}

	for _, block := range r.Blocks {
		sourceText := strings.TrimSpace(block.SourceText)
		if sourceText == "" {
			sourceText = UnreadableText
		}
		page.Blocks = append(page.Blocks, domain.TextBlock{
			ID:             strings.TrimSpace(block.ID),
			SourceText:     sourceText,
			TranslatedText: strings.TrimSpace(block.TranslatedText),
			X:              block.X,
			Y:              block.Y,
			Width:          block.Width,
			Height:         block.Height,
			Rotation:       block.Rotation,
			FontSize:       block.FontSize,
			FontFamily:     strings.TrimSpace(block.FontFamily),
			Align:          block.Align,
			Color:          strings.TrimSpace(block.Color),
			Opacity:        block.Opacity,
			LineHeight:     block.LineHeight,
			Confidence:     block.Confidence,
			Notes:          strings.TrimSpace(block.Notes),
		})
	}

	page.Normalize()
	return page
}
