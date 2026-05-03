package gui

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	appcore "godocmirrortranslator/internal/app"
	"godocmirrortranslator/internal/config"
	base "godocmirrortranslator/internal/renderer"
)

const (
	appID        = "com.vitovt.godocmirrortranslator"
	windowTitle  = "Go Document Mirror Translator"
	narrowWidth  = 860
	minMenuWidth = 220
)

var supportedImageExtensions = []string{".jpg", ".jpeg", ".png", ".webp"}

var pageLayoutOptions = []string{"auto", "Portrait", "Landscape"}

type UI struct {
	ctx         context.Context
	app         fyne.App
	device      fyne.Device
	window      fyne.Window
	application *appcore.Application
	picker      picker

	configPath       string
	cfg              config.Config
	confirmOverwrite func(title string, message string, onDone func(bool))

	menuToggle        *widget.Button
	menuBackground    *canvas.Rectangle
	menuPanel         *fyne.Container
	mainMenuButton    *widget.Button
	aiMenuButton      *widget.Button
	designMenuButton  *widget.Button
	mainContentView   *fyne.Container
	aiContentView     *fyne.Container
	designContentView *fyne.Container
	contentPanel      *fyne.Container
	centerPanel       *fyne.Container
	layoutRoot        *responsiveRoot
	detailsToggle     *widget.Button
	detailsPanel      *fyne.Container
	currentSection    settingsSection
	menuVisible       bool
	compactLayout     bool
	detailsVisible    bool

	inputBrowseButton       *widget.Button
	layoutJSONBrowseButton  *widget.Button
	outputBrowseButton      *widget.Button
	inputEntry              *widget.Entry
	layoutJSONEntry         *widget.Entry
	outputDirEntry          *widget.Entry
	templateEntry           *widget.Entry
	imageDescriptionEntry   *widget.Entry
	rendererSelect          *widget.Select
	providerSelect          *widget.Select
	modelSelect             *widget.Select
	sourceLangEntry         *widget.Entry
	targetLangEntry         *widget.Entry
	timeoutEntry            *widget.Entry
	fontFamilyEntry         *widget.Entry
	fontSizeEntry           *widget.Entry
	fontWeightSelect        *widget.Select
	pageLayoutSelect        *widget.Select
	colorEntry              *widget.Entry
	opacityEntry            *widget.Entry
	rendererCapabilityLabel *widget.Label
	outlineColorEntry       *widget.Entry
	outlineWidthEntry       *widget.Entry
	backgroundEnabled       *widget.Check
	backgroundColorEntry    *widget.Entry
	backgroundOpacityEntry  *widget.Entry
	backgroundPaddingXEntry *widget.Entry
	backgroundPaddingYEntry *widget.Entry
	backgroundRadiusEntry   *widget.Entry
	shadowEnabled           *widget.Check
	shadowColorEntry        *widget.Entry
	shadowOpacityEntry      *widget.Entry
	shadowBlurEntry         *widget.Entry
	shadowOffsetXEntry      *widget.Entry
	shadowOffsetYEntry      *widget.Entry
	preserveColumns         *widget.Check
	openAIKeyEntry          *widget.Entry
	geminiKeyEntry          *widget.Entry
	openAIImageDetail       *widget.Select
	saveButton              *widget.Button
	processButton           *widget.Button
	rerenderButton          *widget.Button
	openOutputButton        *widget.Button
	progress                *widget.ProgressBarInfinite
	statusLabel             *widget.Label
	validationLabel         *widget.Label
	detailsEntry            *widget.Entry
	lastOutputDir           string
	lastOutputPath          string
	running                 bool
}

func Run(ctx context.Context, application *appcore.Application, version string, configPath string) error {
	cfg, resolvedPath, err := config.LoadEffective(configPath)
	if err != nil {
		return fmt.Errorf("load gui config: %w", err)
	}

	guiApp := fyneapp.NewWithID(appID)
	window := guiApp.NewWindow(windowTitle)
	if strings.TrimSpace(version) != "" && version != "dev" {
		window.SetTitle(windowTitle + " " + version)
	}
	device := guiApp.Driver().Device()
	if !isMobileDevice(device) {
		window.SetMaster()
		window.Resize(fyne.NewSize(960, 760))
	}

	ui := newUI(ctx, guiApp, device, window, application, resolvedPath, cfg, newDefaultPicker())
	window.SetCloseIntercept(func() {
		if _, err := ui.saveSettings(); err != nil {
			dialog.ShowError(err, window)
			return
		}
		window.Close()
	})
	window.ShowAndRun()
	return nil
}

func newUI(ctx context.Context, guiApp fyne.App, device fyne.Device, window fyne.Window, application *appcore.Application, configPath string, cfg config.Config, filePicker picker) *UI {
	ui := &UI{
		ctx:         ctx,
		app:         guiApp,
		device:      device,
		window:      window,
		application: application,
		picker:      filePicker,
		configPath:  configPath,
		cfg:         cfg,
	}
	ui.confirmOverwrite = func(title string, message string, onDone func(bool)) {
		dialog.ShowConfirm(title, message, onDone, ui.window)
	}
	ui.menuVisible = true
	ui.compactLayout = isMobileDevice(device)
	ui.detailsVisible = !ui.compactLayout
	ui.currentSection = sectionMain

	ui.inputEntry = widget.NewEntry()
	ui.inputEntry.SetPlaceHolder("Select a source image")
	ui.layoutJSONEntry = widget.NewEntry()
	ui.layoutJSONEntry.SetPlaceHolder("Optional; filled after Analyze or browse an existing JSON")
	ui.inputBrowseButton = widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		ui.pickInputImage()
	})
	ui.layoutJSONBrowseButton = widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		ui.pickLayoutJSON()
	})
	ui.outputDirEntry = widget.NewEntry()
	ui.outputDirEntry.SetPlaceHolder("Optional; defaults to the input folder")
	ui.outputBrowseButton = widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		ui.pickOutputDir()
	})
	ui.templateEntry = widget.NewEntry()
	ui.imageDescriptionEntry = widget.NewMultiLineEntry()
	ui.imageDescriptionEntry.SetMinRowsVisible(4)
	ui.imageDescriptionEntry.SetPlaceHolder("Optional context about the document type, expected fields, abbreviations, or handwriting conventions")
	ui.rendererSelect = widget.NewSelect(application.RendererNames(), func(string) {
		ui.refreshRendererGuidance()
		ui.refreshValidation()
	})
	ui.providerSelect = widget.NewSelect(application.ProviderNames(), func(string) {
		ui.syncModelOptions()
		ui.syncAdvancedOptions()
		ui.refreshValidation()
	})
	ui.modelSelect = widget.NewSelect(nil, func(string) {
		ui.refreshValidation()
	})
	ui.sourceLangEntry = widget.NewEntry()
	ui.targetLangEntry = widget.NewEntry()
	ui.timeoutEntry = widget.NewEntry()
	ui.fontFamilyEntry = widget.NewEntry()
	ui.fontSizeEntry = widget.NewEntry()
	ui.fontWeightSelect = widget.NewSelect([]string{"normal", "bold"}, func(string) {
		ui.refreshValidation()
	})
	ui.pageLayoutSelect = widget.NewSelect(pageLayoutOptions, func(string) {
		ui.refreshValidation()
	})
	ui.colorEntry = widget.NewEntry()
	ui.opacityEntry = widget.NewEntry()
	ui.rendererCapabilityLabel = widget.NewLabel("")
	ui.rendererCapabilityLabel.Wrapping = fyne.TextWrapWord
	ui.outlineColorEntry = widget.NewEntry()
	ui.outlineWidthEntry = widget.NewEntry()
	ui.backgroundEnabled = widget.NewCheck("", func(bool) {
		ui.refreshValidation()
	})
	ui.backgroundColorEntry = widget.NewEntry()
	ui.backgroundOpacityEntry = widget.NewEntry()
	ui.backgroundPaddingXEntry = widget.NewEntry()
	ui.backgroundPaddingYEntry = widget.NewEntry()
	ui.backgroundRadiusEntry = widget.NewEntry()
	ui.shadowEnabled = widget.NewCheck("", func(bool) {
		ui.refreshValidation()
	})
	ui.shadowColorEntry = widget.NewEntry()
	ui.shadowOpacityEntry = widget.NewEntry()
	ui.shadowBlurEntry = widget.NewEntry()
	ui.shadowOffsetXEntry = widget.NewEntry()
	ui.shadowOffsetYEntry = widget.NewEntry()
	ui.preserveColumns = widget.NewCheck("", func(bool) {
		ui.refreshValidation()
	})
	ui.openAIKeyEntry = widget.NewPasswordEntry()
	ui.geminiKeyEntry = widget.NewPasswordEntry()
	ui.openAIImageDetail = widget.NewSelect([]string{"auto", "low", "high"}, func(string) {
		ui.refreshValidation()
	})
	ui.menuToggle = widget.NewButtonWithIcon("", theme.MenuIcon(), func() {
		ui.toggleMenu()
	})
	ui.detailsToggle = widget.NewButton("Show Details", func() {
		ui.toggleDetails()
	})
	ui.mainMenuButton = widget.NewButton("Main", func() {
		ui.selectSection(sectionMain)
	})
	ui.aiMenuButton = widget.NewButton("AI Settings", func() {
		ui.selectSection(sectionAI)
	})
	ui.designMenuButton = widget.NewButton("Design Settings", func() {
		ui.selectSection(sectionDesign)
	})

	ui.saveButton = widget.NewButtonWithIcon("Save Settings", theme.DocumentSaveIcon(), func() {
		if _, err := ui.saveSettings(); err != nil {
			dialog.ShowError(err, ui.window)
			ui.setStatus("Failed to save settings", err.Error())
			return
		}
		ui.setStatus("Settings saved", ui.configPath)
	})
	ui.processButton = widget.NewButtonWithIcon("Process", theme.MediaPlayIcon(), func() {
		ui.startAnalyze()
	})
	ui.processButton.SetText("Analyze")
	ui.processButton.Importance = widget.HighImportance
	ui.rerenderButton = widget.NewButtonWithIcon("Re-render", theme.ViewRefreshIcon(), func() {
		ui.startRerender()
	})
	ui.openOutputButton = widget.NewButtonWithIcon(ui.openOutputActionLabel(), theme.FolderOpenIcon(), func() {
		if err := ui.openOutputDir(); err != nil {
			dialog.ShowError(err, ui.window)
			ui.setStatus("Failed to open output", err.Error())
		}
	})
	ui.openOutputButton.Disable()
	ui.progress = widget.NewProgressBarInfinite()
	ui.progress.Hide()
	ui.statusLabel = widget.NewLabel("Ready")
	ui.validationLabel = widget.NewLabel("")
	ui.detailsEntry = widget.NewMultiLineEntry()
	ui.detailsEntry.SetMinRowsVisible(6)
	ui.detailsEntry.Disable()

	ui.installChangeHandlers()
	ui.layoutRoot = newResponsiveRoot(ui.content(), ui.handleResponsiveLayout)
	window.SetContent(ui.layoutRoot)
	guiApp.Settings().AddListener(func(fyne.Settings) {
		ui.refreshThemeSurfaces()
	})
	ui.refreshThemeSurfaces()
	ui.applyConfig(cfg)
	ui.refreshRendererGuidance()
	ui.syncModelOptions()
	ui.syncAdvancedOptions()
	ui.applyShellState()
	ui.refreshValidation()

	return ui
}

func (u *UI) content() fyne.CanvasObject {
	inputRow := container.NewBorder(nil, nil, nil, u.inputBrowseButton, u.inputEntry)
	layoutJSONRow := container.NewBorder(nil, nil, nil, u.layoutJSONBrowseButton, u.layoutJSONEntry)
	outputRow := container.NewBorder(nil, nil, nil, u.outputBrowseButton, u.outputDirEntry)
	imageDescriptionHint := widget.NewLabel("Optional. Describe what kind of document this is and any field names, abbreviations, or conventions that may help AI recognize visible text more accurately. This text is used only during Analyze and is omitted entirely when left empty.")
	imageDescriptionHint.Wrapping = fyne.TextWrapWord

	mainForm := widget.NewForm(
		widget.NewFormItem("Input Image", inputRow),
		widget.NewFormItem("Layout JSON", layoutJSONRow),
		widget.NewFormItem("Output Folder", outputRow),
		widget.NewFormItem("Filename Template", u.templateEntry),
		widget.NewFormItem("Output Format", u.rendererSelect),
		widget.NewFormItem("Source Language", u.sourceLangEntry),
		widget.NewFormItem("Target Language", u.targetLangEntry),
		widget.NewFormItem("Image Description for AI", container.NewVBox(u.imageDescriptionEntry, imageDescriptionHint)),
		widget.NewFormItem("Preserve Columns", u.preserveColumns),
	)

	aiForm := widget.NewForm(
		widget.NewFormItem("Provider", u.providerSelect),
		widget.NewFormItem("Model", u.modelSelect),
		widget.NewFormItem("Timeout", u.timeoutEntry),
		widget.NewFormItem("OpenAI Image Detail", u.openAIImageDetail),
		widget.NewFormItem("OpenAI API Key", u.openAIKeyEntry),
		widget.NewFormItem("Gemini API Key", u.geminiKeyEntry),
	)

	designForm := widget.NewForm(
		widget.NewFormItem("Font Family", u.fontFamilyEntry),
		widget.NewFormItem("Font Size (pt)", u.fontSizeEntry),
		widget.NewFormItem("Font Weight", u.fontWeightSelect),
		widget.NewFormItem("Page Layout", u.pageLayoutSelect),
		widget.NewFormItem("Text Color", u.colorEntry),
		widget.NewFormItem("Text Opacity (%)", u.opacityEntry),
		widget.NewFormItem("Outline Color", u.outlineColorEntry),
		widget.NewFormItem("Outline Width", u.outlineWidthEntry),
		widget.NewFormItem("Text Shadow", u.shadowEnabled),
		widget.NewFormItem("Shadow Color", u.shadowColorEntry),
		widget.NewFormItem("Shadow Opacity (%)", u.shadowOpacityEntry),
		widget.NewFormItem("Shadow Blur", u.shadowBlurEntry),
		widget.NewFormItem("Shadow Offset X", u.shadowOffsetXEntry),
		widget.NewFormItem("Shadow Offset Y", u.shadowOffsetYEntry),
		widget.NewFormItem("Text Background", u.backgroundEnabled),
		widget.NewFormItem("Background Color", u.backgroundColorEntry),
		widget.NewFormItem("Background Opacity (%)", u.backgroundOpacityEntry),
		widget.NewFormItem("Background Padding X", u.backgroundPaddingXEntry),
		widget.NewFormItem("Background Padding Y", u.backgroundPaddingYEntry),
		widget.NewFormItem("Background Radius", u.backgroundRadiusEntry),
	)
	designSection := container.NewVBox(
		designForm,
		widget.NewSeparator(),
		widget.NewLabel("Renderer Formatting Notes"),
		u.rendererCapabilityLabel,
	)
	u.mainContentView = container.NewPadded(container.NewVScroll(mainForm))
	u.aiContentView = container.NewPadded(container.NewVScroll(aiForm))
	u.designContentView = container.NewPadded(container.NewVScroll(designSection))
	u.contentPanel = container.NewStack(
		u.mainContentView,
		u.aiContentView,
		u.designContentView,
	)
	u.menuBackground = canvas.NewRectangle(theme.Color(theme.ColorNameMenuBackground))
	u.menuBackground.SetMinSize(fyne.NewSize(minMenuWidth, 0))
	u.menuPanel = container.NewStack(
		u.menuBackground,
		container.NewPadded(container.NewVBox(
			u.mainMenuButton,
			u.aiMenuButton,
			u.designMenuButton,
			layout.NewSpacer(),
		)),
	)
	u.centerPanel = container.NewStack(u.contentPanel)

	actions := container.NewHBox(
		u.saveButton,
		u.processButton,
		u.rerenderButton,
		u.openOutputButton,
		layout.NewSpacer(),
		u.progress,
	)

	topBar := container.NewBorder(
		nil,
		widget.NewSeparator(),
		nil,
		nil,
		container.NewHBox(
			u.menuToggle,
			layout.NewSpacer(),
		),
	)

	statusPanel := container.NewVBox(
		u.statusLabel,
		u.validationLabel,
	)
	u.detailsPanel = container.NewVBox(
		u.detailsToggle,
		u.detailsEntry,
	)
	bottom := container.NewVBox(
		actions,
		widget.NewSeparator(),
		statusPanel,
		u.detailsPanel,
	)

	return container.NewBorder(topBar, bottom, nil, nil, u.centerPanel)
}

func (u *UI) installChangeHandlers() {
	changeHandlers := []struct {
		entry *widget.Entry
	}{
		{u.inputEntry},
		{u.layoutJSONEntry},
		{u.outputDirEntry},
		{u.templateEntry},
		{u.imageDescriptionEntry},
		{u.sourceLangEntry},
		{u.targetLangEntry},
		{u.timeoutEntry},
		{u.fontFamilyEntry},
		{u.fontSizeEntry},
		{u.colorEntry},
		{u.opacityEntry},
		{u.outlineColorEntry},
		{u.outlineWidthEntry},
		{u.backgroundColorEntry},
		{u.backgroundOpacityEntry},
		{u.backgroundPaddingXEntry},
		{u.backgroundPaddingYEntry},
		{u.backgroundRadiusEntry},
		{u.shadowColorEntry},
		{u.shadowOpacityEntry},
		{u.shadowBlurEntry},
		{u.shadowOffsetXEntry},
		{u.shadowOffsetYEntry},
		{u.openAIKeyEntry},
		{u.geminiKeyEntry},
	}
	for _, item := range changeHandlers {
		item.entry.OnChanged = func(string) {
			u.syncModelOptions()
			u.refreshValidation()
		}
	}
}

func (u *UI) applyConfig(cfg config.Config) {
	u.inputEntry.SetText("")
	u.layoutJSONEntry.SetText("")
	u.outputDirEntry.SetText(cfg.DefaultOutputDir)
	u.templateEntry.SetText(cfg.OutputTemplate)
	u.imageDescriptionEntry.SetText(cfg.ImageDescription)
	u.rendererSelect.SetSelected(defaultRendererName(cfg.DefaultRenderer))
	u.sourceLangEntry.SetText(cfg.SourceLanguage)
	u.targetLangEntry.SetText(cfg.TargetLanguage)
	u.timeoutEntry.SetText(cfg.Timeout.String())
	u.fontFamilyEntry.SetText(cfg.DefaultFontFamily)
	u.fontSizeEntry.SetText(fmt.Sprintf("%g", cfg.DefaultFontSize))
	u.fontWeightSelect.SetSelected(cfg.DefaultFontWeight)
	u.pageLayoutSelect.SetSelected(pageLayoutLabel(cfg.DefaultPageLayout))
	u.colorEntry.SetText(cfg.OverlayColor)
	u.opacityEntry.SetText(formatPercent(cfg.OverlayOpacity))
	u.outlineColorEntry.SetText(cfg.TextOutlineColor)
	u.outlineWidthEntry.SetText(fmt.Sprintf("%g", cfg.TextOutlineWidth))
	u.backgroundEnabled.SetChecked(cfg.TextBackgroundEnabled)
	u.backgroundColorEntry.SetText(cfg.TextBackgroundColor)
	u.backgroundOpacityEntry.SetText(formatPercent(cfg.TextBackgroundOpacity))
	u.backgroundPaddingXEntry.SetText(fmt.Sprintf("%g", cfg.TextBackgroundPaddingX))
	u.backgroundPaddingYEntry.SetText(fmt.Sprintf("%g", cfg.TextBackgroundPaddingY))
	u.backgroundRadiusEntry.SetText(fmt.Sprintf("%g", cfg.TextBackgroundRadius))
	u.shadowEnabled.SetChecked(cfg.TextShadowEnabled)
	u.shadowColorEntry.SetText(cfg.TextShadowColor)
	u.shadowOpacityEntry.SetText(formatPercent(cfg.TextShadowOpacity))
	u.shadowBlurEntry.SetText(fmt.Sprintf("%g", cfg.TextShadowBlur))
	u.shadowOffsetXEntry.SetText(fmt.Sprintf("%g", cfg.TextShadowOffsetX))
	u.shadowOffsetYEntry.SetText(fmt.Sprintf("%g", cfg.TextShadowOffsetY))
	u.preserveColumns.SetChecked(cfg.PreserveColumns)
	u.openAIKeyEntry.SetText(cfg.OpenAIAPIKey)
	u.geminiKeyEntry.SetText(cfg.GeminiAPIKey)
	u.openAIImageDetail.SetSelected(configValue(cfg.ProviderOptions, "openai", "image_detail", "auto"))

	providerName := cfg.DefaultProvider
	if providerName == "" {
		providerName = "mock"
	}
	u.providerSelect.SetSelected(providerName)
}

func (u *UI) syncModelOptions() {
	cfg, err := u.configFromWidgets()
	if err != nil {
		cfg = u.cfg
	}
	models := u.application.SupportedModels(u.providerSelect.Selected, cfg.ProviderConfig(u.providerSelect.Selected, u.modelSelect.Selected))
	if len(models) == 0 {
		u.modelSelect.SetOptions(nil)
		u.modelSelect.ClearSelected()
		return
	}

	current := strings.TrimSpace(u.modelSelect.Selected)
	u.modelSelect.SetOptions(models)
	for _, candidate := range models {
		if candidate == current {
			u.modelSelect.SetSelected(candidate)
			return
		}
	}
	defaultModel := strings.TrimSpace(cfg.DefaultModel)
	for _, candidate := range models {
		if candidate == defaultModel {
			u.modelSelect.SetSelected(candidate)
			return
		}
	}
	u.modelSelect.SetSelected(models[0])
}

func (u *UI) syncAdvancedOptions() {
	if u.providerSelect.Selected == "openai" {
		u.openAIImageDetail.Enable()
		return
	}
	u.openAIImageDetail.Disable()
}

func (u *UI) refreshValidation() {
	u.refreshInteractivity()
	settingsErr := u.settingsValidationError()
	analyzeErr := u.analyzeValidationError()
	rerenderErr := u.rerenderValidationError()

	if u.running || settingsErr != nil {
		u.saveButton.Disable()
	} else {
		u.saveButton.Enable()
	}
	if u.running || analyzeErr != nil {
		u.processButton.Disable()
	} else {
		u.processButton.Enable()
	}
	if u.running || rerenderErr != nil {
		u.rerenderButton.Disable()
	} else {
		u.rerenderButton.Enable()
	}

	switch {
	case u.running:
		u.validationLabel.SetText("Processing in progress...")
	case settingsErr != nil:
		u.validationLabel.SetText(settingsErr.Error())
	case strings.TrimSpace(u.layoutJSONEntry.Text) != "" && rerenderErr != nil:
		u.validationLabel.SetText(rerenderErr.Error())
	case analyzeErr != nil:
		u.validationLabel.SetText(analyzeErr.Error())
	case rerenderErr != nil:
		u.validationLabel.SetText(rerenderErr.Error())
	default:
		u.validationLabel.SetText("")
	}
}

func (u *UI) refreshInteractivity() {
	for _, control := range u.interactiveControls() {
		if u.running {
			control.Disable()
			continue
		}
		control.Enable()
	}

	if u.running {
		u.openOutputButton.Disable()
		return
	}

	u.syncAdvancedOptions()
	u.preserveColumns.Disable()
	if u.hasOutputTarget() {
		u.openOutputButton.Enable()
		return
	}
	u.openOutputButton.Disable()
}

func (u *UI) settingsValidationError() error {
	cfg, err := u.configFromWidgets()
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.DefaultRenderer) == "" {
		return fmt.Errorf("output format is required")
	}
	if !containsString(u.application.RendererNames(), cfg.DefaultRenderer) {
		return fmt.Errorf("unknown output format %q", cfg.DefaultRenderer)
	}
	if strings.TrimSpace(cfg.SourceLanguage) == "" || strings.TrimSpace(cfg.TargetLanguage) == "" {
		return fmt.Errorf("source and target languages are required")
	}
	if strings.TrimSpace(cfg.OutputTemplate) == "" {
		return fmt.Errorf("filename template is required")
	}
	if strings.TrimSpace(cfg.DefaultFontFamily) == "" {
		return fmt.Errorf("font family is required")
	}
	if cfg.DefaultFontSize <= 0 {
		return fmt.Errorf("font size must be positive")
	}
	if cfg.OverlayOpacity < 0 || cfg.OverlayOpacity > 1 {
		return fmt.Errorf("text opacity must be between 0%% and 100%%")
	}
	if strings.TrimSpace(cfg.DefaultFontWeight) == "" {
		return fmt.Errorf("font weight is required")
	}
	if !base.IsValidPageLayout(base.PageLayout(cfg.DefaultPageLayout)) {
		return fmt.Errorf("page layout must be auto, portrait, or landscape")
	}
	if cfg.TextOutlineWidth < 0 {
		return fmt.Errorf("outline width must be non-negative")
	}
	if cfg.TextBackgroundOpacity < 0 || cfg.TextBackgroundOpacity > 1 {
		return fmt.Errorf("background opacity must be between 0%% and 100%%")
	}
	if cfg.TextBackgroundPaddingX < 0 || cfg.TextBackgroundPaddingY < 0 {
		return fmt.Errorf("background padding must be non-negative")
	}
	if cfg.TextBackgroundRadius < 0 {
		return fmt.Errorf("background radius must be non-negative")
	}
	if cfg.TextShadowOpacity < 0 || cfg.TextShadowOpacity > 1 {
		return fmt.Errorf("shadow opacity must be between 0%% and 100%%")
	}
	if cfg.TextShadowBlur < 0 {
		return fmt.Errorf("shadow blur must be non-negative")
	}
	if cfg.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if strings.TrimSpace(cfg.DefaultProvider) == "" {
		return fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(cfg.DefaultModel) == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

func (u *UI) analyzeValidationError() error {
	if err := u.settingsValidationError(); err != nil {
		return err
	}
	cfg, err := u.configFromWidgets()
	if err != nil {
		return err
	}
	if err := u.application.ValidateProviderConfig(cfg.DefaultProvider, cfg.ProviderConfig(cfg.DefaultProvider, cfg.DefaultModel)); err != nil {
		return err
	}
	inputPath := strings.TrimSpace(u.inputEntry.Text)
	if inputPath == "" {
		return fmt.Errorf("input image is required")
	}
	if err := u.application.ValidateInputImage(inputPath); err != nil {
		return err
	}
	return nil
}

func (u *UI) rerenderValidationError() error {
	if err := u.settingsValidationError(); err != nil {
		return err
	}
	layoutJSONPath := strings.TrimSpace(u.layoutJSONEntry.Text)
	if layoutJSONPath == "" {
		return fmt.Errorf("layout json is required")
	}
	if err := u.application.ValidateLayoutJSON(layoutJSONPath); err != nil {
		return err
	}
	return nil
}

func (u *UI) configFromWidgets() (config.Config, error) {
	cfg := u.cfg.Clone()

	timeout, err := time.ParseDuration(strings.TrimSpace(u.timeoutEntry.Text))
	if err != nil {
		return config.Config{}, fmt.Errorf("timeout must be a valid duration")
	}
	fontSize, err := parseFloatEntry(u.fontSizeEntry.Text, "font size")
	if err != nil {
		return config.Config{}, err
	}
	opacity, err := parsePercentEntry(u.opacityEntry.Text, "text opacity")
	if err != nil {
		return config.Config{}, err
	}
	outlineWidth, err := parseFloatEntry(u.outlineWidthEntry.Text, "outline width")
	if err != nil {
		return config.Config{}, err
	}
	backgroundOpacity, err := parsePercentEntry(u.backgroundOpacityEntry.Text, "background opacity")
	if err != nil {
		return config.Config{}, err
	}
	backgroundPaddingX, err := parseFloatEntry(u.backgroundPaddingXEntry.Text, "background padding x")
	if err != nil {
		return config.Config{}, err
	}
	backgroundPaddingY, err := parseFloatEntry(u.backgroundPaddingYEntry.Text, "background padding y")
	if err != nil {
		return config.Config{}, err
	}
	backgroundRadius, err := parseFloatEntry(u.backgroundRadiusEntry.Text, "background radius")
	if err != nil {
		return config.Config{}, err
	}
	shadowOpacity, err := parsePercentEntry(u.shadowOpacityEntry.Text, "shadow opacity")
	if err != nil {
		return config.Config{}, err
	}
	shadowBlur, err := parseFloatEntry(u.shadowBlurEntry.Text, "shadow blur")
	if err != nil {
		return config.Config{}, err
	}
	shadowOffsetX, err := parseFloatEntry(u.shadowOffsetXEntry.Text, "shadow offset x")
	if err != nil {
		return config.Config{}, err
	}
	shadowOffsetY, err := parseFloatEntry(u.shadowOffsetYEntry.Text, "shadow offset y")
	if err != nil {
		return config.Config{}, err
	}

	cfg.DefaultOutputDir = strings.TrimSpace(u.outputDirEntry.Text)
	cfg.OutputTemplate = strings.TrimSpace(u.templateEntry.Text)
	cfg.ImageDescription = strings.TrimSpace(u.imageDescriptionEntry.Text)
	cfg.DefaultRenderer = strings.TrimSpace(u.rendererSelect.Selected)
	cfg.DefaultProvider = strings.TrimSpace(u.providerSelect.Selected)
	cfg.DefaultModel = strings.TrimSpace(u.modelSelect.Selected)
	cfg.SourceLanguage = strings.TrimSpace(u.sourceLangEntry.Text)
	cfg.TargetLanguage = strings.TrimSpace(u.targetLangEntry.Text)
	cfg.Timeout = timeout
	cfg.DefaultFontFamily = strings.TrimSpace(u.fontFamilyEntry.Text)
	cfg.DefaultFontSize = fontSize
	cfg.DefaultFontWeight = strings.TrimSpace(u.fontWeightSelect.Selected)
	cfg.DefaultPageLayout = selectedPageLayout(u.pageLayoutSelect.Selected)
	cfg.OverlayColor = strings.TrimSpace(u.colorEntry.Text)
	cfg.OverlayOpacity = opacity
	cfg.TextOutlineColor = strings.TrimSpace(u.outlineColorEntry.Text)
	cfg.TextOutlineWidth = outlineWidth
	cfg.TextBackgroundEnabled = u.backgroundEnabled.Checked
	cfg.TextBackgroundColor = strings.TrimSpace(u.backgroundColorEntry.Text)
	cfg.TextBackgroundOpacity = backgroundOpacity
	cfg.TextBackgroundPaddingX = backgroundPaddingX
	cfg.TextBackgroundPaddingY = backgroundPaddingY
	cfg.TextBackgroundRadius = backgroundRadius
	cfg.TextShadowEnabled = u.shadowEnabled.Checked
	cfg.TextShadowColor = strings.TrimSpace(u.shadowColorEntry.Text)
	cfg.TextShadowOpacity = shadowOpacity
	cfg.TextShadowBlur = shadowBlur
	cfg.TextShadowOffsetX = shadowOffsetX
	cfg.TextShadowOffsetY = shadowOffsetY
	cfg.PreserveColumns = u.preserveColumns.Checked
	cfg.OpenAIAPIKey = strings.TrimSpace(u.openAIKeyEntry.Text)
	cfg.GeminiAPIKey = strings.TrimSpace(u.geminiKeyEntry.Text)
	if cfg.ProviderOptions == nil {
		cfg.ProviderOptions = map[string]map[string]string{}
	}
	if cfg.ProviderOptions["openai"] == nil {
		cfg.ProviderOptions["openai"] = map[string]string{}
	}
	cfg.ProviderOptions["openai"]["image_detail"] = strings.TrimSpace(u.openAIImageDetail.Selected)

	return cfg, nil
}

func (u *UI) buildRenderRequest(cfg config.Config) appcore.RenderRequest {
	return appcore.RenderRequest{
		InputPath:        strings.TrimSpace(u.inputEntry.Text),
		OutputDir:        strings.TrimSpace(u.outputDirEntry.Text),
		OutputTemplate:   cfg.OutputTemplate,
		RendererName:     cfg.DefaultRenderer,
		ProviderName:     cfg.DefaultProvider,
		ProviderConfig:   cfg.ProviderConfig(cfg.DefaultProvider, cfg.DefaultModel),
		Model:            cfg.DefaultModel,
		SourceLanguage:   cfg.SourceLanguage,
		TargetLanguage:   cfg.TargetLanguage,
		ImageDescription: cfg.ImageDescription,
		Timeout:          cfg.Timeout,
		RenderOptions:    cfg.RenderOptions(),
		SaveLayoutJSON:   true,
	}
}

func (u *UI) buildRerenderRequest(cfg config.Config) appcore.RerenderRequest {
	return appcore.RerenderRequest{
		LayoutJSONPath: strings.TrimSpace(u.layoutJSONEntry.Text),
		OutputDir:      strings.TrimSpace(u.outputDirEntry.Text),
		OutputTemplate: cfg.OutputTemplate,
		RendererName:   cfg.DefaultRenderer,
		RenderOptions:  cfg.RenderOptions(),
	}
}

func (u *UI) saveSettings() (config.Config, error) {
	cfg, err := u.configFromWidgets()
	if err != nil {
		return config.Config{}, err
	}
	resolvedPath, err := config.Save(u.configPath, cfg)
	if err != nil {
		return config.Config{}, err
	}
	u.configPath = resolvedPath
	u.cfg = cfg
	return cfg, nil
}

func (u *UI) startProcessing() {
	u.startAnalyze()
}

func (u *UI) startAnalyze() {
	if u.running {
		return
	}
	cfg, err := u.saveSettings()
	if err != nil {
		dialog.ShowError(err, u.window)
		u.setStatus("Cannot process", err.Error())
		u.refreshValidation()
		return
	}
	req := u.buildRenderRequest(cfg)
	targets, err := u.application.PlannedRenderTargets(req)
	if err != nil {
		dialog.ShowError(err, u.window)
		u.setStatus("Cannot analyze", err.Error())
		u.refreshValidation()
		return
	}
	u.maybeConfirmOverwrite("Analyze", []string{targets.OutputPath, targets.LayoutJSONPath}, func(overwrite bool) {
		req.OverwriteExisting = overwrite
		u.runAnalyze(req)
	})
}

func (u *UI) runAnalyze(req appcore.RenderRequest) {
	u.running = true
	u.lastOutputDir = ""
	u.lastOutputPath = ""
	u.openOutputButton.Disable()
	u.progress.Show()
	u.setStatus("Analyzing document...", "")
	u.refreshValidation()

	go func() {
		result, err := u.application.Render(u.ctx, req)
		fyne.DoAndWait(func() {
			u.running = false
			u.progress.Hide()
			if err != nil {
				u.setStatus("Analyze failed", err.Error())
				dialog.ShowError(err, u.window)
				u.refreshValidation()
				return
			}

			u.lastOutputDir = filepath.Dir(result.OutputPath)
			u.lastOutputPath = result.OutputPath
			u.layoutJSONEntry.SetText(result.LayoutJSONPath)
			u.openOutputButton.Enable()
			details := []string{result.OutputPath}
			if result.LayoutJSONPath != "" {
				details = append(details, result.LayoutJSONPath)
			}
			u.setStatus("Analyze finished", strings.Join(details, "\n"))
			u.refreshValidation()
		})
	}()
}

func (u *UI) startRerender() {
	if u.running {
		return
	}
	cfg, err := u.saveSettings()
	if err != nil {
		dialog.ShowError(err, u.window)
		u.setStatus("Cannot re-render", err.Error())
		u.refreshValidation()
		return
	}
	req := u.buildRerenderRequest(cfg)
	targets, err := u.application.PlannedRerenderTargets(req)
	if err != nil {
		dialog.ShowError(err, u.window)
		u.setStatus("Cannot re-render", err.Error())
		u.refreshValidation()
		return
	}
	u.maybeConfirmOverwrite("Re-render", []string{targets.OutputPath}, func(overwrite bool) {
		req.OverwriteExisting = overwrite
		u.runRerender(req)
	})
}

func (u *UI) runRerender(req appcore.RerenderRequest) {
	u.running = true
	u.lastOutputDir = ""
	u.lastOutputPath = ""
	u.openOutputButton.Disable()
	u.progress.Show()
	u.setStatus("Re-rendering document...", "")
	u.refreshValidation()

	go func() {
		result, err := u.application.Rerender(u.ctx, req)
		fyne.DoAndWait(func() {
			u.running = false
			u.progress.Hide()
			if err != nil {
				u.setStatus("Re-render failed", err.Error())
				dialog.ShowError(err, u.window)
				u.refreshValidation()
				return
			}

			u.lastOutputDir = filepath.Dir(result.OutputPath)
			u.lastOutputPath = result.OutputPath
			u.openOutputButton.Enable()
			u.setStatus("Re-render finished", strings.Join([]string{result.OutputPath, req.LayoutJSONPath}, "\n"))
			u.refreshValidation()
		})
	}()
}

func (u *UI) setStatus(status, details string) {
	u.statusLabel.SetText(status)
	u.detailsEntry.SetText(details)
}

func (u *UI) pickInputImage() {
	u.picker.PickInputImage(u.window, func(path string, err error) {
		if err != nil {
			dialog.ShowError(err, u.window)
			u.setStatus("Failed to choose input image", err.Error())
			return
		}
		if path == "" {
			return
		}
		u.inputEntry.SetText(path)
		if strings.TrimSpace(u.outputDirEntry.Text) == "" {
			u.outputDirEntry.SetText(filepath.Dir(path))
		}
		u.refreshValidation()
	})
}

func (u *UI) pickOutputDir() {
	u.picker.PickOutputDir(u.window, func(path string, err error) {
		if err != nil {
			dialog.ShowError(err, u.window)
			u.setStatus("Failed to choose output folder", err.Error())
			return
		}
		if path == "" {
			return
		}
		u.outputDirEntry.SetText(path)
		u.refreshValidation()
	})
}

func (u *UI) pickLayoutJSON() {
	u.picker.PickLayoutJSON(u.window, func(path string, err error) {
		if err != nil {
			dialog.ShowError(err, u.window)
			u.setStatus("Failed to choose layout json", err.Error())
			return
		}
		if path == "" {
			return
		}
		u.layoutJSONEntry.SetText(path)
		if strings.TrimSpace(u.outputDirEntry.Text) == "" {
			u.outputDirEntry.SetText(filepath.Dir(path))
		}
		u.refreshValidation()
	})
}

func (u *UI) openOutputDir() error {
	targetPath, err := u.openOutputPath()
	if err != nil {
		return err
	}
	target := (&url.URL{Scheme: "file", Path: filepath.ToSlash(targetPath)})
	if err := u.app.OpenURL(target); err != nil {
		return fmt.Errorf("open output: %w", err)
	}
	return nil
}

func (u *UI) openOutputActionLabel() string {
	if isMobileDevice(u.device) {
		return "Open Output File"
	}
	return "Open Output Folder"
}

func (u *UI) openOutputPath() (string, error) {
	if isMobileDevice(u.device) {
		if strings.TrimSpace(u.lastOutputPath) == "" {
			return "", fmt.Errorf("no output file is available yet")
		}
		return u.lastOutputPath, nil
	}
	if strings.TrimSpace(u.lastOutputDir) == "" {
		return "", fmt.Errorf("no output folder is available yet")
	}
	return u.lastOutputDir, nil
}

func configValue(values map[string]map[string]string, providerName, key, fallback string) string {
	if values == nil {
		return fallback
	}
	if values[providerName] == nil {
		return fallback
	}
	value := strings.TrimSpace(values[providerName][key])
	if value == "" {
		return fallback
	}
	return value
}

func defaultRendererName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "svg"
	}
	return name
}

func defaultPageLayout(layout string) string {
	if !base.IsValidPageLayout(base.PageLayout(strings.TrimSpace(layout))) {
		return string(base.PageLayoutAuto)
	}
	return strings.TrimSpace(layout)
}

func pageLayoutLabel(layout string) string {
	switch defaultPageLayout(layout) {
	case string(base.PageLayoutPortrait):
		return "Portrait"
	case string(base.PageLayoutLandscape):
		return "Landscape"
	default:
		return "auto"
	}
}

func selectedPageLayout(label string) string {
	switch strings.TrimSpace(label) {
	case "Portrait":
		return string(base.PageLayoutPortrait)
	case "Landscape":
		return string(base.PageLayoutLandscape)
	default:
		return string(base.PageLayoutAuto)
	}
}

func rendererGuidanceText(renderer string) string {
	switch defaultRendererName(renderer) {
	case "fodg":
		return strings.Join([]string{
			"FODG / LibreOffice Draw:",
			"Text color, font family, font size, and font weight are applied.",
			"Text opacity is currently ignored by LibreOffice on import.",
			"Outline width works as contour on/off only; it is not a true adjustable stroke width.",
			"Outline color controls the contour color, and contoured text is hollow in LibreOffice.",
			"Background uses character background color only; background opacity, padding, and radius are not supported by the imported text object.",
			"Shadow uses Draw shadow settings; color, opacity, blur, and offsets are applied.",
		}, "\n")
	default:
		return strings.Join([]string{
			"SVG / Inkscape:",
			"Text color and text opacity are applied directly.",
			"Outline color and outline width are applied directly.",
			"Background color, opacity, padding, and radius are applied directly.",
			"Shadow color, opacity, blur, and offsets are applied directly.",
		}, "\n")
	}
}

func (u *UI) refreshRendererGuidance() {
	if u.rendererCapabilityLabel == nil {
		return
	}
	u.rendererCapabilityLabel.SetText(rendererGuidanceText(u.rendererSelect.Selected))
}

func parseFloatEntry(value string, name string) (float64, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be numeric", name)
	}
	return parsed, nil
}

func parsePercentEntry(value string, name string) (float64, error) {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimSuffix(trimmed, "%")
	parsed, err := strconv.ParseFloat(strings.TrimSpace(trimmed), 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be numeric", name)
	}
	if parsed < 0 || parsed > 100 {
		return 0, fmt.Errorf("%s must be between 0 and 100", name)
	}
	return parsed / 100, nil
}

func formatPercent(value float64) string {
	return strconv.FormatFloat(value*100, 'f', -1, 64)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func isMobileDevice(device fyne.Device) bool {
	return device != nil && device.IsMobile()
}

func (u *UI) hasOutputTarget() bool {
	if isMobileDevice(u.device) {
		return strings.TrimSpace(u.lastOutputPath) != ""
	}
	return strings.TrimSpace(u.lastOutputDir) != ""
}

func (u *UI) maybeConfirmOverwrite(action string, paths []string, onContinue func(overwrite bool)) {
	existingPaths, err := existingPaths(paths)
	if err != nil {
		dialog.ShowError(err, u.window)
		u.setStatus("Cannot start "+strings.ToLower(action), err.Error())
		u.refreshValidation()
		return
	}
	if len(existingPaths) == 0 {
		onContinue(false)
		return
	}

	message := "The following files already exist and will be overwritten:\n\n" + strings.Join(existingPaths, "\n") + "\n\nOverwrite them?"
	u.confirmOverwrite(action+" overwrite", message, func(confirm bool) {
		if !confirm {
			u.setStatus(action+" cancelled", "Existing files were left unchanged.")
			u.refreshValidation()
			return
		}
		onContinue(true)
	})
}

func existingPaths(paths []string) ([]string, error) {
	var existing []string
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			existing = append(existing, path)
			continue
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat existing output %q: %w", path, err)
		}
	}
	return existing, nil
}

func (u *UI) toggleMenu() {
	u.menuVisible = !u.menuVisible
	u.applyShellState()
}

func (u *UI) handleResponsiveLayout(size fyne.Size) {
	compact := u.isCompactLayout(size)
	if compact != u.compactLayout {
		u.compactLayout = compact
		if compact {
			u.detailsVisible = false
		} else {
			u.detailsVisible = true
		}
	}
	u.applyShellState()
}

func (u *UI) isCompactLayout(size fyne.Size) bool {
	if isMobileDevice(u.device) {
		return true
	}
	if size.Width <= 0 {
		return false
	}
	return size.Width < narrowWidth
}

func (u *UI) toggleDetails() {
	u.detailsVisible = !u.detailsVisible
	u.applyShellState()
}

func (u *UI) applyShellState() {
	if u.centerPanel != nil {
		u.rebuildCenterPanel()
		u.centerPanel.Refresh()
	}
	u.applySectionState()
	u.applyDetailsState()
	u.updateMenuToggle()
}

func (u *UI) refreshThemeSurfaces() {
	if u.menuBackground == nil {
		return
	}
	u.menuBackground.FillColor = theme.Color(theme.ColorNameMenuBackground)
	u.menuBackground.Refresh()
}

func (u *UI) selectSection(section settingsSection) {
	u.currentSection = section
	u.applySectionState()
	if u.compactLayout {
		u.menuVisible = false
		u.applyShellState()
	}
}

func (u *UI) applySectionState() {
	if u.contentPanel == nil {
		return
	}
	u.setSectionVisible(u.mainContentView, u.currentSection == sectionMain)
	u.setSectionVisible(u.aiContentView, u.currentSection == sectionAI)
	u.setSectionVisible(u.designContentView, u.currentSection == sectionDesign)
	u.contentPanel.Refresh()
	u.updateMenuButtonState()
}

func (u *UI) applyDetailsState() {
	if u.detailsPanel == nil || u.detailsToggle == nil {
		return
	}
	if u.compactLayout {
		u.detailsToggle.Show()
		if u.detailsVisible {
			u.detailsEntry.Show()
			u.detailsToggle.SetText("Hide Details")
		} else {
			u.detailsEntry.Hide()
			u.detailsToggle.SetText("Show Details")
		}
		u.detailsPanel.Refresh()
		return
	}
	u.detailsToggle.Hide()
	u.detailsEntry.Show()
	u.detailsVisible = true
	u.detailsPanel.Refresh()
}

func (u *UI) rebuildCenterPanel() {
	if u.centerPanel == nil || u.contentPanel == nil || u.menuPanel == nil {
		return
	}

	if !u.menuVisible {
		u.menuPanel.Hide()
		u.centerPanel.Objects = []fyne.CanvasObject{u.contentPanel}
		return
	}

	u.menuPanel.Show()
	if u.compactLayout {
		u.centerPanel.Objects = []fyne.CanvasObject{
			u.contentPanel,
			container.NewHBox(u.menuPanel, layout.NewSpacer()),
		}
		return
	}

	u.centerPanel.Objects = []fyne.CanvasObject{
		container.NewBorder(nil, nil, u.menuPanel, nil, u.contentPanel),
	}
}

func (u *UI) updateMenuToggle() {
	if u.menuToggle == nil {
		return
	}
	if u.menuVisible {
		u.menuToggle.SetText("Hide Menu")
		return
	}
	u.menuToggle.SetText("Show Menu")
}

func (u *UI) updateMenuButtonState() {
	u.setMenuButtonState(u.mainMenuButton, u.currentSection == sectionMain)
	u.setMenuButtonState(u.aiMenuButton, u.currentSection == sectionAI)
	u.setMenuButtonState(u.designMenuButton, u.currentSection == sectionDesign)
}

func (u *UI) setMenuButtonState(button *widget.Button, selected bool) {
	if button == nil {
		return
	}
	if selected {
		button.Importance = widget.HighImportance
	} else {
		button.Importance = widget.LowImportance
	}
	button.Refresh()
}

func (u *UI) setSectionVisible(object fyne.CanvasObject, visible bool) {
	if object == nil {
		return
	}
	if visible {
		object.Show()
		return
	}
	object.Hide()
}

func (u *UI) interactiveControls() []disableable {
	return []disableable{
		u.menuToggle,
		u.mainMenuButton,
		u.aiMenuButton,
		u.designMenuButton,
		u.inputBrowseButton,
		u.outputBrowseButton,
		u.inputEntry,
		u.outputDirEntry,
		u.templateEntry,
		u.imageDescriptionEntry,
		u.rendererSelect,
		u.providerSelect,
		u.modelSelect,
		u.sourceLangEntry,
		u.targetLangEntry,
		u.timeoutEntry,
		u.fontFamilyEntry,
		u.fontSizeEntry,
		u.fontWeightSelect,
		u.pageLayoutSelect,
		u.colorEntry,
		u.opacityEntry,
		u.outlineColorEntry,
		u.outlineWidthEntry,
		u.backgroundEnabled,
		u.backgroundColorEntry,
		u.backgroundOpacityEntry,
		u.backgroundPaddingXEntry,
		u.backgroundPaddingYEntry,
		u.backgroundRadiusEntry,
		u.shadowEnabled,
		u.shadowColorEntry,
		u.shadowOpacityEntry,
		u.shadowBlurEntry,
		u.shadowOffsetXEntry,
		u.shadowOffsetYEntry,
		u.preserveColumns,
		u.layoutJSONBrowseButton,
		u.layoutJSONEntry,
		u.openAIKeyEntry,
		u.geminiKeyEntry,
		u.openAIImageDetail,
		u.saveButton,
		u.processButton,
		u.rerenderButton,
	}
}

type disableable interface {
	Disable()
	Enable()
}

type settingsSection string

const (
	sectionMain   settingsSection = "main"
	sectionAI     settingsSection = "ai"
	sectionDesign settingsSection = "design"
)

type responsiveRoot struct {
	widget.BaseWidget
	content  fyne.CanvasObject
	onLayout func(fyne.Size)
}

func newResponsiveRoot(content fyne.CanvasObject, onLayout func(fyne.Size)) *responsiveRoot {
	root := &responsiveRoot{
		content:  content,
		onLayout: onLayout,
	}
	root.ExtendBaseWidget(root)
	return root
}

func (r *responsiveRoot) CreateRenderer() fyne.WidgetRenderer {
	return &responsiveRootRenderer{root: r}
}

type responsiveRootRenderer struct {
	root *responsiveRoot
}

func (r *responsiveRootRenderer) Layout(size fyne.Size) {
	if r.root.onLayout != nil {
		r.root.onLayout(size)
	}
	r.root.content.Move(fyne.NewPos(0, 0))
	r.root.content.Resize(size)
}

func (r *responsiveRootRenderer) MinSize() fyne.Size {
	return r.root.content.MinSize()
}

func (r *responsiveRootRenderer) Refresh() {
	r.root.content.Refresh()
}

func (r *responsiveRootRenderer) Destroy() {}

func (r *responsiveRootRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.root.content}
}
