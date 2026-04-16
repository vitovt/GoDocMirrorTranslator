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
