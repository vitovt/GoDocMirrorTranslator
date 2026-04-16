package provider

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ParseStructuredOutputText(text string) (AnalysisResponse, error) {
	normalized := normalizeStructuredOutputText(text)
	if normalized == "" {
		return AnalysisResponse{}, fmt.Errorf("provider response did not include structured output text")
	}

	var analysis AnalysisResponse
	if err := json.Unmarshal([]byte(normalized), &analysis); err != nil {
		return AnalysisResponse{}, fmt.Errorf("parse provider response: %w", err)
	}
	return analysis, nil
}

func normalizeStructuredOutputText(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}

	withoutFence := strings.TrimPrefix(trimmed, "```")
	lineEnd := strings.IndexByte(withoutFence, '\n')
	if lineEnd < 0 {
		return trimmed
	}

	header := strings.TrimSpace(withoutFence[:lineEnd])
	if header != "" && !isJSONFenceHeader(header) {
		return trimmed
	}

	body := withoutFence[lineEnd+1:]
	if closingFence := strings.LastIndex(body, "```"); closingFence >= 0 {
		body = body[:closingFence]
	}
	return strings.TrimSpace(body)
}

func isJSONFenceHeader(header string) bool {
	switch strings.ToLower(strings.TrimSpace(header)) {
	case "", "json":
		return true
	default:
		return false
	}
}
