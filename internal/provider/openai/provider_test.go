package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"godocmirrortranslator/internal/provider"
)

type capturedRequest struct {
	Model string `json:"model"`
	Input []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"input"`
	Text struct {
		Format responsesFormat `json:"format"`
	} `json:"text"`
}

func TestValidateConfigRequiresAPIKey(t *testing.T) {
	p := New(provider.ProviderConfig{})
	if err := p.ValidateConfig(provider.ProviderConfig{}); err == nil {
		t.Fatal("ValidateConfig() error = nil, want missing API key error")
	}
}

func TestValidateConfigAcceptsAPIKey(t *testing.T) {
	p := New(provider.ProviderConfig{})
	if err := p.ValidateConfig(provider.ProviderConfig{APIKey: "test-key"}); err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}
}

func TestAnalyzePageMapsStructuredOutput(t *testing.T) {
	var requestBody capturedRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("path = %s, want /v1/responses", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		_ = json.NewEncoder(w).Encode(responsesResponse{
			Status: "completed",
			Output: []responsesResponseOutput{{
				Type: "message",
				Content: []responsesResponseContent{{
					Type: "output_text",
					Text: `{"blocks":[{"id":"title","source_text":"Привіт","translated_text":"Hallo","x":12,"y":24,"width":220,"height":48,"confidence":0.93}]}`,
				}},
			}},
		})
	}))
	defer server.Close()

	p := New(provider.ProviderConfig{
		APIKey:       "test-key",
		DefaultModel: "gpt-test-model",
	})
	p.baseURL = server.URL
	p.httpClient = server.Client()

	page, err := p.AnalyzePage(context.Background(), provider.AnalyzeRequest{
		ImagePath:         "page.png",
		ImageBytes:        []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a},
		SourceImageWidth:  640,
		SourceImageHeight: 960,
		SourceLanguage:    "Ukrainian",
		TargetLanguage:    "German",
	})
	if err != nil {
		t.Fatalf("AnalyzePage() error = %v", err)
	}

	if requestBody.Model != "gpt-test-model" {
		t.Fatalf("Model = %q, want gpt-test-model", requestBody.Model)
	}
	if requestBody.Text.Format.Type != "json_schema" || !requestBody.Text.Format.Strict {
		t.Fatalf("Format = %#v, want strict json_schema", requestBody.Text.Format)
	}
	var userContent []responsesInputItem
	if err := json.Unmarshal(requestBody.Input[1].Content, &userContent); err != nil {
		t.Fatalf("Unmarshal(user content) error = %v", err)
	}
	if len(userContent) != 2 || userContent[1].Type != "input_image" || !strings.HasPrefix(userContent[1].ImageURL, "data:image/png;base64,") {
		t.Fatalf("user content = %#v, want input text + data-url image", userContent)
	}
	if len(page.Blocks) != 1 {
		t.Fatalf("len(Blocks) = %d, want 1", len(page.Blocks))
	}
	if page.Blocks[0].TranslatedText != "Hallo" {
		t.Fatalf("TranslatedText = %q, want Hallo", page.Blocks[0].TranslatedText)
	}
	if page.Metadata["provider"] != "openai" || page.Metadata["model"] != "gpt-test-model" {
		t.Fatalf("Metadata = %#v, want provider/model values", page.Metadata)
	}
}

func TestAnalyzePageReturnsAuthenticationFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "invalid API key",
			},
		})
	}))
	defer server.Close()

	p := New(provider.ProviderConfig{APIKey: "bad-key"})
	p.baseURL = server.URL
	p.httpClient = server.Client()

	_, err := p.AnalyzePage(context.Background(), provider.AnalyzeRequest{
		ImagePath:         "page.png",
		ImageBytes:        []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a},
		SourceImageWidth:  640,
		SourceImageHeight: 960,
		SourceLanguage:    "Ukrainian",
		TargetLanguage:    "German",
	})
	if err == nil {
		t.Fatal("AnalyzePage() error = nil, want authentication failure")
	}
	if !strings.Contains(err.Error(), "provider authentication failure") {
		t.Fatalf("AnalyzePage() error = %v, want authentication failure", err)
	}
}
