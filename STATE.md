# STATE.md

## Project
Handwritten Overlay Translator

## Goal
A Go-based desktop and CLI tool that submits a handwritten or mixed document image to an LLM, receives structured translation layout data, and renders an editable A4 SVG or FODG overlay.

## Source of truth
- `SPEC.md` is the primary product and implementation specification.
- `AGENTS.md` defines repository operating rules for the coding agent.
- `STATE.md` records durable project state, workflow, and milestones.

## Current scope
- Language: Go
- GUI: Fyne
- Providers: OpenAI, Gemini
- Output formats: SVG, FODG
- Native-first file/folder pickers
- Local config for provider selection, keys, and rendering defaults
- Modular architecture
- Tests from day one
- Mock provider available for development and integration tests
- Live OpenAI and Gemini HTTP adapters using one shared structured-output schema and prompt contract
- Live provider adapters now also share structured-output text normalization/parsing so fenced JSON and common empty-output failures are handled consistently
- FODG rendering is now implemented through `internal/renderer/fodg`, wired through app, CLI, config defaults, integration tests, and the GUI output-format selector
- Versioned saved-layout JSON export and import are now part of the shared app core, including rerendering without provider calls and relative same-folder source-image paths inside saved layout files
- Initial Fyne desktop GUI shell with validation, async processing, config persistence, picker adapters, layout-JSON rerender flow, and overwrite confirmation
- GUI design settings now expose readability controls for font weight, outline, background, and shadow, all mapped to shared renderer options
- GUI now presents text/background/shadow opacity as percentages and shows renderer-specific formatting guidance so SVG-vs-FODG differences are visible in-app
- GUI main settings now include an optional persisted image-description context field that is passed into shared provider prompts during Analyze only and omitted entirely when empty
- GUI output action now adapts by platform: desktop opens the output folder, mobile opens the generated output file
- Desktop GUI builds now use OS-native file and folder pickers via `github.com/sqweek/dialog`, with Fyne dialog fallback kept for Android and unsupported desktop backends
- Linux-host Windows builds now default to the MinGW-w64 cross-compiler path in `make windows` / `make snapshot` via `WINDOWS_CC`, instead of relying on the host `gcc`
- Android GUI startup now resolves local config through the app sandbox `FILESDIR` fallback when `os.UserConfigDir()` is unavailable
- README, example config, sample input/output assets, and an Android app icon are now checked into the repo as v1 deliverables
- `make android` has been validated locally with the available Fyne + Android SDK toolchain and now emits `build/android/godocmirrortranslator.apk`
- `SPEC.md` now includes a reusable appendix for the Fyne adaptive shell pattern so the same menu/content/status layout rules can be copied into future Go + Fyne application specifications

## Architecture constraints
- Domain model must stay provider-agnostic and renderer-agnostic.
- GUI and CLI must share one application core.
- Renderer interface must support both SVG and FODG without changing the domain model.
- Provider adapters must normalize into one internal `DocumentPage` / `TextBlock` model.
- Provider instances are constructed per render from effective runtime config so credentials and advanced options do not leak into global mutable state.
- OpenAI and Gemini adapters share one prompt builder and one provider response schema, with transport kept provider-specific.
- Optional user-provided image/document description context is injected only through the shared prompt builder during provider analysis; rerender and saved layout JSON remain independent of that context.
- OpenAI and Gemini adapters also share structured-output text parsing/normalization inside `internal/provider` so transport-specific code does not duplicate JSON cleanup or parse behavior.
- No business logic inside GUI widgets.
- The GUI defaults to the shared application core and only orchestrates config, validation, picker input, and async render invocation.
- Saved layout JSON is a versioned app-owned artifact, not a provider response dump, and it is the only rerender input format supported going forward.
- GUI platform differences are handled at the edge: desktop window sizing, native OS picker integration, and folder opening stay in the GUI layer, while mobile uses file-oriented output actions and Fyne picker fallback without changing application-core render behavior.
- Config persistence must not rely solely on desktop-style user config directories; Android falls back to the app sandbox when `HOME` / `XDG_CONFIG_HOME` are unavailable.
- The preferred Fyne shell pattern is now explicit in `SPEC.md`: top menu toggle, left menu, one persistent main content pane, bottom actions, and bottom status/details, with compact overlay behavior documented as the reusable baseline for future apps.
- FODG formatting fidelity now follows LibreOffice Draw’s actual imported text-shape model: font/color/weight/background/shadow work through Draw properties, text opacity is ignored on import, and outline behaves as contour on/off rather than a true adjustable stroke.

## Repository conventions
- `cmd/app` is the application entrypoint and now also contains the Android app icon used by `make android`.
- `internal/domain` holds core models and validation/defaulting logic.
- `internal/app` holds use cases, render orchestration, versioned saved-layout JSON handling, output-target planning, file output, and filename templating.
- `internal/config` holds local config defaults, path resolution, environment overlays, masking, persistence, and readability defaults shared by CLI and GUI.
- `internal/prompts` holds shared provider prompt builders.
- `internal/provider` holds the shared structured response contract, structured-output parsing helpers, and image-loading helpers used by provider adapters.
- `internal/provider/mock` is the development/test provider.
- `internal/provider/openai` calls the OpenAI Responses API with image input and structured output.
- `internal/provider/gemini` calls the Gemini `generateContent` API with inline image data and structured output.
- `internal/renderer` holds renderer contracts.
- `internal/renderer/svg` contains the current SVG renderer.
- `internal/renderer/fodg` contains the flat LibreOffice Draw renderer and golden fixture coverage, plus a LibreOffice round-trip smoke test when `soffice` is available.
- `internal/renderer/svg/testdata` holds stable SVG golden fixtures for renderer snapshots.
- `internal/cli` contains CLI wiring only.
- `internal/gui` contains the Fyne shell, validation logic, analyze/rerender orchestration, picker adapter, overwrite confirmation hook, and GUI tests.
- `tests/integration` contains render-flow integration tests.
- `examples/` contains checked-in example configuration files.
- `samples/input` and `samples/output` contain checked-in sample assets for manual verification and packaging smoke checks.
- `README.md` documents setup, CLI/GUI usage, config precedence, and sample asset locations.

## Validation rules
- Prefer the repository `make` targets as the primary workflow entrypoints.
- Run `make fmt-check` on changed Go files.
- Run `make test` before finalizing code changes.
- Run `make lint` when available.
- Use `make check` for the standard validation bundle.
- In this environment, set `GOCACHE=/tmp/go-build` when the default Go cache is not writable.
- In this environment, GUI test and packaging runs are most stable with `LANG=C.UTF-8 LC_ALL=C.UTF-8 PATH=/tmp/bin:$PATH GOCACHE=/tmp/go-build GOLANGCI_LINT_CACHE=/tmp/golangci-cache XDG_CACHE_HOME=/tmp`.

## Current milestone
Milestone 1 is effectively complete in the current repo: repo skeleton, domain model, renderer interface, SVG and FODG renderers, mock provider, live OpenAI and Gemini adapters, CLI wiring, desktop/mobile-aware GUI shell, local config persistence, provider runtime-config injection, coverage targets for core/config/renderer/provider packages, README, example config, sample assets, Android app icon, validated Android APK packaging, baseline tests, renderer golden files, and GUI output-format selection.

## Next implementation targets
1. Expand golden files and broader renderer/provider edge-case tests, especially around readability decorations and FODG fidelity.
2. Polish the GUI shell further for Android/desktop differences and additional provider-specific settings as real usage drives them.
3. Add packaging and release documentation around the validated desktop/Android build flow.
4. Decide whether future office-format work should target ODG export or deeper FODG fidelity improvements.

## Open decisions
- Final SVG/FODG text wrapping strategy
- How far to push FODG fidelity beyond the current flat ODF writer

## Notes
Update this file when:
- architecture changes;
- repo structure changes;
- milestone changes;
- validation policy changes;
- new hard constraints appear.
