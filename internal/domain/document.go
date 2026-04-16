package domain

import "fmt"

type PageFormat string

const (
	PageFormatA4 PageFormat = "A4"
)

type Orientation string

const (
	OrientationPortrait  Orientation = "portrait"
	OrientationLandscape Orientation = "landscape"
)

type TextAlign string

const (
	TextAlignStart  TextAlign = "start"
	TextAlignCenter TextAlign = "center"
	TextAlignEnd    TextAlign = "end"
)

const (
	DefaultLineHeight = 1.2
	DefaultOpacity    = 1.0
)

type DocumentPage struct {
	SourceImagePath   string            `json:"source_image_path"`
	SourceImageWidth  int               `json:"source_image_width"`
	SourceImageHeight int               `json:"source_image_height"`
	PageFormat        PageFormat        `json:"page_format"`
	Orientation       Orientation       `json:"orientation"`
	Blocks            []TextBlock       `json:"blocks"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

type TextBlock struct {
	ID             string    `json:"id"`
	SourceText     string    `json:"source_text"`
	TranslatedText string    `json:"translated_text"`
	X              float64   `json:"x"`
	Y              float64   `json:"y"`
	Width          float64   `json:"width"`
	Height         float64   `json:"height"`
	Rotation       float64   `json:"rotation,omitempty"`
	FontSize       float64   `json:"font_size,omitempty"`
	FontFamily     string    `json:"font_family,omitempty"`
	Align          TextAlign `json:"align,omitempty"`
	Color          string    `json:"color,omitempty"`
	Opacity        float64   `json:"opacity,omitempty"`
	LineHeight     float64   `json:"line_height,omitempty"`
	Confidence     *float64  `json:"confidence,omitempty"`
	Notes          string    `json:"notes,omitempty"`
}

func (p *DocumentPage) Normalize() {
	if p.PageFormat == "" {
		p.PageFormat = PageFormatA4
	}
	if p.Orientation == "" {
		if p.SourceImageWidth > p.SourceImageHeight {
			p.Orientation = OrientationLandscape
		} else {
			p.Orientation = OrientationPortrait
		}
	}
	if p.Metadata == nil {
		p.Metadata = map[string]string{}
	}
	for i := range p.Blocks {
		if p.Blocks[i].ID == "" {
			p.Blocks[i].ID = fmt.Sprintf("block-%d", i+1)
		}
		p.Blocks[i].Normalize()
	}
}

func (p DocumentPage) Validate() error {
	if p.SourceImagePath == "" {
		return fmt.Errorf("source image path is required")
	}
	if p.SourceImageWidth <= 0 || p.SourceImageHeight <= 0 {
		return fmt.Errorf("source image dimensions must be positive")
	}
	if p.PageFormat != "" && p.PageFormat != PageFormatA4 {
		return fmt.Errorf("unsupported page format %q", p.PageFormat)
	}
	switch p.Orientation {
	case "", OrientationPortrait, OrientationLandscape:
	default:
		return fmt.Errorf("unsupported orientation %q", p.Orientation)
	}
	for i, block := range p.Blocks {
		if err := block.Validate(); err != nil {
			return fmt.Errorf("block %d: %w", i, err)
		}
	}
	return nil
}

func (b *TextBlock) Normalize() {
	if b.LineHeight <= 0 {
		b.LineHeight = DefaultLineHeight
	}
	if b.Opacity == 0 {
		b.Opacity = DefaultOpacity
	}
	if b.Align == "" {
		b.Align = TextAlignStart
	}
}

func (b TextBlock) Validate() error {
	if b.SourceText == "" {
		return fmt.Errorf("source text is required")
	}
	if b.Width < 0 || b.Height < 0 {
		return fmt.Errorf("block dimensions must be non-negative")
	}
	if b.Opacity < 0 || b.Opacity > 1 {
		return fmt.Errorf("opacity must be between 0 and 1")
	}
	if b.LineHeight < 0 {
		return fmt.Errorf("line height must be non-negative")
	}
	switch b.Align {
	case "", TextAlignStart, TextAlignCenter, TextAlignEnd:
	default:
		return fmt.Errorf("unsupported alignment %q", b.Align)
	}
	if b.Confidence != nil && (*b.Confidence < 0 || *b.Confidence > 1) {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	return nil
}
