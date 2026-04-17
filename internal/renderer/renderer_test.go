package renderer

import "testing"

func TestRenderOptionsNormalizedDefaultsOpacity(t *testing.T) {
	opts := (RenderOptions{}).Normalized()
	if !opts.HasOpacity {
		t.Fatal("HasOpacity = false, want true")
	}
	if opts.Opacity != 1 {
		t.Fatalf("Opacity = %v, want 1", opts.Opacity)
	}
}

func TestRenderOptionsNormalizedPreservesExplicitZeroOpacity(t *testing.T) {
	opts := (RenderOptions{Opacity: 0, HasOpacity: true}).Normalized()
	if !opts.HasOpacity {
		t.Fatal("HasOpacity = false, want true")
	}
	if opts.Opacity != 0 {
		t.Fatalf("Opacity = %v, want 0", opts.Opacity)
	}
}

func TestRenderOptionsNormalizedDefaultsReadabilityFields(t *testing.T) {
	opts := (RenderOptions{}).Normalized()
	if opts.FontWeight != "normal" {
		t.Fatalf("FontWeight = %q, want normal", opts.FontWeight)
	}
	if opts.OutlineColor != "#ffffff" {
		t.Fatalf("OutlineColor = %q, want #ffffff", opts.OutlineColor)
	}
	if opts.BackgroundColor != "#ffffff" {
		t.Fatalf("BackgroundColor = %q, want #ffffff", opts.BackgroundColor)
	}
	if opts.ShadowColor != "#000000" {
		t.Fatalf("ShadowColor = %q, want #000000", opts.ShadowColor)
	}
}

func TestRenderOptionsNormalizedRejectsUnknownFontWeight(t *testing.T) {
	opts := (RenderOptions{FontWeight: "heavy"}).Normalized()
	if opts.FontWeight != "normal" {
		t.Fatalf("FontWeight = %q, want normal fallback", opts.FontWeight)
	}
}
