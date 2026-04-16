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

## Architecture constraints
- Domain model must stay provider-agnostic and renderer-agnostic.
- GUI and CLI must share one application core.
- Renderer interface must support SVG now and FODG later.
- Provider adapters must normalize into one internal `DocumentPage` / `TextBlock` model.
- No business logic inside GUI widgets.

## Repository conventions
- `cmd/app` is the application entrypoint.
- `internal/domain` holds core models and validation/defaulting logic.
- `internal/app` holds use cases, render orchestration, file output, and filename templating.
- `internal/config` holds local config defaults, path resolution, environment overlays, masking, and persistence.
- `internal/provider/mock` is the development/test provider.
- `internal/provider/openai` and `internal/provider/gemini` currently contain placeholders for upcoming adapters.
- `internal/renderer` holds renderer contracts.
- `internal/renderer/svg` contains the current SVG renderer.
- `internal/cli` contains CLI wiring only.
- `tests/integration` contains render-flow integration tests.

## Validation rules
- Prefer the repository `make` targets as the primary workflow entrypoints.
- Run `make fmt-check` on changed Go files.
- Run `make test` before finalizing code changes.
- Run `make lint` when available.
- Use `make check` for the standard validation bundle.
- In this environment, set `GOCACHE=/tmp/go-build` when the default Go cache is not writable.

## Current milestone
Milestone 1 foundation is in progress: repo skeleton, domain model, renderer interface, SVG renderer, mock provider, CLI wiring, local config persistence, and baseline tests are in place.

## Next implementation targets
1. Replace provider placeholders with real OpenAI and Gemini adapters.
2. Expand CLI coverage around config and error reporting.
3. Add GUI shell, validation state, and native picker integration.
4. Add golden files and broader renderer/provider tests.
5. Reconcile the accepted spec decisions back into `SPEC.md`.

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
