package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"godocmirrortranslator/internal/domain"
	"godocmirrortranslator/internal/prompts"
	"godocmirrortranslator/internal/provider"
)

const (
	defaultBaseURL = "https://api.openai.com"
	defaultModel   = "gpt-4.1-mini"
)

type Provider struct {
	cfg        provider.ProviderConfig
	baseURL    string
	httpClient *http.Client
}

type responsesRequest struct {
	Model string                    `json:"model"`
	Input []responsesRequestMessage `json:"input"`
	Text  responsesTextConfig       `json:"text"`
}

type responsesRequestMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type responsesInputItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

type responsesTextConfig struct {
	Format responsesFormat `json:"format"`
}

type responsesFormat struct {
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict"`
}

type responsesResponse struct {
	Status string                    `json:"status"`
	Error  *responsesError           `json:"error"`
	Output []responsesResponseOutput `json:"output"`
}

type responsesResponseOutput struct {
	Type    string                     `json:"type"`
	Content []responsesResponseContent `json:"content"`
}

type responsesResponseContent struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Refusal string `json:"refusal"`
}

type responsesError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func New(cfg provider.ProviderConfig) *Provider {
	return &Provider{
		cfg:        cfg,
		baseURL:    defaultBaseURL,
		httpClient: http.DefaultClient,
	}
}

func (p *Provider) Name() string {
	return "openai"
}

func (p *Provider) AnalyzePage(ctx context.Context, req provider.AnalyzeRequest) (*domain.DocumentPage, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := p.ValidateConfig(p.cfg); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	inputImage, err := provider.LoadInputImage(req)
	if err != nil {
		return nil, err
	}
	model := provider.ResolveModel(req, p.cfg, defaultModel)

	requestBody := responsesRequest{
		Model: model,
		Input: []responsesRequestMessage{
			{
				Role:    "system",
				Content: prompts.DocumentAnalysis(req.SourceLanguage, req.TargetLanguage, req.SourceImageWidth, req.SourceImageHeight),
			},
			{
				Role: "user",
				Content: []responsesInputItem{
					{
						Type: "input_text",
						Text: "Extract and translate the document image into the structured block layout JSON schema.",
					},
					{
						Type:     "input_image",
						ImageURL: "data:" + inputImage.MIMEType + ";base64," + base64.StdEncoding.EncodeToString(inputImage.Bytes),
						Detail:   openAIImageDetail(p.cfg.AdvancedOptions),
					},
				},
			},
		},
		Text: responsesTextConfig{
			Format: responsesFormat{
				Type:   "json_schema",
				Name:   "document_page",
				Schema: provider.AnalysisJSONSchema(),
				Strict: true,
			},
		},
	}

	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.baseURL, "/")+"/v1/responses", bytes.NewReader(requestBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.client().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, decodeFailure(httpResp)
	}

	var apiResp responsesResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if apiResp.Error != nil && strings.TrimSpace(apiResp.Error.Message) != "" {
		return nil, fmt.Errorf("provider error: %s", apiResp.Error.Message)
	}
	if apiResp.Status != "" && apiResp.Status != "completed" {
		return nil, fmt.Errorf("provider response status %q", apiResp.Status)
	}

	outputText, err := extractOutputText(apiResp)
	if err != nil {
		return nil, err
	}

	var analysis provider.AnalysisResponse
	if err := json.Unmarshal([]byte(outputText), &analysis); err != nil {
		return nil, fmt.Errorf("parse provider response: %w", err)
	}

	page := analysis.ToDocumentPage(p.Name(), model, req)
	if err := page.Validate(); err != nil {
		return nil, fmt.Errorf("validate provider output: %w", err)
	}
	return page, nil
}

func (p *Provider) ValidateConfig(cfg provider.ProviderConfig) error {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return fmt.Errorf("openai API key is required")
	}
	return nil
}

func (p *Provider) SupportedModels() []string {
	return []string{
		defaultModel,
		"gpt-4.1",
		"gpt-4o",
		"gpt-4o-mini",
	}
}

func (p *Provider) client() *http.Client {
	if p.httpClient != nil {
		return p.httpClient
	}
	return http.DefaultClient
}

func openAIImageDetail(options map[string]string) string {
	switch strings.ToLower(strings.TrimSpace(options["image_detail"])) {
	case "low", "high", "auto":
		return strings.ToLower(strings.TrimSpace(options["image_detail"]))
	default:
		return "auto"
	}
}

func decodeFailure(resp *http.Response) error {
	var apiErr struct {
		Error *responsesError `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error != nil && strings.TrimSpace(apiErr.Error.Message) != "" {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return fmt.Errorf("provider authentication failure: %s", apiErr.Error.Message)
		}
		return fmt.Errorf("provider request failed with status %d: %s", resp.StatusCode, apiErr.Error.Message)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("provider authentication failure")
	}
	return fmt.Errorf("provider request failed with status %d", resp.StatusCode)
}

func extractOutputText(resp responsesResponse) (string, error) {
	for _, output := range resp.Output {
		if output.Type != "message" {
			continue
		}
		for _, content := range output.Content {
			switch content.Type {
			case "output_text":
				if strings.TrimSpace(content.Text) != "" {
					return content.Text, nil
				}
			case "refusal":
				return "", fmt.Errorf("provider refused request: %s", strings.TrimSpace(content.Refusal))
			}
		}
	}
	return "", fmt.Errorf("provider response did not include structured output text")
}
