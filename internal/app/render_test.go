package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"godocmirrortranslator/internal/provider"
	base "godocmirrortranslator/internal/renderer"
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

func TestRenderWritesVersionedLayoutJSONWithRelativeSameFolderImagePath(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	application := New("test")
	result, err := application.Render(context.Background(), RenderRequest{
		InputPath:      inputPath,
		OutputDir:      tempDir,
		OutputTemplate: "translated.svg",
		ProviderName:   "mock",
		SaveLayoutJSON: true,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	data, err := os.ReadFile(result.LayoutJSONPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", result.LayoutJSONPath, err)
	}

	var layoutFile savedLayoutFile
	if err := json.Unmarshal(data, &layoutFile); err != nil {
		t.Fatalf("json.Unmarshal(layout) error = %v", err)
	}
	if layoutFile.SchemaVersion != savedLayoutSchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", layoutFile.SchemaVersion, savedLayoutSchemaVersion)
	}
	if layoutFile.Page.SourceImagePath != "page.png" {
		t.Fatalf("SourceImagePath = %q, want relative filename", layoutFile.Page.SourceImagePath)
	}
	if len(layoutFile.Page.Blocks) == 0 {
		t.Fatal("saved layout blocks are empty")
	}
}

func TestValidateLayoutJSONRejectsMissingSourceImage(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	application := New("test")
	result, err := application.Render(context.Background(), RenderRequest{
		InputPath:      inputPath,
		OutputDir:      tempDir,
		OutputTemplate: "translated.svg",
		ProviderName:   "mock",
		SaveLayoutJSON: true,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if err := os.Remove(inputPath); err != nil {
		t.Fatalf("Remove(%q) error = %v", inputPath, err)
	}

	err = application.ValidateLayoutJSON(result.LayoutJSONPath)
	if err == nil {
		t.Fatal("ValidateLayoutJSON() error = nil, want missing source image failure")
	}
	if !strings.Contains(err.Error(), "validate saved source image") {
		t.Fatalf("ValidateLayoutJSON() error = %v, want saved source image failure", err)
	}
}

func TestValidateLayoutJSONRejectsMissingLayoutFile(t *testing.T) {
	application := New("test")
	err := application.ValidateLayoutJSON("/definitely/missing/layout.json")
	if err == nil {
		t.Fatal("ValidateLayoutJSON() error = nil, want missing layout file failure")
	}
	if !strings.Contains(err.Error(), "read layout json") {
		t.Fatalf("ValidateLayoutJSON() error = %v, want missing layout file failure", err)
	}
}

func TestRerenderRejectsMismatchedSavedSourceImage(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	application := New("test")
	result, err := application.Render(context.Background(), RenderRequest{
		InputPath:      inputPath,
		OutputDir:      tempDir,
		OutputTemplate: "translated.svg",
		ProviderName:   "mock",
		SaveLayoutJSON: true,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	writeTestPNG(t, inputPath, 320, 480)

	_, err = application.Rerender(context.Background(), RerenderRequest{
		LayoutJSONPath: result.LayoutJSONPath,
		OutputDir:      tempDir,
		OutputTemplate: "rerendered.svg",
	})
	if err == nil {
		t.Fatal("Rerender() error = nil, want source image dimension mismatch")
	}
	if !strings.Contains(err.Error(), "do not match current source image") {
		t.Fatalf("Rerender() error = %v, want source image dimension mismatch", err)
	}
}

func TestRerenderUsesSavedLayoutWithoutCallingProvider(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	application := New("test")
	result, err := application.Render(context.Background(), RenderRequest{
		InputPath:      inputPath,
		OutputDir:      tempDir,
		OutputTemplate: "translated.svg",
		ProviderName:   "mock",
		SaveLayoutJSON: true,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	application.ProviderFactories["mock"] = func(provider.ProviderConfig) provider.Provider {
		t.Fatalf("mock provider should not be used during rerender")
		return nil
	}

	rerenderResult, err := application.Rerender(context.Background(), RerenderRequest{
		LayoutJSONPath: result.LayoutJSONPath,
		OutputDir:      tempDir,
		OutputTemplate: "rerendered.svg",
	})
	if err != nil {
		t.Fatalf("Rerender() error = %v", err)
	}
	if rerenderResult.OutputPath != filepath.Join(tempDir, "rerendered.svg") {
		t.Fatalf("OutputPath = %q, want %q", rerenderResult.OutputPath, filepath.Join(tempDir, "rerendered.svg"))
	}
}

func TestPlannedRenderTargetsIncludesLayoutJSON(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	application := New("test")
	targets, err := application.PlannedRenderTargets(RenderRequest{
		InputPath:      inputPath,
		OutputDir:      tempDir,
		OutputTemplate: "translated.svg",
		ProviderName:   "mock",
		SaveLayoutJSON: true,
	})
	if err != nil {
		t.Fatalf("PlannedRenderTargets() error = %v", err)
	}
	if targets.OutputPath != filepath.Join(tempDir, "translated.svg") {
		t.Fatalf("OutputPath = %q, want %q", targets.OutputPath, filepath.Join(tempDir, "translated.svg"))
	}
	if targets.LayoutJSONPath != filepath.Join(tempDir, "translated.json") {
		t.Fatalf("LayoutJSONPath = %q, want %q", targets.LayoutJSONPath, filepath.Join(tempDir, "translated.json"))
	}
}

func TestPlannedRerenderTargetsLoadsSavedLayout(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	application := New("test")
	result, err := application.Render(context.Background(), RenderRequest{
		InputPath:      inputPath,
		OutputDir:      tempDir,
		OutputTemplate: "translated.svg",
		ProviderName:   "mock",
		SaveLayoutJSON: true,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	targets, err := application.PlannedRerenderTargets(RerenderRequest{
		LayoutJSONPath: result.LayoutJSONPath,
		OutputDir:      filepath.Join(tempDir, "rerendered"),
		OutputTemplate: "layout_copy.fodg",
		RendererName:   "fodg",
	})
	if err != nil {
		t.Fatalf("PlannedRerenderTargets() error = %v", err)
	}
	if targets.OutputPath != filepath.Join(tempDir, "rerendered", "layout_copy.fodg") {
		t.Fatalf("OutputPath = %q, want %q", targets.OutputPath, filepath.Join(tempDir, "rerendered", "layout_copy.fodg"))
	}
	if targets.LayoutJSONPath != "" {
		t.Fatalf("LayoutJSONPath = %q, want empty for rerender", targets.LayoutJSONPath)
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

func TestRunRenderWritesOutputPath(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	application := New("test")
	var stdout bytes.Buffer
	err := application.RunRender(context.Background(), &stdout, RenderRequest{
		InputPath:      inputPath,
		OutputDir:      tempDir,
		OutputTemplate: "rendered.svg",
		ProviderName:   "mock",
	})
	if err != nil {
		t.Fatalf("RunRender() error = %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != filepath.Join(tempDir, "rendered.svg") {
		t.Fatalf("RunRender() output = %q, want %q", got, filepath.Join(tempDir, "rendered.svg"))
	}
}

func TestRenderUsesInputDirFallbackAndCollisionSuffix(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	application := New("test")
	result1, err := application.Render(context.Background(), RenderRequest{
		InputPath:      inputPath,
		OutputTemplate: "translated.svg",
		ProviderName:   "mock",
	})
	if err != nil {
		t.Fatalf("first Render() error = %v", err)
	}
	if result1.OutputPath != filepath.Join(tempDir, "translated.svg") {
		t.Fatalf("first OutputPath = %q, want %q", result1.OutputPath, filepath.Join(tempDir, "translated.svg"))
	}

	result2, err := application.Render(context.Background(), RenderRequest{
		InputPath:      inputPath,
		OutputTemplate: "translated.svg",
		ProviderName:   "mock",
	})
	if err != nil {
		t.Fatalf("second Render() error = %v", err)
	}
	if result2.OutputPath != filepath.Join(tempDir, "translated-1.svg") {
		t.Fatalf("second OutputPath = %q, want %q", result2.OutputPath, filepath.Join(tempDir, "translated-1.svg"))
	}
}

func TestRenderWritesFODGOutput(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	application := New("test")
	result, err := application.Render(context.Background(), RenderRequest{
		InputPath:      inputPath,
		OutputDir:      tempDir,
		OutputTemplate: "translated.svg",
		ProviderName:   "mock",
		RendererName:   "fodg",
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if filepath.Ext(result.OutputPath) != ".fodg" {
		t.Fatalf("OutputPath = %q, want .fodg extension", result.OutputPath)
	}
	content, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", result.OutputPath, err)
	}
	if !strings.Contains(string(content), "<office:document") {
		t.Fatalf("FODG output missing office document root: %s", string(content))
	}
}

func TestRenderAppliesPageLayoutOnlyToRenderedOutput(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 1200, 600)

	application := New("test")
	result, err := application.Render(context.Background(), RenderRequest{
		InputPath:      inputPath,
		OutputDir:      tempDir,
		OutputTemplate: "translated.svg",
		ProviderName:   "mock",
		SaveLayoutJSON: true,
		RenderOptions: base.RenderOptions{
			PageLayout: base.PageLayoutPortrait,
		},
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", result.OutputPath, err)
	}
	if !strings.Contains(string(content), "width=\"210.00mm\" height=\"297.00mm\"") {
		t.Fatalf("rendered svg = %q, want portrait A4 dimensions", string(content))
	}

	data, err := os.ReadFile(result.LayoutJSONPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", result.LayoutJSONPath, err)
	}
	var layoutFile savedLayoutFile
	if err := json.Unmarshal(data, &layoutFile); err != nil {
		t.Fatalf("json.Unmarshal(layout) error = %v", err)
	}
	if layoutFile.Page.Orientation != "landscape" {
		t.Fatalf("saved layout orientation = %q, want landscape from source image", layoutFile.Page.Orientation)
	}
}

func TestRerenderAppliesPageLayoutOverrideWithoutChangingSavedLayout(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 1200, 600)

	application := New("test")
	result, err := application.Render(context.Background(), RenderRequest{
		InputPath:      inputPath,
		OutputDir:      tempDir,
		OutputTemplate: "translated.svg",
		ProviderName:   "mock",
		SaveLayoutJSON: true,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	rerenderResult, err := application.Rerender(context.Background(), RerenderRequest{
		LayoutJSONPath: result.LayoutJSONPath,
		OutputDir:      tempDir,
		OutputTemplate: "portrait.svg",
		RenderOptions: base.RenderOptions{
			PageLayout: base.PageLayoutPortrait,
		},
	})
	if err != nil {
		t.Fatalf("Rerender() error = %v", err)
	}

	content, err := os.ReadFile(rerenderResult.OutputPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", rerenderResult.OutputPath, err)
	}
	if !strings.Contains(string(content), "width=\"210.00mm\" height=\"297.00mm\"") {
		t.Fatalf("rerendered svg = %q, want portrait A4 dimensions", string(content))
	}

	data, err := os.ReadFile(result.LayoutJSONPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", result.LayoutJSONPath, err)
	}
	var layoutFile savedLayoutFile
	if err := json.Unmarshal(data, &layoutFile); err != nil {
		t.Fatalf("json.Unmarshal(layout) error = %v", err)
	}
	if layoutFile.Page.Orientation != "landscape" {
		t.Fatalf("saved layout orientation = %q, want original landscape preserved", layoutFile.Page.Orientation)
	}
}

func TestRenderRejectsUnknownRenderer(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	application := New("test")
	_, err := application.Render(context.Background(), RenderRequest{
		InputPath:    inputPath,
		ProviderName: "mock",
		RendererName: "missing",
		OutputDir:    tempDir,
	})
	if err == nil {
		t.Fatal("Render() error = nil, want unknown renderer error")
	}
	if !strings.Contains(err.Error(), "unknown renderer") {
		t.Fatalf("Render() error = %v, want unknown renderer error", err)
	}
}

func TestValidateInputImageRejectsUnsupportedAndOversizedFiles(t *testing.T) {
	tempDir := t.TempDir()
	unsupportedPath := filepath.Join(tempDir, "page.gif")
	if err := os.WriteFile(unsupportedPath, []byte("gif"), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", unsupportedPath, err)
	}

	application := New("test")
	if err := application.ValidateInputImage(unsupportedPath); err == nil {
		t.Fatal("ValidateInputImage(unsupported) error = nil, want unsupported format error")
	} else if !strings.Contains(err.Error(), "unsupported image format") {
		t.Fatalf("ValidateInputImage(unsupported) error = %v, want unsupported image format", err)
	}

	overPath := filepath.Join(tempDir, "too-large.png")
	file, err := os.Create(overPath)
	if err != nil {
		t.Fatalf("Create(%q) error = %v", overPath, err)
	}
	if err := file.Truncate(20*1024*1024 + 1); err != nil {
		_ = file.Close()
		t.Fatalf("Truncate(%q) error = %v", overPath, err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close(%q) error = %v", overPath, err)
	}
	if err := application.ValidateInputImage(overPath); err == nil {
		t.Fatal("ValidateInputImage(oversized) error = nil, want size error")
	} else if !strings.Contains(err.Error(), "exceeds 20 MiB") {
		t.Fatalf("ValidateInputImage(oversized) error = %v, want size error", err)
	}
}

func TestValidateInputImageAcceptsMinimalWebP(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.webp")
	if err := os.WriteFile(inputPath, minimalVP8XWebP(320, 240), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", inputPath, err)
	}

	application := New("test")
	if err := application.ValidateInputImage(inputPath); err != nil {
		t.Fatalf("ValidateInputImage(webp) error = %v", err)
	}
}

func TestDecodeWebPDimensionsSupportsAllChunks(t *testing.T) {
	for _, tt := range []struct {
		name   string
		data   []byte
		width  int
		height int
	}{
		{name: "vp8x", data: minimalVP8XWebP(320, 240), width: 320, height: 240},
		{name: "vp8l", data: minimalVP8LWebP(320, 240), width: 320, height: 240},
		{name: "vp8", data: minimalVP8WebP(320, 240), width: 320, height: 240},
	} {
		t.Run(tt.name, func(t *testing.T) {
			width, height, err := decodeWebPDimensions(tt.data)
			if err != nil {
				t.Fatalf("decodeWebPDimensions() error = %v", err)
			}
			if width != tt.width || height != tt.height {
				t.Fatalf("decodeWebPDimensions() = %dx%d, want %dx%d", width, height, tt.width, tt.height)
			}
		})
	}

	if _, _, err := decodeWebPDimensions([]byte("not-webp")); err == nil {
		t.Fatal("decodeWebPDimensions() error = nil, want invalid header error")
	}
}

func TestOutputFileNameAndCleanupHelpers(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 30, 45, 0, time.Local)
	name := outputFileName("{input_basename}_{provider}_{model}_{date}_{time}_{timestamp}", "openai", "gpt-4.1-mini", "/tmp/page.png", now, ".svg")
	want := "page_openai_gpt-4.1-mini_2026-04-16_12-30-45_20260416-123045.svg"
	if name != want {
		t.Fatalf("outputFileName() = %q, want %q", name, want)
	}
	withExt := outputFileName("custom-name", "mock", "mock-v1", "/tmp/page.png", now, ".svg")
	if withExt != "custom-name.svg" {
		t.Fatalf("outputFileName() = %q, want custom-name.svg", withExt)
	}
	replacedExt := outputFileName(DefaultOutputTemplate, "mock", "mock-v1", "/tmp/page.png", now, ".fodg")
	if !strings.HasSuffix(replacedExt, ".fodg") || strings.HasSuffix(replacedExt, ".svg.fodg") {
		t.Fatalf("outputFileName() = %q, want renderer-specific extension replacement", replacedExt)
	}

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "result.svg")
	if err := os.WriteFile(filePath, []byte("svg"), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", filePath, err)
	}
	dirPath := filepath.Join(tempDir, "dir")
	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatalf("Mkdir(%q) error = %v", dirPath, err)
	}
	if err := os.WriteFile(filepath.Join(dirPath, "child.txt"), []byte("child"), 0o600); err != nil {
		t.Fatalf("WriteFile(child) error = %v", err)
	}
	if err := cleanupRenderOutputs([]string{filePath}, fmt.Errorf("boom")); err == nil || err.Error() != "boom" {
		t.Fatalf("cleanupRenderOutputs(file) = %v, want boom", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected %q removed, stat err = %v", filePath, err)
	}
	if err := cleanupRenderOutputs([]string{dirPath}, fmt.Errorf("boom")); err == nil || !strings.Contains(err.Error(), "cleanup failed") {
		t.Fatalf("cleanupRenderOutputs(dir) = %v, want cleanup failure", err)
	}
}

func minimalVP8XWebP(width, height int) []byte {
	data := make([]byte, 30)
	copy(data[0:4], []byte("RIFF"))
	copy(data[8:12], []byte("WEBP"))
	copy(data[12:16], []byte("VP8X"))
	w := width - 1
	h := height - 1
	data[24] = byte(w)
	data[25] = byte(w >> 8)
	data[26] = byte(w >> 16)
	data[27] = byte(h)
	data[28] = byte(h >> 8)
	data[29] = byte(h >> 16)
	return data
}

func minimalVP8LWebP(width, height int) []byte {
	data := make([]byte, 25)
	copy(data[0:4], []byte("RIFF"))
	copy(data[8:12], []byte("WEBP"))
	copy(data[12:16], []byte("VP8L"))
	bits := uint32(width-1) | (uint32(height-1) << 14)
	data[21] = byte(bits)
	data[22] = byte(bits >> 8)
	data[23] = byte(bits >> 16)
	data[24] = byte(bits >> 24)
	return data
}

func minimalVP8WebP(width, height int) []byte {
	data := make([]byte, 30)
	copy(data[0:4], []byte("RIFF"))
	copy(data[8:12], []byte("WEBP"))
	copy(data[12:16], []byte("VP8 "))
	data[26] = byte(width)
	data[27] = byte(width >> 8)
	data[28] = byte(height)
	data[29] = byte(height >> 8)
	return data
}
