# Handwritten Overlay Translator

Handwritten Overlay Translator is a Go application with a shared CLI and Fyne GUI that sends one document image to an AI provider and produces an editable A4 SVG or FODG overlay. SVG output is intended for Inkscape, and FODG output is intended for LibreOffice Draw.

## Features

- Single-image processing for `.jpg`, `.jpeg`, `.png`, and `.webp`
- Shared application core used by both CLI and GUI
- OpenAI and Gemini provider adapters plus a mock provider for local tests and sample generation
- Editable SVG and FODG output with optional normalized layout JSON export
- Local config storage for provider credentials, rendering defaults, and GUI preferences
- Desktop GUI plus Android packaging support through Fyne
- Native OS file and folder pickers on desktop, with Fyne fallback on Android

## Repository Layout

- `cmd/app` contains the main application entrypoint and Android app icon.
- `internal/app` contains render orchestration, validation, and file output logic.
- `internal/config` contains config loading, persistence, masking, and precedence rules.
- `internal/provider` contains the shared provider contract and provider adapters.
- `internal/renderer/svg` contains the SVG renderer and golden fixtures.
- `internal/renderer/fodg` contains the flat LibreOffice Draw renderer and golden fixtures.
- `internal/gui` contains the Fyne shell and GUI tests.
- `examples/config.example.json` shows a usable config file shape.
- `samples/input` and `samples/output` contain checked-in sample assets.

## Requirements

- Go 1.25+
- For desktop builds on Linux: CGO plus the native dependencies Fyne and the desktop native picker expect, such as `gcc`, `pkg-config`, `libgl1-mesa-dev`, `xorg-dev`, and `libgtk-3-dev`
- For Android packaging: `fyne` CLI, Android SDK/NDK, and the Java toolchain Fyne expects
- `golangci-lint` if you want `make lint` to run instead of skipping

## Common Commands

```bash
make fmt
make fmt-check
make test
GOCACHE=/tmp/go-build make check
make build
make run-help
```

To build an Android APK when the toolchain is installed:

```bash
GOCACHE=/tmp/go-build make android
```

## CLI Usage

Render one file with the mock provider:

```bash
go run ./cmd/app render \
  --input samples/input/sample-page.png \
  --output-dir /tmp/out \
  --provider mock
```

Render a LibreOffice Draw file with the mock provider:

```bash
go run ./cmd/app render \
  --input samples/input/sample-page.png \
  --output-dir /tmp/out \
  --provider mock \
  --renderer fodg
```

Render with a live provider and write layout JSON too:

```bash
go run ./cmd/app render \
  --input page.png \
  --output-dir out \
  --provider openai \
  --model gpt-4.1-mini \
  --save-layout-json
```

List providers:

```bash
go run ./cmd/app providers list
```

Initialize local config:

```bash
go run ./cmd/app config init
```

## Configuration

Configuration precedence is:

1. built-in defaults
2. config file / local preferences
3. environment variables
4. CLI flags

The config file lives under the host OS user config directory in the `handwritten-overlay-translator` subdirectory unless you override it with `--config` or `GODOCMIRRORTRANSLATOR_CONFIG`.

See [examples/config.example.json](examples/config.example.json) for a starting point.

Supported environment variable prefix:

- `GODOCMIRRORTRANSLATOR_`

Examples:

- `GODOCMIRRORTRANSLATOR_OPENAI_API_KEY`
- `GODOCMIRRORTRANSLATOR_GEMINI_API_KEY`
- `GODOCMIRRORTRANSLATOR_DEFAULT_PROVIDER`
- `GODOCMIRRORTRANSLATOR_OUTPUT_TEMPLATE`

## GUI

Running `go run ./cmd/app` with no arguments opens the Fyne GUI. The GUI supports:

- native-first input and output pickers
- desktop native OS picker integration with Fyne fallback on Android or unsupported desktop backends
- provider and model selection
- output format selection
- masked API key inputs
- provider-specific advanced options currently exposed by the repo
- persisted render defaults and GUI preferences
- async processing with visible status and details output

## Sample Assets

The repository includes a checked-in sample input plus outputs generated with the mock provider:

- [samples/input/sample-page.png](samples/input/sample-page.png)
- [samples/output/sample-page_mock.svg](samples/output/sample-page_mock.svg)
- [samples/output/sample-page_mock.json](samples/output/sample-page_mock.json)

These are useful for quick manual checks and packaging smoke tests.
