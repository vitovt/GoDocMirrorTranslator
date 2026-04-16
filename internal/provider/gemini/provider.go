package gemini

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
	defaultBaseURL = "https://generativelanguage.googleapis.com"
	defaultModel   = "gemini-2.5-flash"
)

type Provider struct {
	cfg        provider.ProviderConfig
	baseURL    string
	httpClient *http.Client
}

type generateContentRequest struct {
	Contents         []contentRequest        `json:"contents"`
	GenerationConfig generationConfigRequest `json:"generationConfig"`
}

type contentRequest struct {
	Parts []contentPartRequest `json:"parts"`
}

type contentPartRequest struct {
	Text       string          `json:"text,omitempty"`
	InlineData *inlineDataPart `json:"inline_data,omitempty"`
}

type inlineDataPart struct {
	MIMEType string `json:"mime_type"`
	Data     string `json:"data"`
}

type generationConfigRequest struct {
	ResponseMIMEType   string         `json:"responseMimeType"`
	ResponseJSONSchema map[string]any `json:"responseJsonSchema"`
}

type generateContentResponse struct {
	Candidates     []candidateResponse `json:"candidates"`
	PromptFeedback promptFeedback      `json:"promptFeedback"`
}

type candidateResponse struct {
	Content      contentResponse `json:"content"`
	FinishReason string          `json:"finishReason"`
}

type contentResponse struct {
	Parts []contentPartResponse `json:"parts"`
}

type contentPartResponse struct {
	Text string `json:"text"`
}

type promptFeedback struct {
	BlockReason string `json:"blockReason"`
}

func New(cfg provider.ProviderConfig) *Provider {
	return &Provider{
		cfg:        cfg,
		baseURL:    defaultBaseURL,
		httpClient: http.DefaultClient,
	}
}

func (p *Provider) Name() string {
	return "gemini"
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

	requestBody := generateContentRequest{
		Contents: []contentRequest{{
			Parts: []contentPartRequest{
				{
					Text: prompts.DocumentAnalysis(req.SourceLanguage, req.TargetLanguage, req.SourceImageWidth, req.SourceImageHeight),
				},
				{
					InlineData: &inlineDataPart{
						MIMEType: inputImage.MIMEType,
						Data:     base64.StdEncoding.EncodeToString(inputImage.Bytes),
					},
				},
			},
		}},
		GenerationConfig: generationConfigRequest{
			ResponseMIMEType:   "application/json",
			ResponseJSONSchema: provider.AnalysisJSONSchema(),
		},
	}

	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	url := strings.TrimRight(p.baseURL, "/") + "/v1beta/models/" + model + ":generateContent"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("x-goog-api-key", p.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.client().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, decodeFailure(httpResp)
	}

	var apiResp generateContentResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if strings.TrimSpace(apiResp.PromptFeedback.BlockReason) != "" {
		return nil, fmt.Errorf("provider blocked request: %s", apiResp.PromptFeedback.BlockReason)
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
		return fmt.Errorf("gemini API key is required")
	}
	return nil
}

func (p *Provider) SupportedModels() []string {
	return []string{
		defaultModel,
		"gemini-2.5-pro",
		"gemini-2.0-flash",
	}
}

func (p *Provider) client() *http.Client {
	if p.httpClient != nil {
		return p.httpClient
	}
	return http.DefaultClient
}

func decodeFailure(resp *http.Response) error {
	var apiErr struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
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

func extractOutputText(resp generateContentResponse) (string, error) {
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if strings.TrimSpace(part.Text) != "" {
				return part.Text, nil
			}
		}
		if strings.TrimSpace(candidate.FinishReason) != "" && candidate.FinishReason != "STOP" {
			return "", fmt.Errorf("provider finished with reason %s", candidate.FinishReason)
		}
	}
	return "", fmt.Errorf("provider response did not include structured output text")
}
