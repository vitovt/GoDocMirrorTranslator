package domain

import "testing"

func TestDocumentPageNormalizeAndValidate(t *testing.T) {
	page := &DocumentPage{
		SourceImagePath:   "page.png",
		SourceImageWidth:  1200,
		SourceImageHeight: 1600,
		Blocks: []TextBlock{{
			SourceText:     "Привіт",
			TranslatedText: "Hallo",
		}},
	}

	page.Normalize()
	if err := page.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if page.PageFormat != PageFormatA4 {
		t.Fatalf("PageFormat = %q, want %q", page.PageFormat, PageFormatA4)
	}
	if page.Orientation != OrientationPortrait {
		t.Fatalf("Orientation = %q, want %q", page.Orientation, OrientationPortrait)
	}
	if page.Blocks[0].ID == "" {
		t.Fatal("Normalize() did not assign block ID")
	}
	if page.Blocks[0].Opacity != DefaultOpacity {
		t.Fatalf("Opacity = %v, want %v", page.Blocks[0].Opacity, DefaultOpacity)
	}
	if page.Blocks[0].LineHeight != DefaultLineHeight {
		t.Fatalf("LineHeight = %v, want %v", page.Blocks[0].LineHeight, DefaultLineHeight)
	}
}

func TestTextBlockValidateRejectsInvalidValues(t *testing.T) {
	confidence := 1.5
	block := TextBlock{
		SourceText: "text",
		Opacity:    2,
		Confidence: &confidence,
	}
	if err := block.Validate(); err == nil {
		t.Fatal("Validate() expected error for invalid block values")
	}
}
