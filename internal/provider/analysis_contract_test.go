package provider

import (
	"bytes"
	"encoding/json"
	"reflect"
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

func TestAnalysisJSONSchemaRequiresEveryBlockProperty(t *testing.T) {
	schema := AnalysisJSONSchema()

	blocks, ok := schema["properties"].(map[string]any)["blocks"].(map[string]any)
	if !ok {
		t.Fatalf("blocks schema missing or wrong type: %#v", schema["properties"])
	}
	items, ok := blocks["items"].(map[string]any)
	if !ok {
		t.Fatalf("items schema missing or wrong type: %#v", blocks)
	}
	properties, ok := items["properties"].(map[string]any)
	if !ok {
		t.Fatalf("item properties missing or wrong type: %#v", items)
	}
	required, ok := items["required"].([]string)
	if !ok {
		t.Fatalf("required block fields missing or wrong type: %#v", items["required"])
	}

	wantRequired := []string{
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
	if !reflect.DeepEqual(required, wantRequired) {
		t.Fatalf("required = %#v, want %#v", required, wantRequired)
	}
	if len(properties) != len(required) {
		t.Fatalf("len(properties) = %d, want %d", len(properties), len(required))
	}
}

func TestAnalysisJSONSchemaMakesOptionalBlockFieldsNullable(t *testing.T) {
	schema := AnalysisJSONSchema()
	blocks := schema["properties"].(map[string]any)["blocks"].(map[string]any)
	items := blocks["items"].(map[string]any)
	properties := items["properties"].(map[string]any)

	cases := []struct {
		name     string
		wantType []string
		wantEnum []any
	}{
		{name: "id", wantType: []string{"string", "null"}},
		{name: "rotation", wantType: []string{"number", "null"}},
		{name: "font_size", wantType: []string{"number", "null"}},
		{name: "font_family", wantType: []string{"string", "null"}},
		{name: "align", wantType: []string{"string", "null"}, wantEnum: []any{"start", "center", "end", nil}},
		{name: "color", wantType: []string{"string", "null"}},
		{name: "opacity", wantType: []string{"number", "null"}},
		{name: "line_height", wantType: []string{"number", "null"}},
		{name: "confidence", wantType: []string{"number", "null"}},
		{name: "notes", wantType: []string{"string", "null"}},
	}

	for _, tc := range cases {
		field, ok := properties[tc.name].(map[string]any)
		if !ok {
			t.Fatalf("%s schema missing or wrong type: %#v", tc.name, properties[tc.name])
		}
		gotType, ok := field["type"].([]string)
		if !ok {
			t.Fatalf("%s type = %#v, want []string", tc.name, field["type"])
		}
		if !reflect.DeepEqual(gotType, tc.wantType) {
			t.Fatalf("%s type = %#v, want %#v", tc.name, gotType, tc.wantType)
		}
		if tc.wantEnum != nil && !reflect.DeepEqual(field["enum"], tc.wantEnum) {
			t.Fatalf("%s enum = %#v, want %#v", tc.name, field["enum"], tc.wantEnum)
		}
	}
}

func TestAnalysisResponseAcceptsNullOptionalBlockFields(t *testing.T) {
	raw := []byte(`{
		"blocks": [{
			"id": null,
			"source_text": "Привіт",
			"translated_text": "Hallo",
			"x": 12,
			"y": 24,
			"width": 220,
			"height": 48,
			"rotation": null,
			"font_size": null,
			"font_family": null,
			"align": null,
			"color": null,
			"opacity": null,
			"line_height": null,
			"confidence": null,
			"notes": null
		}]
	}`)

	var response AnalysisResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(response.Blocks) != 1 {
		t.Fatalf("len(Blocks) = %d, want 1", len(response.Blocks))
	}
	block := response.Blocks[0]
	if block.ID != "" {
		t.Fatalf("ID = %q, want empty string for null", block.ID)
	}
	if block.Align != "" {
		t.Fatalf("Align = %q, want empty alignment for null", block.Align)
	}
	if block.Confidence != nil {
		t.Fatalf("Confidence = %#v, want nil for null", block.Confidence)
	}
}
