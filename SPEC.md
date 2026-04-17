# Technical Specification — Go Document Mirror Translator

## 1. Project Summary
Build a cross-platform GUI and CLI application in **Go** that:
- accepts a single input image (`.jpg`, `.jpeg`, `.png`, `.webp`);
- sends the image to an AI provider;
- receives structured page layout + translation data;
- generates an **A4 SVG or FODG** file with:
  - the original image as background;
  - translated text overlaid as editable text elements;
- supports manual post-editing in **Inkscape** or **LibreOffice Draw**;
- provides a **Fyne GUI** on desktop and Android, and a **CLI** on desktop platforms, using the same core logic.

## 2. Technology Stack
- Language: **Go**
- GUI: **Fyne**
- AI providers:
  - **OpenAI**
  - **Google Gemini**
- Output formats v1: **SVG**, **FODG**
- Testing: Go unit, integration, golden, and GUI tests
- Linting: `golangci-lint`

## 3. Scope

### In Scope (v1)
- Single-image processing
- Input formats: JPG, JPEG, PNG, WEBP (20 MiB or smaller)
- Output formats: SVG, FODG, and optional layout JSON export
- OpenAI provider
- Gemini provider
- Fyne GUI for Linux, Windows, macOS, and Android
- CLI interface
- Local configuration storage
- Native-first file/folder pickers
- Automated tests
- Modular architecture for future extensions

### Out of Scope (v1)
- PDF output
- ODG output
- Batch processing
- Layout JSON import
- In-app visual layout editor
- OCR-only mode without LLM
- Cloud sync
- OS keyring integration

## 4. Functional Requirements

### 4.1 Input
The application must:
- accept exactly one input image per processing run;
- support `.jpg`, `.jpeg`, `.png`, `.webp`;
- validate file existence and readability;
- detect image dimensions;
- reject files larger than 20 MiB with a clear error;
- reject unsupported or invalid files with clear errors.

### 4.2 Processing
The application must:
- send the image to the selected AI provider;
- request structured output in a unified internal schema;
- detect text blocks approximately matching the original layout;
- translate text from the configured source language to the configured target language;
- by default source language is **Ukrainian** and target language is **German**;
- preserve rough block/column structure;
- return confidence values where available.

### 4.3 Output
The application must generate an **A4 SVG or FODG** file that:
- embeds the source image as the page background;
- overlays translated text as editable text elements;
- preserves editable text, not paths;
- uses UTF-8 encoding;
- opens correctly in the target editor for the selected output format;
- is valid XML for the selected output format;
- optionally writes a layout JSON file, when requested, that serializes the normalized internal page model used for rendering.

### 4.4 GUI
The GUI must support:
- using an adaptive Fyne shell with a top menu toggle, a left menu, one persistent main content pane, and bottom action/status/details regions;
- selecting an input image;
- selecting an output directory;
- editing an output filename template;
- selecting an output format;
- selecting provider, model, and provider-specific advanced options;
- network/AI timeout;
- entering and saving API keys locally;
- configuring rendering options;
- starting processing;
- showing progress and status;
- opening the output directory after success.

### 4.5 CLI
The CLI must support:
- processing one image;
- launching GUI;
- listing providers;
- showing version;
- reading config;
- writing config;
- clear `--help` output with examples.

## 5. Non-Functional Requirements

### 5.1 Performance
- GUI must remain responsive during processing.
- Rendering after provider response must complete locally without noticeable delay for a single page.

### 5.2 Reliability
- Handle invalid input, missing config, provider errors, timeouts, parsing errors, and file system errors gracefully.
- Do not write corrupt or partial output files on failure.
- Support configurable request timeout.

### 5.3 Maintainability
- Provider-specific logic must be isolated.
- Renderer-specific logic must be isolated.
- GUI must not contain business logic.
- CLI and GUI must share the same application core.
- The internal page model must be provider-agnostic and renderer-agnostic.

### 5.4 Portability
- Required v1 GUI targets: Linux, Windows, macOS, and Android.
- Required v1 CLI targets: Linux, Windows, and macOS.
- Android support in v1 is GUI-only.

## 6. Architecture

### 6.1 Architectural Style
Use a modular architecture with clear separation between:
- domain model
- application services
- providers
- renderers
- config
- CLI
- GUI
- infrastructure helpers

### 6.2 Internal Page Model
Define a neutral internal model for page processing.

#### DocumentPage
- source image path
- source image width/height
- page format
- orientation
- block list
- metadata

#### TextBlock
- id
- source text
- translated text
- x
- y
- width
- height
- rotation
- font size
- font family
- align
- color
- opacity
- line height
- confidence
- notes

#### Field Semantics
- `source image width/height` are the original input image dimensions in pixels.
- `page format` is `A4` in v1.
- `orientation` is `portrait` or `landscape`.
- `x`, `y`, `width`, and `height` are floating-point values in original source-image pixel space with the origin at the top-left corner.
- `font size` is expressed in original source-image pixel space and is scaled proportionally by the renderer to the A4 canvas.
- `rotation` is clockwise degrees and defaults to `0`.
- `align` is one of `start`, `center`, or `end`.
- `color` is an optional CSS/SVG color value and falls back to renderer defaults when absent.
- `opacity` is in `[0,1]` and defaults to `1`.
- `line height` is a positive multiplier and defaults to `1.2`.
- `confidence` is optional and, when present, is in `[0,1]`.
- `source text` is required for every block, and unreadable text must use the literal value `[unreadable]`.

### 6.3 Provider Interface
All providers must implement a shared interface.

```go
type Provider interface {
    Name() string
    AnalyzePage(ctx context.Context, req AnalyzeRequest) (*DocumentPage, error)
    ValidateConfig(cfg ProviderConfig) error
    SupportedModels() []string
}
```

### 6.3.1 AnalyzeRequest and ProviderConfig
`AnalyzeRequest` must include a readable source image reference or bytes, original source image width and height, source language, target language, selected model, and request timeout.

`ProviderConfig` must include API key or equivalent credential reference, default model if any, and provider-specific advanced options loaded from local config.

### 6.4 Renderer Interface

All renderers must implement a shared interface.

```go
type Renderer interface {
    Name() string
    Render(ctx context.Context, page *DocumentPage, opts RenderOptions) ([]byte, error)
    FileExtension() string
}
```

### 6.4.1 RenderOptions
`RenderOptions` must include font family, default font size, text color, opacity, preserve-columns behavior, and any renderer-specific defaults explicitly allowed in v1.

## 7. Repository Structure

```text
/cmd
  /app

/internal
  /domain
  /app
  /provider
    /openai
    /gemini
  /renderer
    /svg
    /fodg
  /config
  /prompts
  /gui
  /cli
  /infra
  /testdata

/tests
  /integration
  /e2e
```

## 8. Provider Requirements

### 8.1 Supported Providers

* `openai`
* `gemini`

### 8.2 Structured Output

Both providers must map their response into the same internal schema.

Provider adapters may create transient resized or recompressed images to satisfy provider API limits, but they must normalize all returned coordinates and size-related values back into the original source-image pixel space before constructing the internal page model.

### 8.3 Prompt Contract

The provider request must instruct the model to:

* read handwritten or mixed handwritten/printed text in the configured source language;
* translate it into the configured target language;
* group content into approximate text blocks;
* preserve rough visual layout;
* return only structured output;
* avoid hallucinating missing text;
* mark uncertain or unreadable text explicitly.

### 8.4 Unreadable Content

Unreadable text must:

* remain represented as a block;
* be marked as unreadable in source text;
* not be silently omitted;
* have reduced confidence.

## 9. Renderer Requirements

### 9.1 Page Format

* A4 page size
* correct page dimensions
* proportional background image placement
* visible overlay text on top of the image

### 9.2 SVG Renderer Requirements

Each text block must be rendered as an editable SVG text element with:

* coordinates
* font family
* font size
* fill color
* opacity
* optional rotation
* text alignment

Generated SVG output must open correctly in Inkscape and keep translated text editable as text.

### 9.3 FODG Renderer Requirements

Each text block must be rendered into a flat LibreOffice Draw (`.fodg`) document with:

* the source image embedded in the document;
* translated text preserved as editable text, not curves;
* A4 page sizing;
* block coordinates scaled from the shared internal page model;
* optional rotation;
* text alignment;
* font family, font size, color, and opacity where supported by the format.

Generated FODG output must open correctly in LibreOffice Draw and keep translated text editable as text.

### 9.4 File Naming

Support output filename templates with variables:

* `{input_basename}`
* `{provider}`
* `{model}`
* `{date}`
* `{time}`
* `{timestamp}`

Default template:

```text
{input_basename}_{provider}_{timestamp}.svg
```

Variable formats:

* `{date}` => `YYYY-MM-DD`
* `{time}` => `HH-MM-SS`
* `{timestamp}` => `YYYYMMDD-HHMMSS` in local time

Collision handling: if the target rendered-output path already exists, append `-1`, `-2`, and so on before the extension instead of overwriting. When layout JSON export is enabled, write the JSON file next to the selected renderer output using the same basename and a `.json` extension.

## 10. GUI Requirements

### 10.1 Main Screen

The main window must contain:

* input file field
* input browse button
* output directory field
* output directory browse button
* output filename template field
* output format selector
* provider selector
* model selector
* provider advanced settings
* source language selector
* target language selector
* rendering settings
* API key settings
* process button
* progress indicator
* status line
* log/details area

### 10.2 Behavior

* `Process` must be disabled until required fields are valid.
* Validation errors must be visible in the UI.
* Long-running tasks must run asynchronously.
* Previous settings must be restored on next launch.
* Provider-specific advanced options are edited in the GUI and persisted to local config.

### 10.3 File Pickers

* Use native file/folder pickers as the primary implementation.

### 10.4 Adaptive Design

* GUI must be flexible and look confident on different screen sizes and orientations, including Android.
* GUI must follow the reusable Fyne adaptive shell requirements defined in Appendix A.

## 11. CLI Requirements

### 11.1 Commands

The CLI must provide:

```text
render
gui
config set
config get
config init
providers list
version
help
```

### 11.2 Render Command Flags

The `render` command must support:

```text
--input
--output-dir
--output-template
--provider
--renderer
--model
--source-lang
--target-lang
--font-family
--font-size
--opacity
--color
--config
--save-layout-json
--timeout
--verbose
```

`--save-layout-json` writes the normalized internal page model next to the selected renderer output using the same basename and a `.json` extension.

### 11.3 Help Output

`--help` must include:

* short product description;
* command list;
* parameter descriptions;
* working examples;
* config precedence rules.

## 12. Configuration Requirements

### 12.1 Stored Values

The application must store:

* OpenAI API key
* Gemini API key
* network/AI timeout
* default provider
* default renderer
* default model
* default output directory
* default font family
* default font size
* output filename template
* overlay color
* overlay opacity
* preserve-columns setting
* provider-specific advanced options supported by the GUI
* GUI preferences

### 12.2 Rules

* GUI and CLI must use the same effective configuration model.
* CLI flags override config file values.
* Environment variables override config defaults and are overridden by CLI flags.
* API keys must never appear in logs or error messages.
* API key fields in GUI must be masked.
* The CLI uses provider-specific advanced options from effective config and does not expose per-run override flags for them in v1.

### 12.3 Precedence

Configuration precedence:

1. built-in defaults
2. config file / local preferences
3. environment variables
4. CLI flags

### 12.4 Storage Location and Permissions

Store config in the per-user config directory for the host OS under a stable app-specific subdirectory.

A missing config file is not an error.

If the config file contains API keys, create it with user-only permissions on Unix-like systems and the closest current-user-only equivalent on Windows.

`config get` must mask API keys by default.

## 13. Error Handling Requirements

The application must provide clear user-facing errors for:

* invalid input file
* unsupported image format
* missing API key
* invalid config
* provider authentication failure
* network timeout
* provider response parse failure
* renderer failure
* file write failure

Rules:

* do not produce empty or corrupted output files;
* clean up partial files on failure;
* return non-zero exit codes in CLI;
* show visible errors in GUI.

## 14. Testing Requirements

### 14.1 Unit Tests

Cover:

* config loading/saving
* config precedence
* input validation
* output filename templating
* provider response mapping
* SVG renderer
* coordinate normalization
* style formatting
* default values

### 14.2 Integration Tests

Cover:

* mock provider → internal page model → SVG output
* file writing
* failure handling

### 14.3 Golden Tests

Cover:

* stable SVG output from fixed input page model

### 14.4 GUI Tests

Cover:

* form validation
* provider switching
* settings save/load
* process button enabled/disabled state
* success/failure status updates

### 14.5 Coverage Targets

* core packages: minimum 70%
* config and renderer: minimum 85%
* provider adapters: minimum 70%

## 15. Code Quality Requirements

* Use `go test ./...`
* Use `golangci-lint`
* No provider logic in GUI
* No renderer logic in GUI
* No GUI dependencies in domain layer
* No hardcoded file paths
* No hardcoded secrets
* Keep interfaces minimal and purposeful

## 16. Delivery Requirements

### 16.1 v1 Deliverables

* Go module
* CLI application
* Fyne GUI
* Android APK
* OpenAI provider
* Gemini provider
* SVG renderer
* FODG renderer
* optional layout JSON export
* local config support
* automated tests
* README
* example config
* sample input/output assets

### 16.2 v2 Preparation

The v1 architecture must allow adding:

* additional providers
* batch mode
* layout JSON import

## 17. Acceptance Criteria

### Functional

* User can select an image in GUI and generate SVG or FODG successfully.
* User can select an image in the Android GUI and generate SVG or FODG successfully.
* User can process the same image from CLI.
* When requested, the CLI writes a JSON export of the normalized layout next to the selected renderer output.
* Generated SVG opens in Inkscape.
* Generated FODG opens in LibreOffice Draw.
* Overlay text remains editable as text.
* Settings persist locally.
* Provider selection works for both OpenAI and Gemini.

### Technical

* Provider layer is isolated from renderer layer.
* GUI and CLI both use the same core processing flow.
* SVG and FODG renderers are independent from provider implementation.
* Android APK builds successfully.
* Test suite passes.
* Lint passes.

## 18. Implementation Order

1. Domain model
2. Config system
3. Renderer interface
4. SVG renderer
5. Mock provider
6. CLI render command
7. OpenAI provider
8. Gemini provider
9. Integration and golden tests
10. Fyne GUI
11. Native picker adapter
12. GUI tests
13. FODG renderer
14. GUI output format selection
15. Packaging and release preparation

## Appendix A. Reusable Fyne Adaptive Shell Template

This appendix is intentionally written as a reusable design section for future Fyne applications. It may be copied into another specification and then adapted by replacing the application-specific menu entries, content views, and action labels while keeping the shell behavior intact.

### A.1 Purpose

Use this shell when a Fyne application needs:

* one persistent main workspace;
* a left navigation or settings menu;
* bottom actions that stay visible;
* a visible status and details area;
* adaptive behavior for desktop, tablet, and mobile layouts.

### A.2 Layout Contract

* The shell is divided into a top bar, a center workspace, a bottom action row, and a bottom status/details region.
* The center workspace contains exactly two functional panes: a left menu pane and one persistent main content pane.
* The main content pane is the only place where section content changes.
* Do not add a permanent third side pane unless the product specification explicitly requires it.
* The status/details region belongs below the action row, not beside the main content pane.

### A.3 Top Bar Requirements

* The top bar contains a primary menu toggle and any truly global actions.
* The menu toggle label must reflect the current state, for example `Show Menu` and `Hide Menu`.
* On application startup the menu is open by default unless the product specification explicitly overrides this.
* The top bar must remain visible in all supported sizes and orientations.

### A.4 Menu Pane Behavior

* The menu pane contains section navigation only.
* Selecting a menu item changes the content displayed in the persistent main content pane; it does not create an additional middle or right-side content column.
* One section is selected by default on startup.
* Menu items should be vertically stacked and visually highlight the active section.
* The menu pane should keep a stable minimum width in docked mode.
* If menu content exceeds available height, the menu area must remain usable through vertical scrolling.
* On compact layouts, selecting a menu item should immediately close the overlay menu unless the product specification explicitly requires it to stay open.

### A.5 Main Content Pane Behavior

* The shell uses one persistent content container.
* Only one major section is visible at a time.
* Each major section should support vertical scrolling.
* Avoid horizontal scrolling where practical; forms and settings should reflow into a narrow-friendly single-column arrangement when needed.
* Application-specific inputs, previews, and settings panels are swapped inside this pane instead of being rendered as separate permanent sidebars.
* The main content pane must remain usable whether the menu is open or closed.

### A.6 Bottom Actions And Status

* Primary actions such as `Save`, `Process`, `Open`, `Run`, `Export`, or equivalents stay in a bottom action row that remains visible while the user navigates between sections.
* Progress indicators may appear in the action row if they do not hide the main actions.
* A short status line is displayed below the action row.
* A details or log area is displayed below the status line.
* On compact layouts, the details area should be collapsed by default behind a `Show Details` / `Hide Details` control or equivalent.
* On wide desktop layouts, the details area may be expanded by default.
* Opening or closing the menu must not move the bottom status/details region into a side column.

### A.7 Adaptive Behavior

* The shell must support both docked and overlay menu behavior.
* Compact mode is triggered on mobile devices and on narrow desktop or tablet widths defined by a code-level threshold.
* In compact mode, the menu overlays the main content instead of permanently reducing the content width.
* In wide mode, the menu may be docked if the product specification prefers it.
* Orientation changes and live window resizing must recompute the layout immediately.
* The same shared UI state model must drive both wide and compact modes; do not duplicate business logic per mode.
* Touch targets, spacing, and scrolling must remain usable in portrait mobile layouts.

### A.8 Fyne Implementation Guidance

* Prefer standard Fyne containers such as `container.NewBorder`, `container.NewStack`, `container.NewVBox`, `container.NewHBox`, and `container.NewVScroll` over custom renderers or custom layout widgets unless the standard containers cannot express the required behavior.
* Keep GUI code thin: widget callbacks gather input, update UI state, and call shared application services.
* Long-running work must run asynchronously in goroutines; return UI updates through `fyne.Do` or `fyne.DoAndWait`.
* Keep menu visibility, selected section, compact-mode state, and details visibility as explicit UI state.
* Persist only restorable preferences such as last-selected provider, default paths, or shell preferences; do not persist transient processing state.
* Use native-first file and folder pickers on desktop and a Fyne fallback on mobile or unsupported backends.
* Keep platform-specific behavior at the edge through small adapters or build-tagged files.
* Error dialogs may be used for blocking failures, but a visible status area must still reflect the current state.

### A.9 Reuse Checklist

* Replace menu labels with the target application's sections.
* Replace main-pane forms or views with the target application's content.
* Replace bottom action labels with the target application's primary operations.
* Keep the top/menu/center/bottom shell behavior unchanged unless the new application has a documented reason to diverge.

### A.10 Test Checklist

* Default section is selected on startup.
* Menu starts in the expected visibility state.
* Menu toggle shows and hides the menu correctly.
* Selecting a menu section changes the visible content pane.
* Compact mode closes the overlay menu after selection when configured to do so.
* Bottom action row remains visible while sections change.
* Details area collapses and expands as specified.
* Resize and orientation changes recompute compact versus wide layout.
* Long-running actions do not block the UI thread.
