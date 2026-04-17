# STATE.md

## Project
Handwritten Overlay Translator

## Goal
A Go-based desktop and CLI tool that submits a handwritten or mixed document image to an LLM, receives structured Ukrainian-to-German translation layout data, and renders an editable A4 SVG overlay. FODG support comes next.

## Source of truth
- `SPEC.md` is the primary product and implementation specification.
- `AGENTS.md` defines repository operating rules for the coding agent.
- `STATE.md` records durable project state, workflow, and milestones.

## Current scope
- Language: Go
- GUI: Fyne
- Providers: OpenAI, Gemini
- Output v1: SVG
- Output v2: FODG
- Native-first file/folder pickers
- Local config for provider selection, keys, and rendering defaults
- Modular architecture
- Tests from day one
- Mock provider available for development and integration tests
- Live OpenAI and Gemini HTTP adapters using one shared structured-output schema and prompt contract
- Live provider adapters now also share structured-output text normalization/parsing so fenced JSON and common empty-output failures are handled consistently
- Initial Fyne desktop GUI shell with validation, async processing, config persistence, and picker adapters
- GUI-controlled optional layout JSON export persisted through GUI preferences
- GUI output action now adapts by platform: desktop opens the output folder, mobile opens the generated output file
- README, example config, sample input/output assets, and an Android app icon are now checked into the repo as v1 deliverables
- `make android` has been validated locally with the available Fyne + Android SDK toolchain and now emits `build/android/godocmirrortranslator.apk`

## Architecture constraints
- Domain model must stay provider-agnostic and renderer-agnostic.
- GUI and CLI must share one application core.
- Renderer interface must support SVG now and FODG later.
- Provider adapters must normalize into one internal `DocumentPage` / `TextBlock` model.
- Provider instances are constructed per render from effective runtime config so credentials and advanced options do not leak into global mutable state.
- OpenAI and Gemini adapters share one prompt builder and one provider response schema, with transport kept provider-specific.
- OpenAI and Gemini adapters also share structured-output text parsing/normalization inside `internal/provider` so transport-specific code does not duplicate JSON cleanup or parse behavior.
- No business logic inside GUI widgets.
- The GUI defaults to the shared application core and only orchestrates config, validation, picker input, and async render invocation.
- GUI platform differences are handled at the edge: desktop window sizing/folder opening stays in the GUI layer, while mobile uses file-oriented output actions without changing application-core render behavior.

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
Milestone 1 is effectively complete in the current repo: repo skeleton, domain model, renderer interface, SVG renderer, mock provider, live OpenAI and Gemini adapters, CLI wiring, desktop/mobile-aware GUI shell, local config persistence, provider runtime-config injection, coverage targets for core/config/renderer/provider packages, README, example config, sample assets, Android app icon, validated Android APK packaging, baseline tests, and an SVG golden snapshot.

## Next implementation targets
1. Polish the GUI shell further for Android/desktop differences and additional provider-specific settings as real usage drives them.
2. Expand golden files and broader renderer/provider edge-case tests.
3. Start the v2 FODG renderer slice without disturbing the current SVG path.
4. Add packaging and release documentation around the validated desktop/Android build flow.

## Open decisions
- Exact native picker package choice
- Final SVG text wrapping strategy
- Exact FODG writer implementation strategy

## Notes
Update this file when:
- architecture changes;
- repo structure changes;
- milestone changes;
- validation policy changes;
- new hard constraints appear.
