package fodg

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math"
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
		"<office:styles/>",
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

func TestRenderUsesConfiguredUniformFontSize(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 800, 1000)

	r := New()
	page := &domain.DocumentPage{
		SourceImagePath:   inputPath,
		SourceImageWidth:  800,
		SourceImageHeight: 1000,
		Blocks: []domain.TextBlock{
			{
				SourceText:     "One",
				TranslatedText: "One",
				X:              100,
				Y:              200,
				Width:          300,
				Height:         100,
				FontSize:       22,
			},
			{
				SourceText:     "Two",
				TranslatedText: "Two",
				X:              100,
				Y:              320,
				Width:          300,
				Height:         100,
				FontSize:       28,
			},
		},
	}

	output, err := r.Render(context.Background(), page, base.RenderOptions{
		FontFamily:      "Noto Sans",
		DefaultFontSize: 7,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(output)
	if strings.Count(content, `fo:font-size="7.0000pt"`) != 2 {
		t.Fatalf("Render() output = %q, want both text styles to use configured 7pt size", content)
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
		`<style:style style:name="T1" style:family="text">`,
		`<text:span text:style-name="T1">Hallo</text:span>`,
		`fo:background-color="#ffffdd"`,
		`style:text-outline="true"`,
		`draw:shadow="visible"`,
		`draw:shadow-color="#000000"`,
		`loext:shadow-blur="0.5250mm"`,
		`fo:font-weight="bold"`,
		`fo:color="#ffffff"`,
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("Render() output missing %q in %q", fragment, content)
		}
	}
	for _, fragment := range []string{
		`<draw:rect `,
		`draw:style-name="bg1"`,
		`draw:style-name="grOutline1"`,
		`draw:style-name="grShadow1"`,
	} {
		if strings.Contains(content, fragment) {
			t.Fatalf("Render() output unexpectedly contained %q in %q", fragment, content)
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

func TestRenderFitsWideLandscapeImageToPageWidth(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "wide.png")
	writeTestPNG(t, inputPath, 1600, 800)

	r := New()
	page := &domain.DocumentPage{
		SourceImagePath:   inputPath,
		SourceImageWidth:  1600,
		SourceImageHeight: 800,
		Orientation:       domain.OrientationLandscape,
		Blocks: []domain.TextBlock{{
			SourceText:     "Привіт",
			TranslatedText: "Hallo",
			X:              1200,
			Y:              100,
			Width:          200,
			Height:         80,
			FontSize:       40,
		}},
	}

	output, err := r.Render(context.Background(), page, base.DefaultRenderOptions())
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(output)
	for _, fragment := range []string{
		`fo:page-width="297.0000mm"`,
		`fo:page-height="210.0000mm"`,
		`<draw:frame draw:style-name="grImage" draw:layer="layout" svg:x="0.0000mm" svg:y="30.7500mm" svg:width="297.0000mm" svg:height="148.5000mm" draw:z-index="0">`,
		`<draw:frame draw:style-name="gr1" draw:text-style-name="P1" draw:layer="layout" svg:x="222.7500mm"`,
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("Render() output missing %q in %q", fragment, content)
		}
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
	cmd.Env = libreOfficeTestEnv(tempDir)
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

func TestRenderLandscapeRemainsLandscapeInLibreOffice(t *testing.T) {
	if _, err := exec.LookPath("soffice"); err != nil {
		t.Skip("soffice is not available")
	}

	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "wide.png")
	writeTestPNG(t, inputPath, 1600, 800)

	r := New()
	page := &domain.DocumentPage{
		SourceImagePath:   inputPath,
		SourceImageWidth:  1600,
		SourceImageHeight: 800,
		Orientation:       domain.OrientationLandscape,
		Blocks: []domain.TextBlock{{
			SourceText:     "Привіт",
			TranslatedText: "Hallo",
			X:              1200,
			Y:              100,
			Width:          200,
			Height:         80,
			FontSize:       40,
		}},
	}

	output, err := r.Render(context.Background(), page, base.DefaultRenderOptions())
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	fodgPath := filepath.Join(tempDir, "wide.fodg")
	if err := os.WriteFile(fodgPath, output, 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", fodgPath, err)
	}

	cmd := exec.Command("soffice", "--headless", "--convert-to", "svg", "--outdir", tempDir, fodgPath)
	cmd.Env = libreOfficeTestEnv(tempDir)
	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("soffice convert error = %v, output = %s", err, string(combined))
	}

	svgPath := filepath.Join(tempDir, "wide.svg")
	svgBytes, err := os.ReadFile(svgPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", svgPath, err)
	}
	svg := string(svgBytes)
	if !strings.Contains(svg, `width="297mm" height="210mm"`) {
		t.Fatalf("converted SVG page size = %q, want landscape 297mm x 210mm", firstSVGTag(svg))
	}
	if !strings.Contains(svg, `<image `) {
		t.Fatalf("converted SVG missing image element: %s", svg)
	}
}

func TestRenderImportsReadablePropertiesInLibreOffice(t *testing.T) {
	if _, err := exec.LookPath("soffice"); err != nil {
		t.Skip("soffice is not available")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not available")
	}
	if err := exec.Command("python3", "-c", "import uno").Run(); err != nil {
		t.Skip("python3 uno bridge is not available")
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
			TranslatedText: "Hallo",
			X:              100,
			Y:              200,
			Width:          300,
			Height:         100,
			FontSize:       24,
		}},
	}

	output, err := r.Render(context.Background(), page, base.RenderOptions{
		FontFamily:        "Noto Sans",
		DefaultFontSize:   18,
		TextColor:         "#3366cc",
		Opacity:           1,
		HasOpacity:        true,
		FontWeight:        "bold",
		OutlineColor:      "#ffffff",
		OutlineWidth:      2,
		BackgroundEnabled: true,
		BackgroundColor:   "#ffffdd",
		ShadowEnabled:     true,
		ShadowColor:       "#000000",
		ShadowOpacity:     0.5,
		ShadowBlur:        2,
		ShadowOffsetX:     2,
		ShadowOffsetY:     1,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	fodgPath := filepath.Join(tempDir, "page.fodg")
	if err := os.WriteFile(fodgPath, output, 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", fodgPath, err)
	}

	listener := exec.Command("soffice", "--headless", "--nologo", "--nodefault", "--nofirststartwizard", "--accept=socket,host=127.0.0.1,port=2011;urp;StarOffice.ComponentContext")
	listener.Env = libreOfficeTestEnv(tempDir)
	if err := listener.Start(); err != nil {
		t.Fatalf("start soffice listener: %v", err)
	}
	defer func() {
		_ = listener.Process.Kill()
		_ = listener.Wait()
	}()

	script := `
import json
import sys
import time
import uno

local_ctx = uno.getComponentContext()
resolver = local_ctx.ServiceManager.createInstanceWithContext('com.sun.star.bridge.UnoUrlResolver', local_ctx)
ctx = None
for _ in range(50):
    try:
        ctx = resolver.resolve('uno:socket,host=127.0.0.1,port=2011;urp;StarOffice.ComponentContext')
        break
    except Exception:
        time.sleep(0.2)
if ctx is None:
    raise RuntimeError('could not connect to soffice listener')
desktop = ctx.ServiceManager.createInstanceWithContext('com.sun.star.frame.Desktop', ctx)
doc = desktop.loadComponentFromURL(uno.systemPathToFileUrl(sys.argv[1]), '_blank', 0, ())
page = doc.getDrawPages().getByIndex(0)
shape = page.getByIndex(1 if page.getCount() > 1 else 0)
text = shape.getText()
cursor = text.createTextCursor()
data = {
    'font_name': cursor.CharFontName,
    'font_size': cursor.CharHeight,
    'font_color': int(cursor.CharColor),
    'font_weight': float(cursor.CharWeight),
    'background_color': int(cursor.CharBackColor),
    'background_transparent': bool(cursor.CharBackTransparent),
    'contoured': bool(cursor.CharContoured),
    'shadow': bool(shape.Shadow),
    'shadow_color': int(shape.ShadowColor),
    'shadow_transparence': int(shape.ShadowTransparence),
    'shadow_x': int(shape.ShadowXDistance),
    'shadow_y': int(shape.ShadowYDistance),
    'shadow_blur': int(shape.ShadowBlur),
}
doc.close(True)
print(json.dumps(data))
`
	cmd := exec.Command("python3", "-c", script, fodgPath)
	cmd.Env = libreOfficeTestEnv(tempDir)
	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("inspect imported properties: %v, output = %s", err, string(combined))
	}

	var props struct {
		FontName              string  `json:"font_name"`
		FontSize              float64 `json:"font_size"`
		FontColor             int     `json:"font_color"`
		FontWeight            float64 `json:"font_weight"`
		BackgroundColor       int     `json:"background_color"`
		BackgroundTransparent bool    `json:"background_transparent"`
		Contoured             bool    `json:"contoured"`
		Shadow                bool    `json:"shadow"`
		ShadowColor           int     `json:"shadow_color"`
		ShadowTransparence    int     `json:"shadow_transparence"`
		ShadowX               int     `json:"shadow_x"`
		ShadowY               int     `json:"shadow_y"`
		ShadowBlur            int     `json:"shadow_blur"`
	}
	if err := json.Unmarshal(combined, &props); err != nil {
		t.Fatalf("json.Unmarshal(%q) error = %v", string(combined), err)
	}

	if props.FontName != "Noto Sans" {
		t.Fatalf("imported font name = %q, want %q", props.FontName, "Noto Sans")
	}
	if math.Abs(props.FontSize-18.0) > 0.2 {
		t.Fatalf("imported font size = %v, want about 18.0", props.FontSize)
	}
	if props.FontColor != 0xffffff {
		t.Fatalf("imported font color = %#x, want %#x", props.FontColor, 0xffffff)
	}
	if props.FontWeight != 150 {
		t.Fatalf("imported font weight = %v, want 150", props.FontWeight)
	}
	if props.BackgroundColor != 0xffffdd || props.BackgroundTransparent {
		t.Fatalf("imported text background = %#x transparent=%v, want %#x transparent=false", props.BackgroundColor, props.BackgroundTransparent, 0xffffdd)
	}
	if !props.Contoured {
		t.Fatal("imported text is not contoured")
	}
	if !props.Shadow {
		t.Fatal("imported shape shadow is not enabled")
	}
	if props.ShadowColor != 0x000000 {
		t.Fatalf("imported shadow color = %#x, want %#x", props.ShadowColor, 0x000000)
	}
	if props.ShadowTransparence != 50 {
		t.Fatalf("imported shadow transparence = %d, want 50", props.ShadowTransparence)
	}
	if props.ShadowX <= 0 || props.ShadowY <= 0 || props.ShadowBlur <= 0 {
		t.Fatalf("imported shadow geometry = x:%d y:%d blur:%d, want positive values", props.ShadowX, props.ShadowY, props.ShadowBlur)
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

func firstSVGTag(svg string) string {
	for _, line := range strings.Split(svg, "\n") {
		if strings.Contains(line, "<svg ") {
			return line
		}
	}
	return svg
}

func libreOfficeTestEnv(tempDir string) []string {
	runtimeDir := filepath.Join(tempDir, "runtime")
	cacheDir := filepath.Join(tempDir, "cache")
	configDir := filepath.Join(tempDir, "config")
	_ = os.MkdirAll(runtimeDir, 0o700)
	_ = os.MkdirAll(cacheDir, 0o700)
	_ = os.MkdirAll(configDir, 0o700)

	env := append([]string{}, os.Environ()...)
	env = append(env,
		"HOME="+tempDir,
		"XDG_RUNTIME_DIR="+runtimeDir,
		"XDG_CACHE_HOME="+cacheDir,
		"XDG_CONFIG_HOME="+configDir,
	)
	return env
}
