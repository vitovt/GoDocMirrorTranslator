package integration_test

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"godocmirrortranslator/internal/app"
)

func TestRenderFlowWritesSVGAndJSON(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 900, 1200)

	application := app.New("test")
	result, err := application.Render(context.Background(), app.RenderRequest{
		InputPath:      inputPath,
		OutputDir:      filepath.Join(tempDir, "out"),
		OutputTemplate: "translated_{provider}_{timestamp}.svg",
		ProviderName:   "mock",
		SaveLayoutJSON: true,
		Now: func() time.Time {
			return time.Date(2026, 4, 16, 12, 30, 45, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	svgBytes, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatalf("ReadFile(svg): %v", err)
	}
	if !strings.Contains(string(svgBytes), "Dies ist ein Beispieltextblock.") {
		t.Fatalf("SVG output missing translated mock content: %s", string(svgBytes))
	}
	jsonBytes, err := os.ReadFile(result.LayoutJSONPath)
	if err != nil {
		t.Fatalf("ReadFile(json): %v", err)
	}
	if !strings.Contains(string(jsonBytes), "mock-body") {
		t.Fatalf("Layout JSON missing block ID: %s", string(jsonBytes))
	}
}

func writeTestPNG(t *testing.T, path string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
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
