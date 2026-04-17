package gui

import (
	"context"
	"fmt"
	"net/url"
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
)

const (
	appID        = "com.vitovt.godocmirrortranslator"
	windowTitle  = "Handwritten Overlay Translator"
	narrowWidth  = 860
	minMenuWidth = 220
)

var supportedImageExtensions = []string{".jpg", ".jpeg", ".png", ".webp"}

type UI struct {
	ctx         context.Context
	app         fyne.App
	device      fyne.Device
	window      fyne.Window
	application *appcore.Application
	picker      picker

	configPath string
	cfg        config.Config

	menuToggle        *widget.Button
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

	inputBrowseButton  *widget.Button
	outputBrowseButton *widget.Button
	inputEntry         *widget.Entry
	outputDirEntry     *widget.Entry
	templateEntry      *widget.Entry
	rendererSelect     *widget.Select
	providerSelect     *widget.Select
	modelSelect        *widget.Select
	sourceLangEntry    *widget.Entry
	targetLangEntry    *widget.Entry
	timeoutEntry       *widget.Entry
	fontFamilyEntry    *widget.Entry
	fontSizeEntry      *widget.Entry
	colorEntry         *widget.Entry
	opacityEntry       *widget.Entry
	preserveColumns    *widget.Check
	saveLayoutJSON     *widget.Check
	openAIKeyEntry     *widget.Entry
	geminiKeyEntry     *widget.Entry
	openAIImageDetail  *widget.Select
	saveButton         *widget.Button
	processButton      *widget.Button
	openOutputButton   *widget.Button
	progress           *widget.ProgressBarInfinite
	statusLabel        *widget.Label
	validationLabel    *widget.Label
	detailsEntry       *widget.Entry
	lastOutputDir      string
	lastOutputPath     string
	running            bool
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
	ui.menuVisible = true
	ui.compactLayout = isMobileDevice(device)
	ui.detailsVisible = !ui.compactLayout
	ui.currentSection = sectionMain

	ui.inputEntry = widget.NewEntry()
	ui.inputEntry.SetPlaceHolder("Select a source image")
	ui.inputBrowseButton = widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		ui.pickInputImage()
	})
	ui.outputDirEntry = widget.NewEntry()
	ui.outputDirEntry.SetPlaceHolder("Optional; defaults to the input folder")
	ui.outputBrowseButton = widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		ui.pickOutputDir()
	})
	ui.templateEntry = widget.NewEntry()
	ui.rendererSelect = widget.NewSelect(application.RendererNames(), func(string) {
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
	ui.colorEntry = widget.NewEntry()
	ui.opacityEntry = widget.NewEntry()
	ui.preserveColumns = widget.NewCheck("", func(bool) {
		ui.refreshValidation()
	})
	ui.saveLayoutJSON = widget.NewCheck("", func(bool) {
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
		ui.startProcessing()
	})
	ui.processButton.Importance = widget.HighImportance
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
	ui.applyConfig(cfg)
	ui.syncModelOptions()
	ui.syncAdvancedOptions()
	ui.applyShellState()
	ui.refreshValidation()

	return ui
}

func (u *UI) content() fyne.CanvasObject {
	inputRow := container.NewBorder(nil, nil, nil, u.inputBrowseButton, u.inputEntry)
	outputRow := container.NewBorder(nil, nil, nil, u.outputBrowseButton, u.outputDirEntry)

	mainForm := widget.NewForm(
		widget.NewFormItem("Input Image", inputRow),
		widget.NewFormItem("Output Folder", outputRow),
		widget.NewFormItem("Filename Template", u.templateEntry),
		widget.NewFormItem("Output Format", u.rendererSelect),
		widget.NewFormItem("Source Language", u.sourceLangEntry),
		widget.NewFormItem("Target Language", u.targetLangEntry),
		widget.NewFormItem("Preserve Columns", u.preserveColumns),
	)

	aiForm := widget.NewForm(
		widget.NewFormItem("Provider", u.providerSelect),
		widget.NewFormItem("Model", u.modelSelect),
		widget.NewFormItem("Timeout", u.timeoutEntry),
		widget.NewFormItem("Save Layout JSON", u.saveLayoutJSON),
		widget.NewFormItem("OpenAI Image Detail", u.openAIImageDetail),
		widget.NewFormItem("OpenAI API Key", u.openAIKeyEntry),
		widget.NewFormItem("Gemini API Key", u.geminiKeyEntry),
	)

	designForm := widget.NewForm(
		widget.NewFormItem("Font Family", u.fontFamilyEntry),
		widget.NewFormItem("Font Size", u.fontSizeEntry),
		widget.NewFormItem("Overlay Color", u.colorEntry),
		widget.NewFormItem("Overlay Opacity", u.opacityEntry),
	)
	u.mainContentView = container.NewPadded(container.NewVScroll(mainForm))
	u.aiContentView = container.NewPadded(container.NewVScroll(aiForm))
	u.designContentView = container.NewPadded(container.NewVScroll(designForm))
	u.contentPanel = container.NewStack(
		u.mainContentView,
		u.aiContentView,
		u.designContentView,
	)
	menuBackground := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	menuBackground.SetMinSize(fyne.NewSize(minMenuWidth, 0))
	u.menuPanel = container.NewStack(
		menuBackground,
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
		{u.outputDirEntry},
		{u.templateEntry},
		{u.sourceLangEntry},
		{u.targetLangEntry},
		{u.timeoutEntry},
		{u.fontFamilyEntry},
		{u.fontSizeEntry},
		{u.colorEntry},
		{u.opacityEntry},
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
	u.outputDirEntry.SetText(cfg.DefaultOutputDir)
	u.templateEntry.SetText(cfg.OutputTemplate)
	u.rendererSelect.SetSelected(defaultRendererName(cfg.DefaultRenderer))
	u.sourceLangEntry.SetText(cfg.SourceLanguage)
	u.targetLangEntry.SetText(cfg.TargetLanguage)
	u.timeoutEntry.SetText(cfg.Timeout.String())
	u.fontFamilyEntry.SetText(cfg.DefaultFontFamily)
	u.fontSizeEntry.SetText(fmt.Sprintf("%g", cfg.DefaultFontSize))
	u.colorEntry.SetText(cfg.OverlayColor)
	u.opacityEntry.SetText(fmt.Sprintf("%g", cfg.OverlayOpacity))
	u.preserveColumns.SetChecked(cfg.PreserveColumns)
	u.saveLayoutJSON.Checked = cfg.SaveLayoutJSONEnabled()
	u.saveLayoutJSON.Refresh()
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
	processErr := u.processValidationError()

	if u.running || settingsErr != nil {
		u.saveButton.Disable()
	} else {
		u.saveButton.Enable()
	}
	if u.running || processErr != nil {
		u.processButton.Disable()
	} else {
		u.processButton.Enable()
	}

	switch {
	case u.running:
		u.validationLabel.SetText("Processing in progress...")
	case processErr != nil:
		u.validationLabel.SetText(processErr.Error())
	case settingsErr != nil:
		u.validationLabel.SetText(settingsErr.Error())
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
	if strings.TrimSpace(cfg.DefaultProvider) == "" {
		return fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(cfg.DefaultRenderer) == "" {
		return fmt.Errorf("output format is required")
	}
	if !containsString(u.application.RendererNames(), cfg.DefaultRenderer) {
		return fmt.Errorf("unknown output format %q", cfg.DefaultRenderer)
	}
	if strings.TrimSpace(cfg.DefaultModel) == "" {
		return fmt.Errorf("model is required")
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
		return fmt.Errorf("overlay opacity must be between 0 and 1")
	}
	if cfg.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if err := u.application.ValidateProviderConfig(cfg.DefaultProvider, cfg.ProviderConfig(cfg.DefaultProvider, cfg.DefaultModel)); err != nil {
		return err
	}
	return nil
}

func (u *UI) processValidationError() error {
	if err := u.settingsValidationError(); err != nil {
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

func (u *UI) configFromWidgets() (config.Config, error) {
	cfg := u.cfg

	timeout, err := time.ParseDuration(strings.TrimSpace(u.timeoutEntry.Text))
	if err != nil {
		return config.Config{}, fmt.Errorf("timeout must be a valid duration")
	}
	fontSize, err := strconv.ParseFloat(strings.TrimSpace(u.fontSizeEntry.Text), 64)
	if err != nil {
		return config.Config{}, fmt.Errorf("font size must be numeric")
	}
	opacity, err := strconv.ParseFloat(strings.TrimSpace(u.opacityEntry.Text), 64)
	if err != nil {
		return config.Config{}, fmt.Errorf("overlay opacity must be numeric")
	}

	cfg.DefaultOutputDir = strings.TrimSpace(u.outputDirEntry.Text)
	cfg.OutputTemplate = strings.TrimSpace(u.templateEntry.Text)
	cfg.DefaultRenderer = strings.TrimSpace(u.rendererSelect.Selected)
	cfg.DefaultProvider = strings.TrimSpace(u.providerSelect.Selected)
	cfg.DefaultModel = strings.TrimSpace(u.modelSelect.Selected)
	cfg.SourceLanguage = strings.TrimSpace(u.sourceLangEntry.Text)
	cfg.TargetLanguage = strings.TrimSpace(u.targetLangEntry.Text)
	cfg.Timeout = timeout
	cfg.DefaultFontFamily = strings.TrimSpace(u.fontFamilyEntry.Text)
	cfg.DefaultFontSize = fontSize
	cfg.OverlayColor = strings.TrimSpace(u.colorEntry.Text)
	cfg.OverlayOpacity = opacity
	cfg.PreserveColumns = u.preserveColumns.Checked
	cfg.SetSaveLayoutJSONEnabled(u.saveLayoutJSON.Checked)
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
		InputPath:      strings.TrimSpace(u.inputEntry.Text),
		OutputDir:      strings.TrimSpace(u.outputDirEntry.Text),
		OutputTemplate: cfg.OutputTemplate,
		RendererName:   cfg.DefaultRenderer,
		ProviderName:   cfg.DefaultProvider,
		ProviderConfig: cfg.ProviderConfig(cfg.DefaultProvider, cfg.DefaultModel),
		Model:          cfg.DefaultModel,
		SourceLanguage: cfg.SourceLanguage,
		TargetLanguage: cfg.TargetLanguage,
		Timeout:        cfg.Timeout,
		RenderOptions:  cfg.RenderOptions(),
		SaveLayoutJSON: cfg.SaveLayoutJSONEnabled(),
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

	u.running = true
	u.lastOutputDir = ""
	u.lastOutputPath = ""
	u.openOutputButton.Disable()
	u.progress.Show()
	u.setStatus("Processing document...", "")
	u.refreshValidation()

	go func() {
		result, err := u.application.Render(u.ctx, req)
		fyne.DoAndWait(func() {
			u.running = false
			u.progress.Hide()
			if err != nil {
				u.setStatus("Processing failed", err.Error())
				dialog.ShowError(err, u.window)
				u.refreshValidation()
				return
			}

			u.lastOutputDir = filepath.Dir(result.OutputPath)
			u.lastOutputPath = result.OutputPath
			u.openOutputButton.Enable()
			details := []string{result.OutputPath}
			if result.LayoutJSONPath != "" {
				details = append(details, result.LayoutJSONPath)
			}
			u.setStatus("Processing finished", strings.Join(details, "\n"))
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
		u.rendererSelect,
		u.providerSelect,
		u.modelSelect,
		u.sourceLangEntry,
		u.targetLangEntry,
		u.timeoutEntry,
		u.fontFamilyEntry,
		u.fontSizeEntry,
		u.colorEntry,
		u.opacityEntry,
		u.preserveColumns,
		u.saveLayoutJSON,
		u.openAIKeyEntry,
		u.geminiKeyEntry,
		u.openAIImageDetail,
		u.saveButton,
		u.processButton,
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
