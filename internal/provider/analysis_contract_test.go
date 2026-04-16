package provider

import (
	"bytes"
	"strings"
	"testing"
)

func TestLoadInputImageUsesRequestBytes(t *testing.T) {
	req := AnalyzeRequest{
		ImagePath:  "page.png",
		ImageBytes: []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a},
	}

	inputImage, err := LoadInputImage(req)
	if err != nil {
		t.Fatalf("LoadInputImage() error = %v", err)
	}
	if inputImage.MIMEType != "image/png" {
		t.Fatalf("MIMEType = %q, want image/png", inputImage.MIMEType)
	}
	if len(inputImage.Bytes) != len(req.ImageBytes) {
		t.Fatalf("len(Bytes) = %d, want %d", len(inputImage.Bytes), len(req.ImageBytes))
	}
}

func TestAnalysisResponseToDocumentPageDefaultsUnreadableSource(t *testing.T) {
	response := AnalysisResponse{
		Blocks: []AnalysisBlock{{
			TranslatedText: "Hallo",
			X:              10,
			Y:              20,
			Width:          30,
			Height:         40,
		}},
	}

	page := response.ToDocumentPage("openai", "gpt-test", AnalyzeRequest{
		ImagePath:         "page.png",
		SourceImageWidth:  100,
		SourceImageHeight: 200,
	})

	if len(page.Blocks) != 1 {
		t.Fatalf("len(Blocks) = %d, want 1", len(page.Blocks))
	}
	if page.Blocks[0].SourceText != UnreadableText {
		t.Fatalf("SourceText = %q, want %q", page.Blocks[0].SourceText, UnreadableText)
	}
	if page.Metadata["provider"] != "openai" || page.Metadata["model"] != "gpt-test" {
		t.Fatalf("Metadata = %#v, want provider/model values", page.Metadata)
	}
}

func TestLoadInputImageRejectsUnsupportedMIMEType(t *testing.T) {
	req := AnalyzeRequest{
		ImagePath:  "page.gif",
		ImageBytes: []byte("GIF89a"),
	}

	_, err := LoadInputImage(req)
	if err == nil {
		t.Fatal("LoadInputImage() error = nil, want unsupported mime type error")
	}
	if !strings.Contains(err.Error(), "unsupported input image mime type") {
		t.Fatalf("LoadInputImage() error = %v, want unsupported mime type", err)
	}
}

func TestLoadInputImageRejectsOversizedBytes(t *testing.T) {
	req := AnalyzeRequest{
		ImagePath:  "page.png",
		ImageBytes: bytes.Repeat([]byte{0x89}, maxInputImageBytes+1),
	}

	_, err := LoadInputImage(req)
	if err == nil {
		t.Fatal("LoadInputImage() error = nil, want size limit error")
	}
	if !strings.Contains(err.Error(), "exceeds 20 MiB") {
		t.Fatalf("LoadInputImage() error = %v, want size limit error", err)
	}
}
