package gemini

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
	Contents []struct {
		Parts []contentPartRequest `json:"parts"`
	} `json:"contents"`
	GenerationConfig generationConfigRequest `json:"generationConfig"`
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
		if r.URL.Path != "/v1beta/models/gemini-test-model:generateContent" {
			t.Fatalf("path = %s, want model generateContent path", r.URL.Path)
		}
		if got := r.Header.Get("x-goog-api-key"); got != "test-key" {
			t.Fatalf("x-goog-api-key = %q, want test-key", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		_ = json.NewEncoder(w).Encode(generateContentResponse{
			Candidates: []candidateResponse{{
				FinishReason: "STOP",
				Content: contentResponse{
					Parts: []contentPartResponse{{
						Text: `{"blocks":[{"id":"body","source_text":"Текст","translated_text":"Text","x":16,"y":32,"width":180,"height":60}]}`,
					}},
				},
			}},
		})
	}))
	defer server.Close()

	p := New(provider.ProviderConfig{
		APIKey:       "test-key",
		DefaultModel: "gemini-test-model",
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

	if requestBody.GenerationConfig.ResponseMIMEType != "application/json" {
		t.Fatalf("ResponseMIMEType = %q, want application/json", requestBody.GenerationConfig.ResponseMIMEType)
	}
	if len(requestBody.Contents) != 1 || len(requestBody.Contents[0].Parts) != 2 {
		t.Fatalf("contents = %#v, want prompt text plus image parts", requestBody.Contents)
	}
	if requestBody.Contents[0].Parts[1].InlineData == nil || requestBody.Contents[0].Parts[1].InlineData.MIMEType != "image/png" {
		t.Fatalf("inline data = %#v, want image/png inline data", requestBody.Contents[0].Parts[1].InlineData)
	}
	if len(page.Blocks) != 1 || page.Blocks[0].TranslatedText != "Text" {
		t.Fatalf("page blocks = %#v, want one translated block", page.Blocks)
	}
	if page.Metadata["provider"] != "gemini" || page.Metadata["model"] != "gemini-test-model" {
		t.Fatalf("Metadata = %#v, want provider/model values", page.Metadata)
	}
}

func TestAnalyzePageReturnsAuthenticationFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "bad api key",
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
