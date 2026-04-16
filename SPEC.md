# Technical Specification — Go Document Mirror Translator

## 1. Project Summary
Build a cross-platform desktop and CLI application in **Go** that:
- accepts a single input image (`.jpg`, `.jpeg`, `.png`);
- sends the image to an AI provider;
- receives structured page layout + translation data;
- generates an **A4 SVG** file with:
  - the original image as background;
  - translated text overlaid as editable text elements;
- supports manual post-editing in **Inkscape**;
- provides a **Fyne GUI** and a **CLI** using the same core logic.

## 2. Technology Stack
- Language: **Go**
- GUI: **Fyne**
- AI providers:
  - **OpenAI**
  - **Google Gemini**
- Output format v1: **SVG**
- Output format v2: **FODG**
- Testing: Go unit, integration, golden, and GUI tests
- Linting: `golangci-lint`

## 3. Scope

### In Scope (v1)
- Single-image processing
- Input formats: JPG, JPEG, PNG, WEBP (Is 20 MB or smaller)
- Output format: SVG
- OpenAI provider
- Gemini provider
- Fyne desktop GUI
- CLI interface
- Local configuration storage
- Native-first file/folder pickers
- Automated tests
- Modular architecture for future extensions

### Out of Scope (v1)
- PDF output
- ODG output
- Batch processing
- In-app visual layout editor
- OCR-only mode without LLM
- Cloud sync
- OS keyring integration

## 4. Functional Requirements

### 4.1 Input
The application must:
- accept exactly one input image per processing run;
- support `.jpg`, `.jpeg`, `.png`, `webp`;
- validate file existence and readability;
- detect image dimensions;
- reject unsupported or invalid files with clear errors.

### 4.2 Processing
The application must:
- send the image to the selected AI provider;
- request structured output in a unified internal schema;
- detect text blocks approximately matching the original layout;
- translate text from **SOURCE LANG** to **DESTINATION LANG**;
- by default source language is **Ukrainian** and destination language is **German**;
- preserve rough block/column structure;
- return confidence values where available.

### 4.3 Output
The application must generate an **A4 SVG** file that:
- embeds the source image as the page background;
- overlays translated text as editable SVG text elements;
- preserves editable text, not paths;
- uses UTF-8 encoding;
- opens correctly in Inkscape;
- is valid XML/SVG.

### 4.4 GUI
The GUI must support:
- selecting an input image;
- selecting an output directory;
- editing an output filename template;
- selecting provider and model and specific AI/model options;
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
- Primary target: Linux
- Secondary targets: Windows, macOS, android

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

### 6.3 Provider Interface
All providers must implement a shared interface.

```go
type Provider interface {
    Name() string
    AnalyzePage(ctx context.Context, req AnalyzeRequest) (*DocumentPage, error)
    ValidateConfig(cfg ProviderConfig) error
    SupportedModels() []string
}
````

### 6.4 Renderer Interface

All renderers must implement a shared interface.

```go
type Renderer interface {
    Name() string
    Render(ctx context.Context, page *DocumentPage, opts RenderOptions) ([]byte, error)
    FileExtension() string
}
```

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

### 8.3 Prompt Contract

The provider request must instruct the model to:

* read handwritten or mixed handwritten/printed Ukrainian text;
* translate it into German;
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

## 9. SVG Renderer Requirements

### 9.1 Page Format

* A4 page size
* correct page dimensions
* proportional background image placement
* visible overlay text on top of the image

### 9.2 Text Rendering

Each text block must be rendered as an editable SVG text element with:

* coordinates
* font family
* font size
* fill color
* opacity
* optional rotation
* text alignment

### 9.3 File Naming

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

## 10. GUI Requirements

### 10.1 Main Screen

The main window must contain:

* input file field
* input browse button
* output directory field
* output directory browse button
* output filename template field
* provider selector
* model selector
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

### 10.3 File Pickers

* Use native file/folder pickers as the primary implementation.

### 10.3 Adaptive design

* Gui must be flexible and looks confient in different screen sizes and orienttaions, including Android

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
--output-name
--provider
--model
--source-lang
--target-lang
--font-family
--font-size
--opacity
--color
--page-format
--config
--save-layout-json
--timeout
--verbose
```

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
* default model
* default output directory
* default font family
* default font size
* output filename template
* overlay color
* overlay opacity
* preserve-columns setting
* GUI preferences

### 12.2 Rules

* GUI and CLI must use the same effective configuration model.
* CLI flags override config file values.
* Environment variables override config defaults and are overridden by CLI flags.
* API keys must never appear in logs or error messages.
* API key fields in GUI must be masked.

### 12.3 Precedence

Configuration precedence:

1. built-in defaults
2. config file / local preferences
3. environment variables
4. CLI flags

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
* OpenAI provider
* Gemini provider
* SVG renderer
* local config support
* automated tests
* README
* example config
* sample input/output assets

### 16.2 v2 Preparation

The v1 architecture must allow adding:

* FODG renderer
* additional providers
* batch mode
* layout JSON export/import

## 17. Acceptance Criteria

### Functional

* User can select an image in GUI and generate an SVG successfully.
* User can process the same image from CLI.
* Generated SVG opens in Inkscape.
* Overlay text remains editable as text.
* Settings persist locally.
* Provider selection works for both OpenAI and Gemini.

### Technical

* Provider layer is isolated from renderer layer.
* GUI and CLI both use the same core processing flow.
* SVG renderer is independent from provider implementation.
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
13. Packaging and release preparation
14. FODG renderer in v2
