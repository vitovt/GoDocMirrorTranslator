package cli

import (
	"context"
	"encoding/json"
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
	configPath := ""
	fs.StringVar(&configPath, "config", "", "Optional config file path")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := gui.Run(ctx, application, Version, configPath); err != nil {
		_, _ = fmt.Fprintf(stderr, "gui: %v\n", err)
		return 1
	}
	return 0
}

func runRender(ctx context.Context, application *app.Application, args []string, stdout io.Writer, stderr io.Writer) int {
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

	cfg, _, err := config.LoadEffective(configPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "render: load config: %v\n", err)
		return 1
	}

	req := app.RenderRequest{
		OutputDir:      cfg.DefaultOutputDir,
		OutputTemplate: cfg.OutputTemplate,
		ProviderName:   cfg.DefaultProvider,
		Model:          cfg.DefaultModel,
		SourceLanguage: cfg.SourceLanguage,
		TargetLanguage: cfg.TargetLanguage,
		Timeout:        cfg.Timeout,
		RenderOptions:  cfg.RenderOptions(),
	}

	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	fs.SetOutput(stderr)

	fs.StringVar(&req.InputPath, "input", req.InputPath, "Path to the input image")
	fs.StringVar(&req.OutputDir, "output-dir", req.OutputDir, "Directory for generated files")
	fs.StringVar(&req.OutputTemplate, "output-template", req.OutputTemplate, "Filename template for SVG output")
	fs.StringVar(&req.ProviderName, "provider", req.ProviderName, "Provider to use")
	fs.StringVar(&req.Model, "model", req.Model, "Model to use")
	fs.StringVar(&req.SourceLanguage, "source-lang", req.SourceLanguage, "Source language")
	fs.StringVar(&req.TargetLanguage, "target-lang", req.TargetLanguage, "Target language")
	fs.StringVar(&req.RenderOptions.FontFamily, "font-family", req.RenderOptions.FontFamily, "Fallback font family")
	fs.Float64Var(&req.RenderOptions.DefaultFontSize, "font-size", req.RenderOptions.DefaultFontSize, "Fallback font size")
	fs.Float64Var(&req.RenderOptions.Opacity, "opacity", req.RenderOptions.Opacity, "Fallback text opacity")
	fs.StringVar(&req.RenderOptions.TextColor, "color", req.RenderOptions.TextColor, "Fallback text color")
	fs.DurationVar(&req.Timeout, "timeout", req.Timeout, "Provider request timeout")
	fs.BoolVar(&req.SaveLayoutJSON, "save-layout-json", false, "Write layout JSON next to the SVG output")
	verbose := fs.Bool("verbose", false, "Print extra result information")
	fs.StringVar(&configPath, "config", configPath, "Optional config file path")

	if err := fs.Parse(args); err != nil {
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

func runConfig(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "config command requires a subcommand: init, get, set")
		return 2
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
	configPath := ""
	fs.StringVar(&configPath, "config", "", "Optional config file path")
	if err := fs.Parse(args); err != nil {
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
	configPath := ""
	fs.StringVar(&configPath, "config", "", "Optional config file path")
	if err := fs.Parse(args); err != nil {
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
	configPath := ""
	fs.StringVar(&configPath, "config", "", "Optional config file path")
	if err := fs.Parse(args); err != nil {
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
	default:
		return "", fmt.Errorf("unknown config key %q", key)
	}
}

func printHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Go Document Mirror Translator")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Translate one supported image into an editable A4 SVG overlay.")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Commands:")
	_, _ = fmt.Fprintln(w, "  render")
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
	_, _ = fmt.Fprintln(w, "  --output-dir DIR")
	_, _ = fmt.Fprintln(w, "  --output-template TEMPLATE")
	_, _ = fmt.Fprintln(w, "  --provider NAME")
	_, _ = fmt.Fprintln(w, "  --model NAME")
	_, _ = fmt.Fprintln(w, "  --source-lang LANG")
	_, _ = fmt.Fprintln(w, "  --target-lang LANG")
	_, _ = fmt.Fprintln(w, "  --font-family NAME")
	_, _ = fmt.Fprintln(w, "  --font-size NUMBER")
	_, _ = fmt.Fprintln(w, "  --opacity NUMBER")
	_, _ = fmt.Fprintln(w, "  --color VALUE")
	_, _ = fmt.Fprintln(w, "  --timeout DURATION")
	_, _ = fmt.Fprintln(w, "  --save-layout-json")
	_, _ = fmt.Fprintln(w, "  --config PATH")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Examples:")
	_, _ = fmt.Fprintln(w, "  app render --input page.png --output-dir out --provider mock")
	_, _ = fmt.Fprintln(w, "  app render --input page.png --output-dir out --provider openai --model gpt-4.1-mini --save-layout-json")
	_, _ = fmt.Fprintln(w, "  app config set default_provider openai")
	_, _ = fmt.Fprintln(w, "  app config set provider_options.openai.image_detail high")
	_, _ = fmt.Fprintln(w, "  app providers list")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Config precedence: built-in defaults, config file, environment, CLI flags")
}
