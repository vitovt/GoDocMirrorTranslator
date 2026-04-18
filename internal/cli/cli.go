package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"godocmirrortranslator/internal/app"
	"godocmirrortranslator/internal/config"
	"godocmirrortranslator/internal/gui"
)

const Version = "dev"

func Run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	application := app.New(Version)

	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "render":
		return runRender(ctx, application, args[1:], stdout, stderr)
	case "rerender":
		return runRerender(ctx, application, args[1:], stdout, stderr)
	case "providers":
		if len(args) > 1 && args[1] == "list" {
			for _, name := range application.ProviderNames() {
				_, _ = fmt.Fprintln(stdout, name)
			}
			return 0
		}
	case "version":
		_, _ = fmt.Fprintln(stdout, Version)
		return 0
	case "gui":
		return runGUI(ctx, application, args[1:], stderr)
	case "config":
		return runConfig(args[1:], stdout, stderr)
	case "help", "--help", "-h":
		printHelp(stdout)
		return 0
	}

	_, _ = fmt.Fprintf(stderr, "unknown command: %s\n", strings.Join(args, " "))
	printHelp(stderr)
	return 1
}

func runGUI(ctx context.Context, application *app.Application, args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("gui", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		printGUIUsage(stderr, fs)
	}
	configPath := ""
	fs.StringVar(&configPath, "config", "", "Optional config file path")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if err := gui.Run(ctx, application, Version, configPath); err != nil {
		_, _ = fmt.Fprintf(stderr, "gui: %v\n", err)
		return 1
	}
	return 0
}

func runRender(ctx context.Context, application *app.Application, args []string, stdout io.Writer, stderr io.Writer) int {
	showHelpOnly := hasHelpFlag(args)
	configPath := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--config" {
			if i+1 < len(args) {
				configPath = args[i+1]
			}
			continue
		}
		if strings.HasPrefix(args[i], "--config=") {
			configPath = strings.TrimPrefix(args[i], "--config=")
		}
	}

	cfg := config.Default()
	if !showHelpOnly {
		var err error
		cfg, _, err = config.LoadEffective(configPath)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "render: load config: %v\n", err)
			return 1
		}
	}

	req := app.RenderRequest{
		OutputDir:        cfg.DefaultOutputDir,
		OutputTemplate:   cfg.OutputTemplate,
		ProviderName:     cfg.DefaultProvider,
		RendererName:     cfg.DefaultRenderer,
		Model:            cfg.DefaultModel,
		SourceLanguage:   cfg.SourceLanguage,
		TargetLanguage:   cfg.TargetLanguage,
		ImageDescription: cfg.ImageDescription,
		Timeout:          cfg.Timeout,
		RenderOptions:    cfg.RenderOptions(),
	}

	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		printRenderUsage(stderr, fs)
	}

	fs.StringVar(&req.InputPath, "input", req.InputPath, "Path to the input image")
	fs.StringVar(&req.OutputDir, "output-dir", req.OutputDir, "Directory for generated files")
	fs.StringVar(&req.OutputTemplate, "output-template", req.OutputTemplate, "Filename template for rendered output")
	fs.StringVar(&req.ProviderName, "provider", req.ProviderName, "Provider to use")
	fs.StringVar(&req.RendererName, "renderer", req.RendererName, "Renderer to use")
	fs.StringVar(&req.Model, "model", req.Model, "Model to use")
	fs.StringVar(&req.SourceLanguage, "source-lang", req.SourceLanguage, "Source language")
	fs.StringVar(&req.TargetLanguage, "target-lang", req.TargetLanguage, "Target language")
	fs.StringVar(&req.RenderOptions.FontFamily, "font-family", req.RenderOptions.FontFamily, "Fallback font family")
	fs.Float64Var(&req.RenderOptions.DefaultFontSize, "font-size", req.RenderOptions.DefaultFontSize, "Fallback font size")
	fs.Float64Var(&req.RenderOptions.Opacity, "opacity", req.RenderOptions.Opacity, "Fallback text opacity")
	fs.StringVar(&req.RenderOptions.TextColor, "color", req.RenderOptions.TextColor, "Fallback text color")
	fs.DurationVar(&req.Timeout, "timeout", req.Timeout, "Provider request timeout")
	fs.BoolVar(&req.OverwriteExisting, "overwrite", false, "Overwrite the exact output path instead of generating a suffixed filename")
	fs.BoolVar(&req.SaveLayoutJSON, "save-layout-json", false, "Write layout JSON next to the rendered output")
	verbose := fs.Bool("verbose", false, "Print extra result information")
	fs.StringVar(&configPath, "config", configPath, "Optional config file path")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	req.ProviderConfig = cfg.ProviderConfig(req.ProviderName, req.Model)

	result, err := application.Render(ctx, req)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "render: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintln(stdout, result.OutputPath)
	if *verbose && result.LayoutJSONPath != "" {
		_, _ = fmt.Fprintf(stdout, "layout_json=%s\n", result.LayoutJSONPath)
	}
	return 0
}

func runRerender(ctx context.Context, application *app.Application, args []string, stdout io.Writer, stderr io.Writer) int {
	showHelpOnly := hasHelpFlag(args)
	configPath := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--config" {
			if i+1 < len(args) {
				configPath = args[i+1]
			}
			continue
		}
		if strings.HasPrefix(args[i], "--config=") {
			configPath = strings.TrimPrefix(args[i], "--config=")
		}
	}

	cfg := config.Default()
	if !showHelpOnly {
		var err error
		cfg, _, err = config.LoadEffective(configPath)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "rerender: load config: %v\n", err)
			return 1
		}
	}

	req := app.RerenderRequest{
		OutputDir:      cfg.DefaultOutputDir,
		OutputTemplate: cfg.OutputTemplate,
		RendererName:   cfg.DefaultRenderer,
		RenderOptions:  cfg.RenderOptions(),
	}

	fs := flag.NewFlagSet("rerender", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		printRerenderUsage(stderr, fs)
	}

	fs.StringVar(&req.LayoutJSONPath, "layout-json", "", "Path to the saved layout JSON file")
	fs.StringVar(&req.OutputDir, "output-dir", req.OutputDir, "Directory for generated files")
	fs.StringVar(&req.OutputTemplate, "output-template", req.OutputTemplate, "Filename template for rendered output")
	fs.StringVar(&req.RendererName, "renderer", req.RendererName, "Renderer to use")
	fs.StringVar(&req.RenderOptions.FontFamily, "font-family", req.RenderOptions.FontFamily, "Fallback font family")
	fs.Float64Var(&req.RenderOptions.DefaultFontSize, "font-size", req.RenderOptions.DefaultFontSize, "Fallback font size")
	fs.Float64Var(&req.RenderOptions.Opacity, "opacity", req.RenderOptions.Opacity, "Fallback text opacity")
	fs.StringVar(&req.RenderOptions.TextColor, "color", req.RenderOptions.TextColor, "Fallback text color")
	fs.BoolVar(&req.OverwriteExisting, "overwrite", false, "Overwrite the exact output path instead of generating a suffixed filename")
	verbose := fs.Bool("verbose", false, "Print extra result information")
	fs.StringVar(&configPath, "config", configPath, "Optional config file path")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	result, err := application.Rerender(ctx, req)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "rerender: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintln(stdout, result.OutputPath)
	if *verbose {
		_, _ = fmt.Fprintf(stdout, "layout_json=%s\n", req.LayoutJSONPath)
	}
	return 0
}

func runConfig(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printConfigUsage(stderr)
		return 0
	}
	if len(args) == 1 && hasHelpFlag(args) {
		printConfigUsage(stdout)
		return 0
	}

	switch args[0] {
	case "init":
		return runConfigInit(args[1:], stdout, stderr)
	case "get":
		return runConfigGet(args[1:], stdout, stderr)
	case "set":
		return runConfigSet(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown config subcommand: %s\n", args[0])
		return 2
	}
}

func runConfigInit(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("config init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		printConfigInitUsage(stderr, fs)
	}
	configPath := ""
	fs.StringVar(&configPath, "config", "", "Optional config file path")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	cfg, resolvedPath, err := config.Load(configPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config init: %v\n", err)
		return 1
	}
	resolvedPath, err = config.Save(resolvedPath, cfg)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config init: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, resolvedPath)
	return 0
}

func runConfigGet(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("config get", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		printConfigGetUsage(stderr, fs)
	}
	configPath := ""
	fs.StringVar(&configPath, "config", "", "Optional config file path")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	cfg, _, err := config.LoadEffective(configPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config get: %v\n", err)
		return 1
	}
	cfg = cfg.Masked()
	remaining := fs.Args()
	if len(remaining) == 1 {
		value, err := lookupConfigValue(cfg, remaining[0])
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "config get: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintln(stdout, value)
		return 0
	}

	encoded, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config get: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, string(encoded))
	return 0
}

func runConfigSet(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("config set", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		printConfigSetUsage(stderr, fs)
	}
	configPath := ""
	fs.StringVar(&configPath, "config", "", "Optional config file path")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	remaining := fs.Args()
	if len(remaining) != 2 {
		_, _ = fmt.Fprintln(stderr, "config set requires KEY VALUE")
		return 2
	}

	cfg, resolvedPath, err := config.Load(configPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config set: %v\n", err)
		return 1
	}
	if err := cfg.Set(remaining[0], remaining[1]); err != nil {
		_, _ = fmt.Fprintf(stderr, "config set: %v\n", err)
		return 1
	}
	resolvedPath, err = config.Save(resolvedPath, cfg)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config set: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, resolvedPath)
	return 0
}

func lookupConfigValue(cfg config.Config, key string) (string, error) {
	normalizedKey := strings.ToLower(key)
	if strings.HasPrefix(normalizedKey, "provider_options.") {
		parts := strings.SplitN(normalizedKey, ".", 3)
		if len(parts) != 3 || strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[2]) == "" {
			return "", fmt.Errorf("provider option key must be provider_options.<provider>.<option>")
		}
		return cfg.ProviderOptions[parts[1]][parts[2]], nil
	}

	switch normalizedKey {
	case "openai_api_key":
		return cfg.OpenAIAPIKey, nil
	case "gemini_api_key":
		return cfg.GeminiAPIKey, nil
	case "timeout":
		return cfg.Timeout.String(), nil
	case "default_provider":
		return cfg.DefaultProvider, nil
	case "default_renderer":
		return cfg.DefaultRenderer, nil
	case "default_model":
		return cfg.DefaultModel, nil
	case "default_output_dir":
		return cfg.DefaultOutputDir, nil
	case "default_font_family":
		return cfg.DefaultFontFamily, nil
	case "default_font_size":
		return fmt.Sprintf("%g", cfg.DefaultFontSize), nil
	case "output_template":
		return cfg.OutputTemplate, nil
	case "overlay_color":
		return cfg.OverlayColor, nil
	case "overlay_opacity":
		return fmt.Sprintf("%g", cfg.OverlayOpacity), nil
	case "preserve_columns":
		return fmt.Sprintf("%t", cfg.PreserveColumns), nil
	case "source_language":
		return cfg.SourceLanguage, nil
	case "target_language":
		return cfg.TargetLanguage, nil
	case "image_description":
		return cfg.ImageDescription, nil
	default:
		return "", fmt.Errorf("unknown config key %q", key)
	}
}

func printHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Go Document Mirror Translator")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Translate one supported image into an editable A4 SVG or FODG overlay.")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Commands:")
	_, _ = fmt.Fprintln(w, "  render")
	_, _ = fmt.Fprintln(w, "  rerender")
	_, _ = fmt.Fprintln(w, "  gui")
	_, _ = fmt.Fprintln(w, "  config set")
	_, _ = fmt.Fprintln(w, "  config get")
	_, _ = fmt.Fprintln(w, "  config init")
	_, _ = fmt.Fprintln(w, "  providers list")
	_, _ = fmt.Fprintln(w, "  version")
	_, _ = fmt.Fprintln(w, "  help")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Key render flags:")
	_, _ = fmt.Fprintln(w, "  --input PATH")
	_, _ = fmt.Fprintln(w, "  --layout-json PATH")
	_, _ = fmt.Fprintln(w, "  --output-dir DIR")
	_, _ = fmt.Fprintln(w, "  --output-template TEMPLATE")
	_, _ = fmt.Fprintln(w, "  --provider NAME")
	_, _ = fmt.Fprintln(w, "  --renderer NAME")
	_, _ = fmt.Fprintln(w, "  --model NAME")
	_, _ = fmt.Fprintln(w, "  --source-lang LANG")
	_, _ = fmt.Fprintln(w, "  --target-lang LANG")
	_, _ = fmt.Fprintln(w, "  --font-family NAME")
	_, _ = fmt.Fprintln(w, "  --font-size NUMBER")
	_, _ = fmt.Fprintln(w, "  --opacity NUMBER")
	_, _ = fmt.Fprintln(w, "  --color VALUE")
	_, _ = fmt.Fprintln(w, "  --timeout DURATION")
	_, _ = fmt.Fprintln(w, "  --overwrite")
	_, _ = fmt.Fprintln(w, "  --save-layout-json")
	_, _ = fmt.Fprintln(w, "  --config PATH")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Examples:")
	_, _ = fmt.Fprintln(w, "  app render --input page.png --output-dir out --provider mock")
	_, _ = fmt.Fprintln(w, "  app render --input page.png --output-dir out --provider openai --model gpt-4.1-mini --save-layout-json")
	_, _ = fmt.Fprintln(w, "  app render --input page.png --output-dir out --renderer fodg")
	_, _ = fmt.Fprintln(w, "  app rerender --layout-json out/page.json --renderer fodg")
	_, _ = fmt.Fprintln(w, "  app config set default_provider openai")
	_, _ = fmt.Fprintln(w, "  app config set default_renderer fodg")
	_, _ = fmt.Fprintln(w, "  app config set provider_options.openai.image_detail high")
	_, _ = fmt.Fprintln(w, "  app providers list")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Config precedence: built-in defaults, config file, environment, CLI flags")
}

func printRenderUsage(w io.Writer, fs *flag.FlagSet) {
	_, _ = fmt.Fprintln(w, "Usage: app render [flags]")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Render one input image to an editable A4 SVG or FODG overlay.")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Flags:")
	fs.PrintDefaults()
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Examples:")
	_, _ = fmt.Fprintln(w, "  app render --input page.png --output-dir out --provider mock")
	_, _ = fmt.Fprintln(w, "  app render --input page.png --output-dir out --provider openai --model gpt-4.1-mini --save-layout-json")
	_, _ = fmt.Fprintln(w, "  app render --input page.png --output-dir out --overwrite")
	_, _ = fmt.Fprintln(w, "  app render --input page.png --output-dir out --renderer fodg")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Config precedence: built-in defaults, config file, environment, CLI flags")
}

func printRerenderUsage(w io.Writer, fs *flag.FlagSet) {
	_, _ = fmt.Fprintln(w, "Usage: app rerender [flags]")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Render a new SVG or FODG from an existing layout JSON without calling the AI provider again.")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Flags:")
	fs.PrintDefaults()
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Examples:")
	_, _ = fmt.Fprintln(w, "  app rerender --layout-json out/page.json")
	_, _ = fmt.Fprintln(w, "  app rerender --layout-json out/page.json --renderer fodg --output-template review_copy.fodg")
	_, _ = fmt.Fprintln(w, "  app rerender --layout-json out/page.json --overwrite")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Config precedence: built-in defaults, config file, environment, CLI flags")
}

func printGUIUsage(w io.Writer, fs *flag.FlagSet) {
	_, _ = fmt.Fprintln(w, "Usage: app gui [flags]")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Launch the Fyne GUI.")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Flags:")
	fs.PrintDefaults()
}

func printConfigUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: app config <init|get|set> [flags]")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Manage persisted local configuration values.")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Subcommands:")
	_, _ = fmt.Fprintln(w, "  init")
	_, _ = fmt.Fprintln(w, "  get")
	_, _ = fmt.Fprintln(w, "  set")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Examples:")
	_, _ = fmt.Fprintln(w, "  app config init")
	_, _ = fmt.Fprintln(w, "  app config get default_provider")
	_, _ = fmt.Fprintln(w, "  app config set provider_options.openai.image_detail high")
}

func printConfigInitUsage(w io.Writer, fs *flag.FlagSet) {
	_, _ = fmt.Fprintln(w, "Usage: app config init [flags]")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Create the config file with defaults if it does not already exist.")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Flags:")
	fs.PrintDefaults()
}

func printConfigGetUsage(w io.Writer, fs *flag.FlagSet) {
	_, _ = fmt.Fprintln(w, "Usage: app config get [flags] [KEY]")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Read the effective configuration or one specific key.")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Flags:")
	fs.PrintDefaults()
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Example:")
	_, _ = fmt.Fprintln(w, "  app config get provider_options.openai.image_detail")
}

func printConfigSetUsage(w io.Writer, fs *flag.FlagSet) {
	_, _ = fmt.Fprintln(w, "Usage: app config set [flags] KEY VALUE")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Persist one configuration value.")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Flags:")
	fs.PrintDefaults()
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Example:")
	_, _ = fmt.Fprintln(w, "  app config set provider_options.openai.image_detail high")
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "--help", "-h", "help":
			return true
		}
	}
	return false
}
