package gui

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

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	appcore "godocmirrortranslator/internal/app"
	"godocmirrortranslator/internal/config"
)

type noopPicker struct{}

func (noopPicker) PickInputImage(_ fyne.Window, onPicked func(path string, err error)) {
	onPicked("", nil)
}

func (noopPicker) PickOutputDir(_ fyne.Window, onPicked func(path string, err error)) {
	onPicked("", nil)
}

func TestProcessDisabledUntilRequiredFieldsAreValid(t *testing.T) {
	ui, tempDir, _ := newTestUI(t)

	if !ui.processButton.Disabled() {
		t.Fatal("process button should start disabled")
	}

	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 64, 64)

	ui.inputEntry.SetText(inputPath)
	ui.outputDirEntry.SetText(filepath.Join(tempDir, "out"))
	ui.refreshValidation()

	if ui.processButton.Disabled() {
		t.Fatalf("process button is disabled with valid mock render inputs: %s", ui.validationLabel.Text)
	}
}

func TestProviderSelectionUpdatesModelsAndAdvancedOptions(t *testing.T) {
	ui, _, _ := newTestUI(t)

	ui.providerSelect.SetSelected("openai")
	if !contains(ui.modelSelect.Options, "gpt-4.1-mini") {
		t.Fatalf("openai models = %#v, want gpt-4.1-mini present", ui.modelSelect.Options)
	}
	if ui.openAIImageDetail.Disabled() {
		t.Fatal("openai image detail should be enabled for openai provider")
	}

	ui.providerSelect.SetSelected("gemini")
	if !contains(ui.modelSelect.Options, "gemini-2.5-flash") {
		t.Fatalf("gemini models = %#v, want gemini-2.5-flash present", ui.modelSelect.Options)
	}
	if !ui.openAIImageDetail.Disabled() {
		t.Fatal("openai image detail should be disabled for non-openai providers")
	}
}

func TestSaveSettingsPersistsConfig(t *testing.T) {
	ui, _, cfgPath := newTestUI(t)

	ui.providerSelect.SetSelected("openai")
	ui.openAIKeyEntry.SetText("sk-test-key")
	ui.templateEntry.SetText("saved_{provider}.svg")
	ui.sourceLangEntry.SetText("Polish")
	ui.targetLangEntry.SetText("German")
	ui.saveLayoutJSON.SetChecked(false)
	ui.syncModelOptions()
	ui.modelSelect.SetSelected("gpt-4.1-mini")

	if _, err := ui.saveSettings(); err != nil {
		t.Fatalf("saveSettings() error = %v", err)
	}

	loaded, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load(%q) error = %v", cfgPath, err)
	}
	if loaded.DefaultProvider != "openai" {
		t.Fatalf("DefaultProvider = %q, want openai", loaded.DefaultProvider)
	}
	if loaded.OpenAIAPIKey != "sk-test-key" {
		t.Fatalf("OpenAIAPIKey = %q, want sk-test-key", loaded.OpenAIAPIKey)
	}
	if loaded.OutputTemplate != "saved_{provider}.svg" {
		t.Fatalf("OutputTemplate = %q, want saved_{provider}.svg", loaded.OutputTemplate)
	}
	if loaded.SaveLayoutJSONEnabled() {
		t.Fatal("SaveLayoutJSONEnabled() = true, want false")
	}
}

func TestStartProcessingWithMockProvider(t *testing.T) {
	ui, tempDir, cfgPath := newTestUI(t)

	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 128, 128)

	ui.inputEntry.SetText(inputPath)
	ui.outputDirEntry.SetText(filepath.Join(tempDir, "out"))
	ui.saveLayoutJSON.SetChecked(true)
	ui.refreshValidation()
	ui.startProcessing()

	waitFor(t, 3*time.Second, func() bool {
		return !ui.running
	})

	if ui.openOutputButton.Disabled() {
		t.Fatal("open output button is disabled after successful render")
	}
	if !strings.Contains(ui.statusLabel.Text, "Processing finished") {
		t.Fatalf("status = %q, want success", ui.statusLabel.Text)
	}

	outputs := strings.Split(strings.TrimSpace(ui.detailsEntry.Text), "\n")
	if len(outputs) != 2 {
		t.Fatalf("details = %q, want svg and json paths", ui.detailsEntry.Text)
	}
	for _, path := range outputs {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected output %q to exist: %v", path, err)
		}
	}

	loaded, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load(%q) error = %v", cfgPath, err)
	}
	if loaded.DefaultOutputDir != filepath.Join(tempDir, "out") {
		t.Fatalf("DefaultOutputDir = %q, want persisted output dir", loaded.DefaultOutputDir)
	}
}

func TestNewUIDisablesLayoutJSONByDefault(t *testing.T) {
	ui, _, _ := newTestUI(t)
	if ui.saveLayoutJSON.Checked {
		t.Fatal("saveLayoutJSON.Checked = true, want false from default config")
	}
}

func TestStartProcessingWithoutLayoutJSON(t *testing.T) {
	ui, tempDir, _ := newTestUI(t)

	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 128, 128)

	ui.inputEntry.SetText(inputPath)
	ui.outputDirEntry.SetText(filepath.Join(tempDir, "out"))
	ui.saveLayoutJSON.SetChecked(false)
	ui.refreshValidation()
	ui.startProcessing()

	waitFor(t, 3*time.Second, func() bool {
		return !ui.running
	})

	outputs := strings.Split(strings.TrimSpace(ui.detailsEntry.Text), "\n")
	if len(outputs) != 1 {
		t.Fatalf("details = %q, want only svg path", ui.detailsEntry.Text)
	}
	if filepath.Ext(outputs[0]) != ".svg" {
		t.Fatalf("details = %q, want svg output path", ui.detailsEntry.Text)
	}
	jsonPath := strings.TrimSuffix(outputs[0], filepath.Ext(outputs[0])) + ".json"
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Fatalf("expected no layout json at %q, stat err = %v", jsonPath, err)
	}
}

func newTestUI(t *testing.T) (*UI, string, string) {
	t.Helper()

	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")
	cfg := config.Default()
	cfg.DefaultOutputDir = filepath.Join(tempDir, "default-out")

	fyneApp := test.NewTempApp(t)
	window := fyneApp.NewWindow("test")
	ui := newUI(context.Background(), fyneApp, window, appcore.New("test"), cfgPath, cfg, noopPicker{})
	return ui, tempDir, cfgPath
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

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not satisfied before timeout")
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
