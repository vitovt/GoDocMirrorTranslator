package cli

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunProvidersList(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"providers", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, name := range []string{"gemini", "mock", "openai"} {
		if !strings.Contains(stdout.String(), name) {
			t.Fatalf("providers list missing %q in %q", name, stdout.String())
		}
	}
}

func TestRunRender(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{
		"render",
		"--input", inputPath,
		"--output-dir", tempDir,
		"--provider", "mock",
		"--save-layout-json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() exit code = %d, stderr = %s", code, stderr.String())
	}
	outputPath := strings.TrimSpace(strings.Split(stdout.String(), "\n")[0])
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected svg output at %q: %v", outputPath, err)
	}
	jsonPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".json"
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("expected layout json at %q: %v", jsonPath, err)
	}
}

func writeTestPNG(t *testing.T, path string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 250, G: 250, B: 250, A: 255})
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
