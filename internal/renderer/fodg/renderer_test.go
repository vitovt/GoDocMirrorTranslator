package fodg

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
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
	for _, fragment := range []string{
		"<office:document",
		"<draw:page",
		"<draw:image",
		"<office:binary-data>",
		"<draw:text-box>",
		"Hallo",
		"Welt",
		"fo:text-align=\"center\"",
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("Render() output missing %q", fragment)
		}
	}
}

func TestRenderAppliesReadabilityDecorations(t *testing.T) {
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
			TranslatedText: "Hallo",
			X:              100,
			Y:              200,
			Width:          300,
			Height:         100,
			FontSize:       24,
		}},
	}

	output, err := r.Render(context.Background(), page, base.RenderOptions{
		FontFamily:         "Noto Sans",
		DefaultFontSize:    18,
		TextColor:          "#111111",
		Opacity:            1,
		HasOpacity:         true,
		FontWeight:         "bold",
		OutlineColor:       "#ffffff",
		OutlineWidth:       2,
		BackgroundEnabled:  true,
		BackgroundColor:    "#ffffdd",
		BackgroundOpacity:  0.85,
		BackgroundPaddingX: 6,
		BackgroundPaddingY: 3,
		BackgroundRadius:   4,
		ShadowEnabled:      true,
		ShadowColor:        "#000000",
		ShadowOpacity:      0.5,
		ShadowBlur:         2,
		ShadowOffsetX:      2,
		ShadowOffsetY:      1,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(output)
	for _, fragment := range []string{
		"<draw:rect ",
		`draw:style-name="bg1"`,
		`draw:style-name="grOutline1"`,
		`draw:style-name="grShadow1"`,
		`fo:font-weight="bold"`,
		`draw:fill-color="#ffffdd"`,
		`draw:opacity="85.00%"`,
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("Render() output missing %q in %q", fragment, content)
		}
	}
}

func TestRenderMatchesGoldenFile(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 4, 4)

	r := New()
	page := &domain.DocumentPage{
		SourceImagePath:   inputPath,
		SourceImageWidth:  4,
		SourceImageHeight: 4,
		Blocks: []domain.TextBlock{{
			SourceText:     "Привіт",
			TranslatedText: "Hallo\nWelt",
			X:              1,
			Y:              1,
			Width:          2,
			Height:         1,
			FontSize:       1,
			Align:          domain.TextAlignCenter,
		}},
	}

	output, err := r.Render(context.Background(), page, base.DefaultRenderOptions())
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	goldenPath := filepath.Join("testdata", "translated_page.fodg")
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", goldenPath, err)
	}

	if string(output) != string(golden) {
		t.Fatalf("Render() output did not match golden file %q", goldenPath)
	}
}

func TestRenderRoundTripsThroughLibreOffice(t *testing.T) {
	if _, err := exec.LookPath("soffice"); err != nil {
		t.Skip("soffice is not available")
	}

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
			TranslatedText: "Hallo FODG",
			X:              100,
			Y:              200,
			Width:          300,
			Height:         100,
			FontSize:       24,
			Align:          domain.TextAlignStart,
		}},
	}

	output, err := r.Render(context.Background(), page, base.DefaultRenderOptions())
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	fodgPath := filepath.Join(tempDir, "page.fodg")
	if err := os.WriteFile(fodgPath, output, 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", fodgPath, err)
	}

	cmd := exec.Command("soffice", "--headless", "--convert-to", "svg", "--outdir", tempDir, fodgPath)
	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("soffice convert error = %v, output = %s", err, string(combined))
	}

	svgPath := filepath.Join(tempDir, "page.svg")
	svgBytes, err := os.ReadFile(svgPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", svgPath, err)
	}
	if !strings.Contains(string(svgBytes), "Hallo FODG") {
		t.Fatalf("converted SVG missing translated text: %s", string(svgBytes))
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
