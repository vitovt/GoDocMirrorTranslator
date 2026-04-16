package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"godocmirrortranslator/internal/app"
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
		_, _ = fmt.Fprintln(stderr, "gui command is not implemented yet")
		return 1
	case "config":
		_, _ = fmt.Fprintln(stderr, "config commands are not implemented yet")
		return 1
	case "help", "--help", "-h":
		printHelp(stdout)
		return 0
	}

	_, _ = fmt.Fprintf(stderr, "unknown command: %s\n", strings.Join(args, " "))
	printHelp(stderr)
	return 1
}

func runRender(ctx context.Context, application *app.Application, args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var req app.RenderRequest
	fs.StringVar(&req.InputPath, "input", "", "Path to the input image")
	fs.StringVar(&req.OutputDir, "output-dir", "", "Directory for generated files")
	fs.StringVar(&req.OutputTemplate, "output-template", app.DefaultOutputTemplate, "Filename template for SVG output")
	fs.StringVar(&req.ProviderName, "provider", "mock", "Provider to use")
	fs.StringVar(&req.Model, "model", "", "Model to use")
	fs.StringVar(&req.SourceLanguage, "source-lang", "Ukrainian", "Source language")
	fs.StringVar(&req.TargetLanguage, "target-lang", "German", "Target language")
	fs.StringVar(&req.RenderOptions.FontFamily, "font-family", "Noto Sans", "Fallback font family")
	fs.Float64Var(&req.RenderOptions.DefaultFontSize, "font-size", 18, "Fallback font size")
	fs.Float64Var(&req.RenderOptions.Opacity, "opacity", 1, "Fallback text opacity")
	fs.StringVar(&req.RenderOptions.TextColor, "color", "#111111", "Fallback text color")
	fs.DurationVar(&req.Timeout, "timeout", 30*time.Second, "Provider request timeout")
	fs.BoolVar(&req.SaveLayoutJSON, "save-layout-json", false, "Write layout JSON next to the SVG output")
	verbose := fs.Bool("verbose", false, "Print extra result information")
	_ = fs.String("config", "", "Reserved for future config file support")

	if err := fs.Parse(args); err != nil {
		return 2
	}

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

func printHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Go Document Mirror Translator")
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
	_, _ = fmt.Fprintln(w, "Examples:")
	_, _ = fmt.Fprintln(w, "  app render --input page.png --output-dir out --provider mock")
	_, _ = fmt.Fprintln(w, "  app providers list")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Config precedence: built-in defaults, config file, environment, CLI flags")
}
