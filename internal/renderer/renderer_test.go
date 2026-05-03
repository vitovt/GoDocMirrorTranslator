package renderer

import (
	"math"
	"testing"

	"godocmirrortranslator/internal/domain"
)

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

func TestRenderOptionsNormalizedDefaultsFontSizeMode(t *testing.T) {
	opts := (RenderOptions{}).Normalized()
	if opts.FontSizeMode != FontSizeModeUnisizefont {
		t.Fatalf("FontSizeMode = %q, want %q", opts.FontSizeMode, FontSizeModeUnisizefont)
	}
}

func TestRenderOptionsNormalizedRejectsUnknownFontSizeMode(t *testing.T) {
	opts := (RenderOptions{FontSizeMode: "scaled"}).Normalized()
	if opts.FontSizeMode != FontSizeModeUnisizefont {
		t.Fatalf("FontSizeMode = %q, want %q", opts.FontSizeMode, FontSizeModeUnisizefont)
	}
}

func TestFontSizerUsesUnisizefont(t *testing.T) {
	page := &domain.DocumentPage{
		Blocks: []domain.TextBlock{
			{FontSize: 22},
			{FontSize: 28},
		},
	}

	sizer := NewFontSizer(page, RenderOptions{
		DefaultFontSize: 7,
		FontSizeMode:    FontSizeModeUnisizefont,
	})

	for _, block := range page.Blocks {
		if got := sizer.PointSize(block); got != 7 {
			t.Fatalf("PointSize(%v) = %v, want 7", block.FontSize, got)
		}
	}
}

func TestFontSizerUsesProportionalMode(t *testing.T) {
	page := &domain.DocumentPage{
		Blocks: []domain.TextBlock{
			{FontSize: 20},
			{FontSize: 24},
			{FontSize: 28},
			{FontSize: 0},
		},
	}

	sizer := NewFontSizer(page, RenderOptions{
		DefaultFontSize: 7,
		FontSizeMode:    FontSizeModeProportional,
	})

	if got := sizer.PointSize(page.Blocks[0]); math.Abs(got-(20*(7.0/24.0))) > 1e-9 {
		t.Fatalf("PointSize(20) = %v, want %v", got, 20*(7.0/24.0))
	}
	if got := sizer.PointSize(page.Blocks[1]); got != 7 {
		t.Fatalf("PointSize(24) = %v, want 7", got)
	}
	if got := sizer.PointSize(page.Blocks[2]); math.Abs(got-(28*(7.0/24.0))) > 1e-9 {
		t.Fatalf("PointSize(28) = %v, want %v", got, 28*(7.0/24.0))
	}
	if got := sizer.PointSize(page.Blocks[3]); got != 7 {
		t.Fatalf("PointSize(0) = %v, want 7 fallback", got)
	}
}
