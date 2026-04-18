# Go Document Mirror Translator

The app takes a photo or scan of a document page, sends it to an AI provider, and generates an editable translated overlay (mirrors it) on top of the original image.

The result is an **A4 SVG** for **Inkscape** or an **A4 FODG** for **LibreOffice Draw**. The background stays the original page image, while the translated text is written as editable text objects so you can fine-tune the result manually afterward.

## What This App Does

Use it when you have:
- a handwritten or mixed handwritten/printed page;
- text in one language that you want translated into another;
- a need to keep the original page visually recognizable;
- a need to keep the translated text editable after generation.

The app:
- accepts one image file: `.jpg`, `.jpeg`, `.png`, or `.webp`;
- analyzes the page with OpenAI or Gemini;
- extracts an approximate text layout;
- translates the text;
- renders a new editable overlay document;
- can save a reusable layout JSON so you can change formatting later without sending the image to AI again.

## Typical Workflow

1. Select a source image.
2. Choose provider, model, source language, and target language.
3. Click `Analyze`.
4. Review the generated SVG or FODG in Inkscape or LibreOffice Draw.
5. If needed, change fonts, colors, outline, background, or shadow and click `Re-render` from the saved layout JSON without re-running AI analysis.

## Main Features

- Shared **GUI + CLI** application core
- **Fyne GUI** for Linux, Windows, macOS, and Android
- **CLI** for desktop use and scripting
- **OpenAI** and **Gemini** provider support
- **SVG** and **FODG** output
- **Layout JSON export/import** for rerendering without another provider call
- Local config for API keys, provider defaults, and render defaults
- Native desktop file/folder pickers, with Fyne fallback on Android
- Readability controls for:
  - text color;
  - font weight;
  - outline;
  - text background;
  - text shadow

## Output Formats

### SVG

Best when you want to edit the result in **Inkscape**.

SVG has the best formatting fidelity in this project right now:
- editable text;
- text color and opacity;
- outline color and width;
- text background;
- shadow controls.

### FODG

Best when you want to edit the result in **LibreOffice Draw**.

FODG is supported, but LibreOffice has some import limitations compared with SVG:
- text remains editable;
- font family, size, weight, color, background, and shadow work;
- text opacity is currently ignored by LibreOffice on import;
- outline behaves like Draw contour mode, not a true adjustable SVG-style stroke.

## GUI

The GUI is designed around a reusable adaptive shell:
- top bar with `Show Menu` / `Hide Menu`;
- left navigation menu;
- one persistent main content area;
- bottom action bar;
- bottom status/details area.

On smaller or vertical screens, the left menu overlays the content instead of shrinking it. On compact layouts, selecting a section auto-hides the menu again.

The main GUI sections are:
- `Main`: input image, layout JSON, output folder, filename template, output format, source/target language, preserve columns;
- `AI Settings`: provider, model, timeout, API keys, provider-specific options;
- `Design Settings`: font and readability controls.

## CLI

The CLI supports:
- analyzing one image;
- rerendering from one saved layout JSON;
- listing providers;
- reading and writing config;
- launching the GUI;
- version and help output.

Examples:

```bash
go run ./cmd/app render \
  --input samples/input/sample-page.png \
  --output-dir /tmp/out \
  --provider mock
```

```bash
go run ./cmd/app rerender \
  --layout-json samples/output/sample-page_mock.json \
  --output-dir /tmp/out \
  --renderer fodg
```

```bash
go run ./cmd/app providers list
```

If you prefer repository helpers:

```bash
make build
make run-help
```

## Why Layout JSON Matters

AI analysis is the expensive part.

After a successful `Analyze`, the app can save a normalized layout JSON file that contains the app's internal page model. That lets you:
- rerender with different fonts, colors, outline, shadow, or background;
- switch between SVG and FODG;
- avoid paying for and waiting for the same provider call again.

## Requirements

- Go `1.25+`
- For Linux desktop builds: CGO and the native libraries Fyne expects
- For Android packaging: `fyne` CLI, Android SDK/NDK, and Java toolchain
- `golangci-lint` if you want `make lint` to run locally

## Build And Validate

Common repository commands:

```bash
make fmt
make fmt-check
make test
make check
make build
make android
make run-help
```

In constrained environments you may need:

```bash
GOCACHE=/tmp/go-build make check
```

## Configuration

Configuration precedence is:

1. built-in defaults
2. saved config file
3. environment variables
4. CLI flags

The app stores config in the OS user config directory under the app-specific folder unless you override it with `--config` or `GODOCMIRRORTRANSLATOR_CONFIG`.

Useful environment variables:
- `GODOCMIRRORTRANSLATOR_OPENAI_API_KEY`
- `GODOCMIRRORTRANSLATOR_GEMINI_API_KEY`
- `GODOCMIRRORTRANSLATOR_DEFAULT_PROVIDER`
- `GODOCMIRRORTRANSLATOR_OUTPUT_TEMPLATE`

Example config:
- [examples/config.example.json](examples/config.example.json)

## Sample Files

The repository includes sample assets for quick manual testing:
- [samples/input/sample-page.png](samples/input/sample-page.png)
- [samples/output/sample-page_mock.svg](samples/output/sample-page_mock.svg)
- [samples/output/sample-page_mock.json](samples/output/sample-page_mock.json)

## Repository Structure

- `cmd/app`: application entrypoint and Android icon
- `internal/app`: render orchestration, rerendering, file output
- `internal/config`: config loading, masking, persistence, defaults
- `internal/provider`: provider contract plus adapters
- `internal/renderer/svg`: SVG renderer
- `internal/renderer/fodg`: FODG renderer
- `internal/gui`: Fyne GUI
- `internal/cli`: CLI wiring

## Current Status

The project currently supports the full core workflow:
- analyze image -> generate editable SVG or FODG;
- save layout JSON;
- rerender from JSON with different formatting;
- run through GUI or CLI;
- package the GUI for Android.
