package gui

import (
	"context"
	"errors"
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
	"fyne.io/fyne/v2/theme"

	appcore "godocmirrortranslator/internal/app"
	"godocmirrortranslator/internal/config"
	"godocmirrortranslator/internal/domain"
	"godocmirrortranslator/internal/provider"
	base "godocmirrortranslator/internal/renderer"
)

type noopPicker struct{}

func (noopPicker) PickInputImage(_ fyne.Window, onPicked func(path string, err error)) {
	onPicked("", nil)
}

func (noopPicker) PickLayoutJSON(_ fyne.Window, onPicked func(path string, err error)) {
	onPicked("", nil)
}

func (noopPicker) PickOutputDir(_ fyne.Window, onPicked func(path string, err error)) {
	onPicked("", nil)
}

type fakeDevice struct {
	mobile bool
}

func (d fakeDevice) Orientation() fyne.DeviceOrientation {
	return fyne.OrientationVertical
}

func (d fakeDevice) IsMobile() bool {
	return d.mobile
}

func (d fakeDevice) IsBrowser() bool {
	return false
}

func (d fakeDevice) HasKeyboard() bool {
	return !d.mobile
}

func (d fakeDevice) SystemScaleForWindow(fyne.Window) float32 {
	return 1
}

func (d fakeDevice) Locale() fyne.Locale {
	return fyne.Locale("en-US")
}

type blockingProvider struct {
	started chan struct{}
	release chan struct{}
}

func (p *blockingProvider) Name() string {
	return "mock"
}

func (p *blockingProvider) AnalyzePage(ctx context.Context, req provider.AnalyzeRequest) (*domain.DocumentPage, error) {
	select {
	case p.started <- struct{}{}:
	default:
	}

	select {
	case <-p.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	page := &domain.DocumentPage{
		SourceImagePath:   req.ImagePath,
		SourceImageWidth:  req.SourceImageWidth,
		SourceImageHeight: req.SourceImageHeight,
		Blocks: []domain.TextBlock{{
			ID:             "mock-title",
			SourceText:     "Привіт",
			TranslatedText: "Hallo",
			X:              20,
			Y:              30,
			Width:          200,
			Height:         50,
			FontSize:       24,
			FontFamily:     "Noto Sans",
		}},
		Metadata: map[string]string{
			"provider": "mock",
			"model":    "mock-v1",
		},
	}
	page.Normalize()
	return page, nil
}

func (p *blockingProvider) ValidateConfig(provider.ProviderConfig) error {
	return nil
}

func (p *blockingProvider) SupportedModels() []string {
	return []string{"mock-v1"}
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

func TestSettingsValidationUsesApplicationProviderValidation(t *testing.T) {
	ui, _, _ := newTestUI(t)

	ui.providerSelect.SetSelected("openai")
	ui.openAIKeyEntry.SetText("")
	ui.syncModelOptions()
	ui.refreshValidation()

	if !strings.Contains(ui.validationLabel.Text, "openai API key is required") {
		t.Fatalf("validationLabel = %q, want application provider validation message", ui.validationLabel.Text)
	}
}

func TestProcessValidationUsesApplicationInputValidation(t *testing.T) {
	ui, tempDir, _ := newTestUI(t)

	inputPath := filepath.Join(tempDir, "bad.png")
	if err := os.WriteFile(inputPath, []byte("not-a-real-png"), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", inputPath, err)
	}

	ui.inputEntry.SetText(inputPath)
	ui.outputDirEntry.SetText(filepath.Join(tempDir, "out"))
	ui.refreshValidation()

	if !ui.processButton.Disabled() {
		t.Fatal("process button should be disabled for invalid png input")
	}
	if !strings.Contains(ui.validationLabel.Text, "decode input image config") {
		t.Fatalf("validationLabel = %q, want shared input validation error", ui.validationLabel.Text)
	}
}

func TestDesignValidationRejectsInvalidBackgroundOpacity(t *testing.T) {
	ui, _, _ := newTestUI(t)

	ui.backgroundOpacityEntry.SetText("not-a-number")
	ui.refreshValidation()

	if !ui.saveButton.Disabled() {
		t.Fatal("save button should be disabled for invalid background opacity")
	}
	if !strings.Contains(ui.validationLabel.Text, "background opacity must be numeric") {
		t.Fatalf("validationLabel = %q, want background opacity parse error", ui.validationLabel.Text)
	}
}

func TestDesignValidationRejectsOutOfRangeTextOpacityPercent(t *testing.T) {
	ui, _, _ := newTestUI(t)

	ui.opacityEntry.SetText("101")
	ui.refreshValidation()

	if !ui.saveButton.Disabled() {
		t.Fatal("save button should be disabled for out-of-range text opacity")
	}
	if !strings.Contains(ui.validationLabel.Text, "text opacity must be between 0 and 100") {
		t.Fatalf("validationLabel = %q, want text opacity percent range error", ui.validationLabel.Text)
	}
}

func TestSaveSettingsPersistsConfig(t *testing.T) {
	ui, _, cfgPath := newTestUI(t)

	ui.rendererSelect.SetSelected("fodg")
	ui.providerSelect.SetSelected("openai")
	ui.openAIKeyEntry.SetText("sk-test-key")
	ui.templateEntry.SetText("saved_{provider}.svg")
	ui.sourceLangEntry.SetText("Polish")
	ui.targetLangEntry.SetText("German")
	ui.opacityEntry.SetText("25")
	ui.fontWeightSelect.SetSelected("bold")
	ui.pageLayoutSelect.SetSelected("Landscape")
	ui.outlineColorEntry.SetText("#ffffff")
	ui.outlineWidthEntry.SetText("2")
	ui.backgroundEnabled.SetChecked(true)
	ui.backgroundColorEntry.SetText("#101010")
	ui.backgroundOpacityEntry.SetText("75")
	ui.backgroundPaddingXEntry.SetText("6")
	ui.backgroundPaddingYEntry.SetText("3")
	ui.backgroundRadiusEntry.SetText("8")
	ui.shadowEnabled.SetChecked(true)
	ui.shadowColorEntry.SetText("#000000")
	ui.shadowOpacityEntry.SetText("50")
	ui.shadowBlurEntry.SetText("4")
	ui.shadowOffsetXEntry.SetText("1.5")
	ui.shadowOffsetYEntry.SetText("2.5")
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
	if loaded.DefaultRenderer != "fodg" {
		t.Fatalf("DefaultRenderer = %q, want fodg", loaded.DefaultRenderer)
	}
	if loaded.OpenAIAPIKey != "sk-test-key" {
		t.Fatalf("OpenAIAPIKey = %q, want sk-test-key", loaded.OpenAIAPIKey)
	}
	if loaded.OutputTemplate != "saved_{provider}.svg" {
		t.Fatalf("OutputTemplate = %q, want saved_{provider}.svg", loaded.OutputTemplate)
	}
	if loaded.OverlayOpacity != 0.25 {
		t.Fatalf("OverlayOpacity = %v, want 0.25", loaded.OverlayOpacity)
	}
	if loaded.DefaultFontWeight != "bold" {
		t.Fatalf("DefaultFontWeight = %q, want bold", loaded.DefaultFontWeight)
	}
	if loaded.DefaultPageLayout != string(base.PageLayoutLandscape) {
		t.Fatalf("DefaultPageLayout = %q, want landscape", loaded.DefaultPageLayout)
	}
	if loaded.TextOutlineWidth != 2 {
		t.Fatalf("TextOutlineWidth = %v, want 2", loaded.TextOutlineWidth)
	}
	if !loaded.TextBackgroundEnabled {
		t.Fatal("TextBackgroundEnabled = false, want true")
	}
	if loaded.TextBackgroundColor != "#101010" {
		t.Fatalf("TextBackgroundColor = %q, want #101010", loaded.TextBackgroundColor)
	}
	if loaded.TextBackgroundOpacity != 0.75 {
		t.Fatalf("TextBackgroundOpacity = %v, want 0.75", loaded.TextBackgroundOpacity)
	}
	if loaded.TextBackgroundPaddingX != 6 || loaded.TextBackgroundPaddingY != 3 {
		t.Fatalf("background padding = (%v,%v), want (6,3)", loaded.TextBackgroundPaddingX, loaded.TextBackgroundPaddingY)
	}
	if loaded.TextBackgroundRadius != 8 {
		t.Fatalf("TextBackgroundRadius = %v, want 8", loaded.TextBackgroundRadius)
	}
	if !loaded.TextShadowEnabled {
		t.Fatal("TextShadowEnabled = false, want true")
	}
	if loaded.TextShadowOpacity != 0.5 || loaded.TextShadowBlur != 4 {
		t.Fatalf("shadow opacity/blur = (%v,%v), want (0.5,4)", loaded.TextShadowOpacity, loaded.TextShadowBlur)
	}
	if loaded.TextShadowOffsetX != 1.5 || loaded.TextShadowOffsetY != 2.5 {
		t.Fatalf("shadow offsets = (%v,%v), want (1.5,2.5)", loaded.TextShadowOffsetX, loaded.TextShadowOffsetY)
	}
}

func TestApplyConfigDisplaysOpacityAsPercent(t *testing.T) {
	ui, _, _ := newTestUI(t)

	if ui.pageLayoutSelect.Selected != "auto" {
		t.Fatalf("pageLayoutSelect.Selected = %q, want auto", ui.pageLayoutSelect.Selected)
	}
	if ui.opacityEntry.Text != "100" {
		t.Fatalf("opacityEntry.Text = %q, want 100", ui.opacityEntry.Text)
	}
	if ui.backgroundOpacityEntry.Text != "85" {
		t.Fatalf("backgroundOpacityEntry.Text = %q, want 85", ui.backgroundOpacityEntry.Text)
	}
	if ui.shadowOpacityEntry.Text != "60" {
		t.Fatalf("shadowOpacityEntry.Text = %q, want 60", ui.shadowOpacityEntry.Text)
	}
}

func TestStartProcessingWithMockProvider(t *testing.T) {
	ui, tempDir, cfgPath := newTestUI(t)

	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 128, 128)

	ui.inputEntry.SetText(inputPath)
	ui.outputDirEntry.SetText(filepath.Join(tempDir, "out"))
	ui.refreshValidation()
	ui.startProcessing()

	waitForAnalyzeTerminal(t, ui)

	if ui.openOutputButton.Disabled() {
		t.Fatal("open output button is disabled after successful render")
	}
	status := statusText(ui)
	if !strings.Contains(status, "Analyze finished") {
		t.Fatalf("status = %q, want success", status)
	}

	details := detailsText(ui)
	outputs := strings.Split(strings.TrimSpace(details), "\n")
	if len(outputs) != 2 {
		t.Fatalf("details = %q, want svg and json paths", details)
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
	if layoutJSONPathText(ui) == "" {
		t.Fatal("layoutJSONEntry.Text is empty after successful analyze")
	}
}

func TestNewUIStartsWithEmptyLayoutJSONField(t *testing.T) {
	ui, _, _ := newTestUI(t)
	if ui.layoutJSONEntry.Text != "" {
		t.Fatalf("layoutJSONEntry.Text = %q, want empty", ui.layoutJSONEntry.Text)
	}
}

func TestNewUIDefaultsToMainTabWithVisibleDesktopPanel(t *testing.T) {
	ui, _, _ := newTestUI(t)

	if ui.currentSection != sectionMain {
		t.Fatalf("currentSection = %q, want %q", ui.currentSection, sectionMain)
	}
	if !ui.menuVisible {
		t.Fatal("menu should start visible on desktop width")
	}
	if !ui.menuPanel.Visible() {
		t.Fatal("menu panel should be visible on desktop startup")
	}
	if ui.menuToggle.Text != "Hide Menu" {
		t.Fatalf("menuToggle.Text = %q, want Hide Menu", ui.menuToggle.Text)
	}
	if ui.compactLayout {
		t.Fatal("desktop UI should not start in compact layout")
	}
	if !ui.detailsVisible {
		t.Fatal("desktop UI should start with details visible")
	}
	if ui.rendererSelect.Selected != "svg" {
		t.Fatalf("rendererSelect.Selected = %q, want svg", ui.rendererSelect.Selected)
	}
	if !strings.Contains(ui.rendererCapabilityLabel.Text, "SVG / Inkscape:") {
		t.Fatalf("rendererCapabilityLabel.Text = %q, want SVG guidance", ui.rendererCapabilityLabel.Text)
	}
}

func TestRendererGuidanceUpdatesForFODG(t *testing.T) {
	ui, _, _ := newTestUI(t)

	ui.rendererSelect.SetSelected("fodg")

	if !strings.Contains(ui.rendererCapabilityLabel.Text, "FODG / LibreOffice Draw:") {
		t.Fatalf("rendererCapabilityLabel.Text = %q, want FODG guidance", ui.rendererCapabilityLabel.Text)
	}
	if !strings.Contains(ui.rendererCapabilityLabel.Text, "Text opacity is currently ignored by LibreOffice on import.") {
		t.Fatalf("rendererCapabilityLabel.Text = %q, want FODG opacity limitation", ui.rendererCapabilityLabel.Text)
	}
}

func TestNewUIUsesMobileOutputActionLabel(t *testing.T) {
	ui, _, _ := newTestUIWithDevice(t, fakeDevice{mobile: true})
	if ui.openOutputButton.Text != "Open Output File" {
		t.Fatalf("openOutputButton.Text = %q, want mobile file label", ui.openOutputButton.Text)
	}
	if !ui.menuVisible {
		t.Fatal("menu should start visible on mobile")
	}
	if !ui.menuPanel.Visible() {
		t.Fatal("menu panel should be visible on mobile startup")
	}
	if ui.menuToggle.Text != "Hide Menu" {
		t.Fatalf("menuToggle.Text = %q, want Hide Menu", ui.menuToggle.Text)
	}
	if !ui.compactLayout {
		t.Fatal("mobile UI should start in compact layout")
	}
	if ui.detailsVisible {
		t.Fatal("mobile UI should start with details collapsed")
	}
}

func TestToggleMenuShowsAndHidesSettings(t *testing.T) {
	ui, _, _ := newTestUI(t)

	ui.toggleMenu()
	if ui.menuVisible {
		t.Fatal("menu should hide after toggle")
	}
	if ui.menuPanel.Visible() {
		t.Fatal("menu panel should be hidden after toggle")
	}
	if ui.menuToggle.Text != "Show Menu" {
		t.Fatalf("menuToggle.Text = %q, want Show Menu", ui.menuToggle.Text)
	}

	ui.toggleMenu()
	if !ui.menuVisible {
		t.Fatal("menu should show after second toggle")
	}
	if !ui.menuPanel.Visible() {
		t.Fatal("menu panel should be visible after second toggle")
	}
	if ui.menuToggle.Text != "Hide Menu" {
		t.Fatalf("menuToggle.Text = %q, want Hide Menu", ui.menuToggle.Text)
	}
}

func TestMenuBackgroundTracksThemeSurface(t *testing.T) {
	ui, _, _ := newTestUI(t)

	if !sameColor(ui.menuBackground.FillColor, theme.Color(theme.ColorNameMenuBackground)) {
		t.Fatalf("initial menu background = %#v, want %#v", ui.menuBackground.FillColor, theme.Color(theme.ColorNameMenuBackground))
	}

	test.ApplyTheme(t, test.NewTheme())

	waitFor(t, time.Second, func() bool {
		return sameColor(ui.menuBackground.FillColor, theme.Color(theme.ColorNameMenuBackground))
	})
}

func TestResponsiveLayoutUsesCompactOverlayWithoutAutoHidingMenu(t *testing.T) {
	ui, _, _ := newTestUI(t)

	ui.handleResponsiveLayout(fyne.NewSize(1200, 760))
	if ui.compactLayout {
		t.Fatal("wide layout should remain non-compact")
	}
	if !ui.menuVisible {
		t.Fatal("menu should remain visible on wide layout")
	}

	ui.handleResponsiveLayout(fyne.NewSize(640, 760))
	if !ui.compactLayout {
		t.Fatal("narrow layout should switch to compact mode")
	}
	if !ui.menuVisible {
		t.Fatal("menu should stay visible when switching to compact layout")
	}
	if ui.detailsVisible {
		t.Fatal("compact layout should collapse details by default")
	}
}

func TestSelectingSectionAutoClosesMenuInCompactLayout(t *testing.T) {
	ui, _, _ := newTestUIWithDevice(t, fakeDevice{mobile: true})

	ui.selectSection(sectionAI)

	if ui.currentSection != sectionAI {
		t.Fatalf("currentSection = %q, want %q", ui.currentSection, sectionAI)
	}
	if ui.menuVisible {
		t.Fatal("menu should auto-close after selecting a section in compact layout")
	}
	if ui.menuToggle.Text != "Show Menu" {
		t.Fatalf("menuToggle.Text = %q, want Show Menu after compact selection", ui.menuToggle.Text)
	}
	if ui.menuPanel.Visible() {
		t.Fatal("menu panel should be hidden after compact selection")
	}
	if !ui.aiContentView.Visible() {
		t.Fatal("AI section should be visible after selection")
	}
	if ui.mainContentView.Visible() {
		t.Fatal("Main section should be hidden after AI selection")
	}
}

func TestCompactDetailsToggleShowsAndHidesDetails(t *testing.T) {
	ui, _, _ := newTestUIWithDevice(t, fakeDevice{mobile: true})

	if ui.detailsVisible {
		t.Fatal("details should start collapsed on compact layout")
	}
	if ui.detailsToggle.Text != "Show Details" {
		t.Fatalf("detailsToggle.Text = %q, want Show Details", ui.detailsToggle.Text)
	}

	ui.toggleDetails()
	if !ui.detailsVisible {
		t.Fatal("details should show after toggle")
	}
	if ui.detailsToggle.Text != "Hide Details" {
		t.Fatalf("detailsToggle.Text = %q, want Hide Details", ui.detailsToggle.Text)
	}
	if !ui.detailsEntry.Visible() {
		t.Fatal("details entry should be visible after expanding compact details")
	}
}

func TestSelectingSectionSwitchesVisibleContent(t *testing.T) {
	ui, _, _ := newTestUI(t)

	if !ui.mainContentView.Visible() {
		t.Fatal("main section should start visible")
	}
	if ui.aiContentView.Visible() {
		t.Fatal("AI section should start hidden")
	}

	ui.selectSection(sectionDesign)

	if !ui.designContentView.Visible() {
		t.Fatal("design section should be visible after selection")
	}
	if ui.mainContentView.Visible() {
		t.Fatal("main section should be hidden after design selection")
	}
}

func TestWideLayoutGivesActiveContentVisibleSize(t *testing.T) {
	ui, _, _ := newTestUI(t)
	renderer := test.WidgetRenderer(ui.layoutRoot)

	renderer.Layout(fyne.NewSize(960, 760))
	renderer.Refresh()
	if ui.mainContentView.Size().Width <= 0 || ui.mainContentView.Size().Height <= 0 {
		t.Fatalf("main content size = %v, want non-zero visible area", ui.mainContentView.Size())
	}

	ui.toggleMenu()
	renderer.Layout(fyne.NewSize(960, 760))
	renderer.Refresh()
	if ui.mainContentView.Size().Width <= 0 || ui.mainContentView.Size().Height <= 0 {
		t.Fatalf("main content size with hidden menu = %v, want non-zero visible area", ui.mainContentView.Size())
	}
}

func TestOpenOutputPathUsesFileOnMobile(t *testing.T) {
	ui, _, _ := newTestUIWithDevice(t, fakeDevice{mobile: true})
	ui.lastOutputDir = "/tmp/out"
	ui.lastOutputPath = "/tmp/out/result.svg"

	got, err := ui.openOutputPath()
	if err != nil {
		t.Fatalf("openOutputPath() error = %v", err)
	}
	if got != "/tmp/out/result.svg" {
		t.Fatalf("openOutputPath() = %q, want mobile output file", got)
	}
}

func TestStartAnalyzeAlwaysWritesLayoutJSON(t *testing.T) {
	ui, tempDir, _ := newTestUI(t)

	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 128, 128)

	ui.inputEntry.SetText(inputPath)
	ui.outputDirEntry.SetText(filepath.Join(tempDir, "out"))
	ui.refreshValidation()
	ui.startProcessing()

	waitForAnalyzeTerminal(t, ui)

	details := detailsText(ui)
	outputs := strings.Split(strings.TrimSpace(details), "\n")
	if len(outputs) != 2 {
		t.Fatalf("details = %q, want svg and json paths", details)
	}
	if filepath.Ext(outputs[0]) != ".svg" {
		t.Fatalf("details = %q, want svg output path", details)
	}
	if filepath.Ext(outputs[1]) != ".json" {
		t.Fatalf("details = %q, want layout json path", details)
	}
}

func TestRerenderUsesSelectedLayoutJSON(t *testing.T) {
	ui, tempDir, _ := newTestUI(t)

	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 128, 128)

	ui.inputEntry.SetText(inputPath)
	ui.outputDirEntry.SetText(filepath.Join(tempDir, "out"))
	ui.refreshValidation()
	ui.startProcessing()

	waitForAnalyzeTerminal(t, ui)

	layoutJSONPath := layoutJSONPathText(ui)
	if layoutJSONPath == "" {
		t.Fatal("layoutJSONEntry.Text is empty after analyze")
	}

	ui.rendererSelect.SetSelected("fodg")
	ui.templateEntry.SetText("rerendered_output.fodg")
	ui.refreshValidation()
	ui.startRerender()

	waitForRerenderTerminal(t, ui)

	status := statusText(ui)
	if status != "Re-render finished" {
		t.Fatalf("statusLabel.Text = %q, want Re-render finished", status)
	}
	details := detailsText(ui)
	outputs := strings.Split(strings.TrimSpace(details), "\n")
	if len(outputs) != 2 {
		t.Fatalf("details = %q, want output path and source layout json path", details)
	}
	if filepath.Ext(outputs[0]) != ".fodg" {
		t.Fatalf("rerender output = %q, want .fodg", outputs[0])
	}
	if outputs[1] != layoutJSONPath {
		t.Fatalf("rerender details layout json = %q, want %q", outputs[1], layoutJSONPath)
	}
	if got := layoutJSONPathText(ui); got != layoutJSONPath {
		t.Fatalf("layoutJSONEntry.Text = %q, want unchanged %q", got, layoutJSONPath)
	}
}

func TestRerenderValidationDoesNotRequireProviderCredentials(t *testing.T) {
	ui, tempDir, _ := newTestUI(t)

	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 128, 128)

	ui.inputEntry.SetText(inputPath)
	ui.outputDirEntry.SetText(filepath.Join(tempDir, "out"))
	ui.refreshValidation()
	ui.startProcessing()

	waitForAnalyzeTerminal(t, ui)

	ui.providerSelect.SetSelected("openai")
	ui.openAIKeyEntry.SetText("")
	ui.syncModelOptions()
	ui.refreshValidation()

	if ui.rerenderButton.Disabled() {
		t.Fatalf("rerender button is disabled with valid layout json and missing provider key: %s", ui.validationLabel.Text)
	}
	if !ui.processButton.Disabled() {
		t.Fatal("analyze button should remain disabled without openai credentials")
	}
}

func TestAnalyzeCancelLeavesExistingOutputsUnchanged(t *testing.T) {
	ui, tempDir, _ := newTestUI(t)

	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 128, 128)
	outputDir := filepath.Join(tempDir, "out")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", outputDir, err)
	}
	existingOutput := filepath.Join(outputDir, "existing.svg")
	existingJSON := filepath.Join(outputDir, "existing.json")
	if err := os.WriteFile(existingOutput, []byte("old svg"), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", existingOutput, err)
	}
	if err := os.WriteFile(existingJSON, []byte("old json"), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", existingJSON, err)
	}

	ui.confirmOverwrite = func(_ string, _ string, onDone func(bool)) {
		onDone(false)
	}
	ui.inputEntry.SetText(inputPath)
	ui.outputDirEntry.SetText(outputDir)
	ui.templateEntry.SetText("existing.svg")
	ui.refreshValidation()
	ui.startAnalyze()

	if ui.running {
		t.Fatal("analyze should not start after overwrite rejection")
	}
	if ui.statusLabel.Text != "Analyze cancelled" {
		t.Fatalf("statusLabel.Text = %q, want Analyze cancelled", ui.statusLabel.Text)
	}
	content, err := os.ReadFile(existingOutput)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", existingOutput, err)
	}
	if string(content) != "old svg" {
		t.Fatalf("existing output content = %q, want unchanged old svg", string(content))
	}
	content, err = os.ReadFile(existingJSON)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", existingJSON, err)
	}
	if string(content) != "old json" {
		t.Fatalf("existing layout json content = %q, want unchanged old json", string(content))
	}
}

func TestStartProcessingWithFODGRenderer(t *testing.T) {
	ui, tempDir, _ := newTestUI(t)

	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 128, 128)

	ui.inputEntry.SetText(inputPath)
	ui.outputDirEntry.SetText(filepath.Join(tempDir, "out"))
	ui.rendererSelect.SetSelected("fodg")
	ui.refreshValidation()
	ui.startProcessing()

	waitForAnalyzeTerminal(t, ui)

	details := detailsText(ui)
	outputPath := strings.Split(strings.TrimSpace(details), "\n")[0]
	if filepath.Ext(outputPath) != ".fodg" {
		t.Fatalf("details = %q, want fodg output path", details)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected output %q to exist: %v", outputPath, err)
	}
}

func TestStartProcessingUsesInputFolderWhenOutputDirEmpty(t *testing.T) {
	ui, tempDir, _ := newTestUI(t)

	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 128, 128)

	ui.inputEntry.SetText(inputPath)
	ui.outputDirEntry.SetText("")
	ui.refreshValidation()

	if ui.processButton.Disabled() {
		t.Fatalf("process button is disabled with empty output dir fallback: %s", ui.validationLabel.Text)
	}

	ui.startProcessing()

	waitForAnalyzeTerminal(t, ui)

	outputPath := strings.Split(strings.TrimSpace(detailsText(ui)), "\n")[0]
	if outputPath == "" {
		t.Fatal("detailsEntry.Text is empty, want rendered output path")
	}
	if filepath.Dir(outputPath) != tempDir {
		t.Fatalf("output dir = %q, want input dir %q", filepath.Dir(outputPath), tempDir)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected output %q to exist: %v", outputPath, err)
	}
}

func TestProcessingDisablesInteractiveControls(t *testing.T) {
	application := appcore.New("test")
	providerImpl := &blockingProvider{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	application.ProviderFactories["mock"] = func(provider.ProviderConfig) provider.Provider {
		return providerImpl
	}

	ui, tempDir, _ := newTestUIWithApplication(t, nil, application)
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 128, 128)

	ui.inputEntry.SetText(inputPath)
	ui.outputDirEntry.SetText(filepath.Join(tempDir, "out"))
	ui.refreshValidation()
	ui.startProcessing()

	waitFor(t, time.Second, func() bool {
		select {
		case <-providerImpl.started:
			return true
		default:
			return false
		}
	})

	for _, check := range []struct {
		name     string
		disabled bool
	}{
		{"input entry", ui.inputEntry.Disabled()},
		{"input browse", ui.inputBrowseButton.Disabled()},
		{"output entry", ui.outputDirEntry.Disabled()},
		{"output browse", ui.outputBrowseButton.Disabled()},
		{"renderer select", ui.rendererSelect.Disabled()},
		{"provider select", ui.providerSelect.Disabled()},
		{"save settings", ui.saveButton.Disabled()},
		{"analyze", ui.processButton.Disabled()},
		{"rerender", ui.rerenderButton.Disabled()},
	} {
		if !check.disabled {
			t.Fatalf("%s should be disabled while processing", check.name)
		}
	}

	close(providerImpl.release)
	waitForAnalyzeTerminal(t, ui)

	if ui.inputEntry.Disabled() {
		t.Fatal("input entry should be re-enabled after processing")
	}
	if ui.inputBrowseButton.Disabled() {
		t.Fatal("input browse should be re-enabled after processing")
	}
	if ui.outputDirEntry.Disabled() {
		t.Fatal("output entry should be re-enabled after processing")
	}
	if ui.outputBrowseButton.Disabled() {
		t.Fatal("output browse should be re-enabled after processing")
	}
	if ui.rendererSelect.Disabled() {
		t.Fatal("renderer select should be re-enabled after processing")
	}
	if ui.providerSelect.Disabled() {
		t.Fatal("provider select should be re-enabled after processing")
	}
	if ui.processButton.Disabled() {
		t.Fatal("process button should be enabled again after successful processing")
	}
	if ui.rerenderButton.Disabled() {
		t.Fatal("rerender button should be enabled again after successful processing")
	}
}

type scriptedPicker struct {
	inputPath  string
	inputErr   error
	layoutPath string
	layoutErr  error
	outputPath string
	outputErr  error
}

func (p scriptedPicker) PickInputImage(_ fyne.Window, onPicked func(path string, err error)) {
	onPicked(p.inputPath, p.inputErr)
}

func (p scriptedPicker) PickLayoutJSON(_ fyne.Window, onPicked func(path string, err error)) {
	onPicked(p.layoutPath, p.layoutErr)
}

func (p scriptedPicker) PickOutputDir(_ fyne.Window, onPicked func(path string, err error)) {
	onPicked(p.outputPath, p.outputErr)
}

type failingProvider struct {
	err error
}

func (p *failingProvider) Name() string {
	return "mock"
}

func (p *failingProvider) AnalyzePage(context.Context, provider.AnalyzeRequest) (*domain.DocumentPage, error) {
	return nil, p.err
}

func (p *failingProvider) ValidateConfig(provider.ProviderConfig) error {
	return nil
}

func (p *failingProvider) SupportedModels() []string {
	return []string{"mock-v1"}
}

func TestPickInputImageSetsInputAndDefaultOutputDir(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 64, 64)

	ui, _, _ := newTestUIWithPicker(t, nil, appcore.New("test"), scriptedPicker{inputPath: inputPath})
	ui.outputDirEntry.SetText("")

	ui.pickInputImage()

	if ui.inputEntry.Text != inputPath {
		t.Fatalf("inputEntry.Text = %q, want %q", ui.inputEntry.Text, inputPath)
	}
	if ui.outputDirEntry.Text != tempDir {
		t.Fatalf("outputDirEntry.Text = %q, want %q", ui.outputDirEntry.Text, tempDir)
	}
}

func TestPickInputImageHandlesPickerError(t *testing.T) {
	ui, _, _ := newTestUIWithPicker(t, nil, appcore.New("test"), scriptedPicker{inputErr: errors.New("input picker failed")})

	ui.pickInputImage()

	if ui.statusLabel.Text != "Failed to choose input image" {
		t.Fatalf("statusLabel.Text = %q, want picker failure status", ui.statusLabel.Text)
	}
	if !strings.Contains(ui.detailsEntry.Text, "input picker failed") {
		t.Fatalf("detailsEntry.Text = %q, want picker failure details", ui.detailsEntry.Text)
	}
}

func TestPickOutputDirSetsSelectedPath(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "out")
	ui, _, _ := newTestUIWithPicker(t, nil, appcore.New("test"), scriptedPicker{outputPath: outputDir})

	ui.pickOutputDir()

	if ui.outputDirEntry.Text != outputDir {
		t.Fatalf("outputDirEntry.Text = %q, want %q", ui.outputDirEntry.Text, outputDir)
	}
}

func TestPickOutputDirHandlesPickerError(t *testing.T) {
	ui, _, _ := newTestUIWithPicker(t, nil, appcore.New("test"), scriptedPicker{outputErr: errors.New("output picker failed")})

	ui.pickOutputDir()

	if ui.statusLabel.Text != "Failed to choose output folder" {
		t.Fatalf("statusLabel.Text = %q, want picker failure status", ui.statusLabel.Text)
	}
	if !strings.Contains(ui.detailsEntry.Text, "output picker failed") {
		t.Fatalf("detailsEntry.Text = %q, want picker failure details", ui.detailsEntry.Text)
	}
}

func TestPickLayoutJSONSetsSelectedPath(t *testing.T) {
	layoutJSONPath := filepath.Join(t.TempDir(), "page.json")
	ui, _, _ := newTestUIWithPicker(t, nil, appcore.New("test"), scriptedPicker{layoutPath: layoutJSONPath})

	ui.pickLayoutJSON()

	if ui.layoutJSONEntry.Text != layoutJSONPath {
		t.Fatalf("layoutJSONEntry.Text = %q, want %q", ui.layoutJSONEntry.Text, layoutJSONPath)
	}
}

func TestPickLayoutJSONHandlesPickerError(t *testing.T) {
	ui, _, _ := newTestUIWithPicker(t, nil, appcore.New("test"), scriptedPicker{layoutErr: errors.New("layout picker failed")})

	ui.pickLayoutJSON()

	if ui.statusLabel.Text != "Failed to choose layout json" {
		t.Fatalf("statusLabel.Text = %q, want picker failure status", ui.statusLabel.Text)
	}
	if !strings.Contains(ui.detailsEntry.Text, "layout picker failed") {
		t.Fatalf("detailsEntry.Text = %q, want picker failure details", ui.detailsEntry.Text)
	}
}

func TestStartProcessingFailureShowsStatus(t *testing.T) {
	application := appcore.New("test")
	application.ProviderFactories["mock"] = func(provider.ProviderConfig) provider.Provider {
		return &failingProvider{err: errors.New("provider exploded")}
	}

	ui, tempDir, _ := newTestUIWithPicker(t, nil, application, noopPicker{})
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 128, 128)

	ui.inputEntry.SetText(inputPath)
	ui.outputDirEntry.SetText(filepath.Join(tempDir, "out"))
	ui.refreshValidation()
	ui.startProcessing()

	waitForAnalyzeTerminal(t, ui)

	if ui.openOutputButton.Disabled() != true {
		t.Fatal("open output button should remain disabled after failed processing")
	}
	status := statusText(ui)
	if status != "Analyze failed" {
		t.Fatalf("statusLabel.Text = %q, want Analyze failed", status)
	}
	details := detailsText(ui)
	if !strings.Contains(details, "provider exploded") {
		t.Fatalf("detailsEntry.Text = %q, want provider error details", details)
	}
}

func TestOpenOutputPathDesktopRequiresOutputDir(t *testing.T) {
	ui, _, _ := newTestUI(t)
	if _, err := ui.openOutputPath(); err == nil {
		t.Fatal("openOutputPath() error = nil, want missing desktop output dir error")
	}

	ui.lastOutputDir = "/tmp/out"
	got, err := ui.openOutputPath()
	if err != nil {
		t.Fatalf("openOutputPath() error = %v", err)
	}
	if got != "/tmp/out" {
		t.Fatalf("openOutputPath() = %q, want /tmp/out", got)
	}
}

func newTestUIWithPicker(t *testing.T, device fyne.Device, application *appcore.Application, picker picker) (*UI, string, string) {
	t.Helper()

	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")
	cfg := config.Default()
	cfg.DefaultOutputDir = filepath.Join(tempDir, "default-out")
	t.Setenv("LANG", "en_US.UTF-8")
	t.Setenv("LC_ALL", "en_US.UTF-8")

	fyneApp := test.NewTempApp(t)
	window := fyneApp.NewWindow("test")
	if device == nil {
		device = fyneApp.Driver().Device()
	}
	if !isMobileDevice(device) {
		window.Resize(fyne.NewSize(960, 760))
	}
	ui := newUI(context.Background(), fyneApp, device, window, application, cfgPath, cfg, picker)
	return ui, tempDir, cfgPath
}

func newTestUI(t *testing.T) (*UI, string, string) {
	t.Helper()
	return newTestUIWithApplication(t, nil, appcore.New("test"))
}

func newTestUIWithDevice(t *testing.T, device fyne.Device) (*UI, string, string) {
	t.Helper()
	return newTestUIWithApplication(t, device, appcore.New("test"))
}

func newTestUIWithApplication(t *testing.T, device fyne.Device, application *appcore.Application) (*UI, string, string) {
	t.Helper()

	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")
	cfg := config.Default()
	cfg.DefaultOutputDir = filepath.Join(tempDir, "default-out")
	t.Setenv("LANG", "en_US.UTF-8")
	t.Setenv("LC_ALL", "en_US.UTF-8")

	fyneApp := test.NewTempApp(t)
	window := fyneApp.NewWindow("test")
	if device == nil {
		device = fyneApp.Driver().Device()
	}
	if !isMobileDevice(device) {
		window.Resize(fyne.NewSize(960, 760))
	}
	ui := newUI(context.Background(), fyneApp, device, window, application, cfgPath, cfg, noopPicker{})
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

func waitForRunCompletion(t *testing.T, ui *UI) {
	t.Helper()
	waitFor(t, 3*time.Second, func() bool {
		return !ui.running
	})
	fyne.DoAndWait(func() {})
}

func waitForAnalyzeTerminal(t *testing.T, ui *UI) {
	t.Helper()
	waitForTerminalStatus(t, ui, "Analyze finished", "Analyze failed")
}

func waitForRerenderTerminal(t *testing.T, ui *UI) {
	t.Helper()
	waitForTerminalStatus(t, ui, "Re-render finished", "Re-render failed")
}

func waitForTerminalStatus(t *testing.T, ui *UI, success, failure string) {
	t.Helper()
	waitFor(t, 3*time.Second, func() bool {
		status := statusText(ui)
		return status == success || status == failure
	})
	waitForRunCompletion(t, ui)
}

func detailsText(ui *UI) string {
	var details string
	fyne.DoAndWait(func() {
		details = ui.detailsEntry.Text
	})
	return details
}

func statusText(ui *UI) string {
	var status string
	fyne.DoAndWait(func() {
		status = ui.statusLabel.Text
	})
	return status
}

func layoutJSONPathText(ui *UI) string {
	var path string
	fyne.DoAndWait(func() {
		path = ui.layoutJSONEntry.Text
	})
	return path
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func sameColor(got color.Color, want color.Color) bool {
	return color.NRGBAModel.Convert(got) == color.NRGBAModel.Convert(want)
}
