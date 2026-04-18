package prompts

import (
	"strings"
	"testing"
)

func TestDocumentAnalysisOmitsOptionalImageDescriptionWhenEmpty(t *testing.T) {
	prompt := DocumentAnalysis("Ukrainian", "German", 640, 960, "")
	if strings.Contains(prompt, "Additional user-provided document context") {
		t.Fatalf("prompt = %q, want no optional image description section", prompt)
	}
}

func TestDocumentAnalysisIncludesOptionalImageDescriptionWhenProvided(t *testing.T) {
	prompt := DocumentAnalysis("Ukrainian", "German", 640, 960, "Employment record book page with handwritten work-history rows")
	if !strings.Contains(prompt, "Additional user-provided document context") {
		t.Fatalf("prompt = %q, want optional image description section", prompt)
	}
	if !strings.Contains(prompt, "Employment record book page with handwritten work-history rows") {
		t.Fatalf("prompt = %q, want provided image description text", prompt)
	}
	if !strings.Contains(prompt, "Do not use it to invent text that is not visible in the image.") {
		t.Fatalf("prompt = %q, want anti-hallucination constraint for image description", prompt)
	}
}
