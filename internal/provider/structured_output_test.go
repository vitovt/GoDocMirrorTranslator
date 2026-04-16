package provider

import (
	"strings"
	"testing"
)

func TestParseStructuredOutputTextParsesRawJSON(t *testing.T) {
	analysis, err := ParseStructuredOutputText(`{"blocks":[{"source_text":"Привіт","translated_text":"Hallo","x":1,"y":2,"width":3,"height":4}]}`)
	if err != nil {
		t.Fatalf("ParseStructuredOutputText() error = %v", err)
	}
	if len(analysis.Blocks) != 1 {
		t.Fatalf("len(Blocks) = %d, want 1", len(analysis.Blocks))
	}
}

func TestParseStructuredOutputTextParsesJSONFence(t *testing.T) {
	analysis, err := ParseStructuredOutputText("```json\n{\"blocks\":[{\"source_text\":\"Текст\",\"translated_text\":\"Text\",\"x\":1,\"y\":2,\"width\":3,\"height\":4}]}\n```")
	if err != nil {
		t.Fatalf("ParseStructuredOutputText() error = %v", err)
	}
	if len(analysis.Blocks) != 1 || analysis.Blocks[0].TranslatedText != "Text" {
		t.Fatalf("analysis = %#v, want parsed fenced JSON block", analysis)
	}
}

func TestParseStructuredOutputTextRejectsEmptyOutput(t *testing.T) {
	_, err := ParseStructuredOutputText("   ")
	if err == nil {
		t.Fatal("ParseStructuredOutputText() error = nil, want empty-output error")
	}
	if !strings.Contains(err.Error(), "did not include structured output text") {
		t.Fatalf("ParseStructuredOutputText() error = %v, want missing structured output text", err)
	}
}

func TestParseStructuredOutputTextRejectsInvalidJSON(t *testing.T) {
	_, err := ParseStructuredOutputText("```json\nnot json\n```")
	if err == nil {
		t.Fatal("ParseStructuredOutputText() error = nil, want parse error")
	}
	if !strings.Contains(err.Error(), "parse provider response") {
		t.Fatalf("ParseStructuredOutputText() error = %v, want parse provider response", err)
	}
}
