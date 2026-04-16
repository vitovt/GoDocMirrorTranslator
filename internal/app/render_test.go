package app

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderValidatesProviderConfigBeforeAnalyze(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	application := New("test")
	_, err := application.Render(context.Background(), RenderRequest{
		InputPath:    inputPath,
		OutputDir:    tempDir,
		ProviderName: "openai",
	})
	if err == nil {
		t.Fatal("Render() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "validate provider config") {
		t.Fatalf("Render() error = %v, want provider validation failure", err)
	}
}

func TestValidateInputImageRejectsInvalidPNG(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "bad.png")
	if err := os.WriteFile(inputPath, []byte("not-a-real-png"), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", inputPath, err)
	}

	application := New("test")
	err := application.ValidateInputImage(inputPath)
	if err == nil {
		t.Fatal("ValidateInputImage() error = nil, want decode failure")
	}
	if !strings.Contains(err.Error(), "decode input image config") {
		t.Fatalf("ValidateInputImage() error = %v, want decode failure", err)
	}
}

func TestRenderCleansUpSVGWhenLayoutJSONWriteFails(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	originalWrite := atomicWriteFile
	t.Cleanup(func() {
		atomicWriteFile = originalWrite
	})
	atomicWriteFile = func(path string, data []byte) error {
		if filepath.Ext(path) == ".json" {
			return fmt.Errorf("boom")
		}
		return writeAtomically(path, data)
	}

	application := New("test")
	_, err := application.Render(context.Background(), RenderRequest{
		InputPath:      inputPath,
		OutputDir:      filepath.Join(tempDir, "out"),
		OutputTemplate: "translated.svg",
		ProviderName:   "mock",
		SaveLayoutJSON: true,
	})
	if err == nil {
		t.Fatal("Render() error = nil, want layout json write failure")
	}
	if !strings.Contains(err.Error(), "write layout json") {
		t.Fatalf("Render() error = %v, want layout json write failure", err)
	}

	outputPath := filepath.Join(tempDir, "out", "translated.svg")
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected svg output %q to be cleaned up, stat err = %v", outputPath, statErr)
	}
	jsonPath := filepath.Join(tempDir, "out", "translated.json")
	if _, statErr := os.Stat(jsonPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected json output %q to be absent, stat err = %v", jsonPath, statErr)
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
