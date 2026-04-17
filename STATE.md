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
- Initial Fyne desktop GUI shell with validation, async processing, config persistence, and picker adapters
- GUI-controlled optional layout JSON export persisted through GUI preferences
- GUI output action now adapts by platform: desktop opens the output folder, mobile opens the generated output file
- Desktop GUI builds now use OS-native file and folder pickers via `github.com/sqweek/dialog`, with Fyne dialog fallback kept for Android and unsupported desktop backends
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
- OpenAI and Gemini adapters also share structured-output text parsing/normalization inside `internal/provider` so transport-specific code does not duplicate JSON cleanup or parse behavior.
- No business logic inside GUI widgets.
- The GUI defaults to the shared application core and only orchestrates config, validation, picker input, and async render invocation.
- GUI platform differences are handled at the edge: desktop window sizing, native OS picker integration, and folder opening stay in the GUI layer, while mobile uses file-oriented output actions and Fyne picker fallback without changing application-core render behavior.
- The preferred Fyne shell pattern is now explicit in `SPEC.md`: top menu toggle, left menu, one persistent main content pane, bottom actions, and bottom status/details, with compact overlay behavior documented as the reusable baseline for future apps.

## Repository conventions
- `cmd/app` is the application entrypoint and now also contains the Android app icon used by `make android`.
- `internal/domain` holds core models and validation/defaulting logic.
- `internal/app` holds use cases, render orchestration, file output, and filename templating.
- `internal/config` holds local config defaults, path resolution, environment overlays, masking, and persistence.
- `internal/config` also stores GUI preferences such as whether layout JSON export is enabled by default in the GUI.
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
- `internal/gui` contains the Fyne shell, validation logic, picker adapter, and GUI tests.
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

## Current milestone
Milestone 1 is effectively complete in the current repo: repo skeleton, domain model, renderer interface, SVG and FODG renderers, mock provider, live OpenAI and Gemini adapters, CLI wiring, desktop/mobile-aware GUI shell, local config persistence, provider runtime-config injection, coverage targets for core/config/renderer/provider packages, README, example config, sample assets, Android app icon, validated Android APK packaging, baseline tests, renderer golden files, and GUI output-format selection.

## Next implementation targets
1. Polish the GUI shell further for Android/desktop differences and additional provider-specific settings as real usage drives them.
2. Expand golden files and broader renderer/provider edge-case tests, especially around FODG fidelity and renderer parity.
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
