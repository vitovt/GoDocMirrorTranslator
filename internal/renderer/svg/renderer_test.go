package svg

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"godocmirrortranslator/internal/domain"
	base "godocmirrortranslator/internal/renderer"
)

func TestRenderEmbedsImageAndText(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 800, 1000)

	r := New()
	page := &domain.DocumentPage{
		SourceImagePath:   inputPath,
		SourceImageWidth:  800,
		SourceImageHeight: 1000,
		Blocks: []domain.TextBlock{{
			SourceText:     "Привіт",
			TranslatedText: "Hallo\nWelt",
			X:              100,
			Y:              200,
			Width:          300,
			Height:         100,
			FontSize:       24,
			Align:          domain.TextAlignCenter,
		}},
	}

	output, err := r.Render(context.Background(), page, base.DefaultRenderOptions())
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	content := string(output)
	for _, fragment := range []string{"<svg", "<image", "data:image/", "Hallo", "Welt", "text-anchor=\"middle\""} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("Render() output missing %q", fragment)
		}
	}
}

func writeTestPNG(t *testing.T, path string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 240, G: 240, B: 240, A: 255})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%q): %v", path, err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("png.Encode(): %v", err)
	}
}
