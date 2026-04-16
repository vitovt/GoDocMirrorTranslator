package cli

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"godocmirrortranslator/internal/config"
)

func TestRunProvidersList(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"providers", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, name := range []string{"gemini", "mock", "openai"} {
		if !strings.Contains(stdout.String(), name) {
			t.Fatalf("providers list missing %q in %q", name, stdout.String())
		}
	}
}

func TestRunHelpIncludesRenderFlagsAndProviderOptionExample(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(help) exit code = %d, stderr = %s", code, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"--output-template TEMPLATE",
		"--save-layout-json",
		"provider_options.openai.image_detail",
		"Config precedence: built-in defaults, config file, environment, CLI flags",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("help output missing %q in %q", want, output)
		}
	}
}

func TestSubcommandHelpReturnsSuccess(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "render", args: []string{"render", "--help"}, want: "Usage: app render [flags]"},
		{name: "gui", args: []string{"gui", "--help"}, want: "Usage: app gui [flags]"},
		{name: "config", args: []string{"config", "--help"}, want: "Usage: app config <init|get|set> [flags]"},
		{name: "config init", args: []string{"config", "init", "--help"}, want: "Usage: app config init [flags]"},
		{name: "config get", args: []string{"config", "get", "--help"}, want: "Usage: app config get [flags] [KEY]"},
		{name: "config set", args: []string{"config", "set", "--help"}, want: "Usage: app config set [flags] KEY VALUE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(context.Background(), tt.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("Run(%v) exit code = %d, stderr = %s", tt.args, code, stderr.String())
			}
			output := stdout.String() + stderr.String()
			if !strings.Contains(output, tt.want) {
				t.Fatalf("help output for %v missing %q in %q", tt.args, tt.want, output)
			}
		})
	}
}

func TestRenderHelpIncludesExamplesAndPrecedence(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"render", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(render --help) exit code = %d, stderr = %s", code, stderr.String())
	}
	output := stdout.String() + stderr.String()
	for _, want := range []string{
		"Examples:",
		"--save-layout-json",
		"Config precedence: built-in defaults, config file, environment, CLI flags",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("render help missing %q in %q", want, output)
		}
	}
}

func TestRunRender(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{
		"render",
		"--input", inputPath,
		"--output-dir", tempDir,
		"--provider", "mock",
		"--save-layout-json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() exit code = %d, stderr = %s", code, stderr.String())
	}
	outputPath := strings.TrimSpace(strings.Split(stdout.String(), "\n")[0])
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected svg output at %q: %v", outputPath, err)
	}
	jsonPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".json"
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("expected layout json at %q: %v", jsonPath, err)
	}
}

func TestRunRenderUsesConfigDefaults(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	cfgPath := filepath.Join(tempDir, "config.json")
	cfg := config.Default()
	cfg.DefaultOutputDir = filepath.Join(tempDir, "configured-out")
	cfg.OutputTemplate = "configured_{provider}.svg"
	if _, err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{
		"render",
		"--config", cfgPath,
		"--input", inputPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() exit code = %d, stderr = %s", code, stderr.String())
	}
	outputPath := strings.TrimSpace(stdout.String())
	if !strings.HasPrefix(outputPath, cfg.DefaultOutputDir) {
		t.Fatalf("output path = %q, want prefix %q", outputPath, cfg.DefaultOutputDir)
	}
}

func TestRunRenderFailsWhenProviderCredentialsAreMissing(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	cfgPath := filepath.Join(tempDir, "config.json")
	cfg := config.Default()
	cfg.DefaultProvider = "openai"
	if _, err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{
		"render",
		"--config", cfgPath,
		"--input", inputPath,
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "validate provider config") {
		t.Fatalf("stderr = %q, want provider config validation error", stderr.String())
	}
}

func TestRunRenderAppliesEnvOverridesBeforeConfigDefaults(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "page.png")
	writeTestPNG(t, inputPath, 640, 960)

	cfgPath := filepath.Join(tempDir, "config.json")
	cfg := config.Default()
	cfg.DefaultProvider = "openai"
	cfg.OutputTemplate = "from_config.svg"
	if _, err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	t.Setenv(config.EnvPrefix+"DEFAULT_PROVIDER", "mock")
	t.Setenv(config.EnvPrefix+"OUTPUT_TEMPLATE", "from_env_{provider}.svg")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{
		"render",
		"--config", cfgPath,
		"--input", inputPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() exit code = %d, stderr = %s", code, stderr.String())
	}
	outputPath := strings.TrimSpace(stdout.String())
	if !strings.HasSuffix(outputPath, "from_env_mock.svg") {
		t.Fatalf("output path = %q, want env-driven mock output name", outputPath)
	}
}

func TestRunConfigCommands(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run(context.Background(), []string{"config", "init", "--config", cfgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("config init exit code = %d, stderr = %s", code, stderr.String())
	}
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("expected config file at %q: %v", cfgPath, err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run(context.Background(), []string{"config", "set", "--config", cfgPath, "default_provider", "openai"}, &stdout, &stderr); code != 0 {
		t.Fatalf("config set exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run(context.Background(), []string{"config", "get", "--config", cfgPath, "default_provider"}, &stdout, &stderr); code != 0 {
		t.Fatalf("config get exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "openai" {
		t.Fatalf("config get default_provider = %q, want openai", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run(context.Background(), []string{"config", "set", "--config", cfgPath, "openai_api_key", "sk-secret-value"}, &stdout, &stderr); code != 0 {
		t.Fatalf("config set api key exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run(context.Background(), []string{"config", "get", "--config", cfgPath, "openai_api_key"}, &stdout, &stderr); code != 0 {
		t.Fatalf("config get api key exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "sk-secret-value") {
		t.Fatalf("config get leaked raw API key: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run(context.Background(), []string{"config", "set", "--config", cfgPath, "provider_options.openai.image_detail", "high"}, &stdout, &stderr); code != 0 {
		t.Fatalf("config set provider option exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run(context.Background(), []string{"config", "get", "--config", cfgPath, "provider_options.openai.image_detail"}, &stdout, &stderr); code != 0 {
		t.Fatalf("config get provider option exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "high" {
		t.Fatalf("config get provider option = %q, want high", stdout.String())
	}
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
