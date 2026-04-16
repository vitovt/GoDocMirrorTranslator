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

## Architecture constraints
- Domain model must stay provider-agnostic and renderer-agnostic.
- GUI and CLI must share one application core.
- Renderer interface must support SVG now and FODG later.
- Provider adapters must normalize into one internal `DocumentPage` / `TextBlock` model.
- Provider instances are constructed per render from effective runtime config so credentials and advanced options do not leak into global mutable state.
- OpenAI and Gemini adapters share one prompt builder and one provider response schema, with transport kept provider-specific.
- No business logic inside GUI widgets.

## Repository conventions
- `cmd/app` is the application entrypoint.
- `internal/domain` holds core models and validation/defaulting logic.
- `internal/app` holds use cases, render orchestration, file output, and filename templating.
- `internal/config` holds local config defaults, path resolution, environment overlays, masking, and persistence.
- `internal/prompts` holds shared provider prompt builders.
- `internal/provider` holds the shared structured response contract and image-loading helpers used by provider adapters.
- `internal/provider/mock` is the development/test provider.
- `internal/provider/openai` calls the OpenAI Responses API with image input and structured output.
- `internal/provider/gemini` calls the Gemini `generateContent` API with inline image data and structured output.
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
Milestone 1 backend foundation is in place: repo skeleton, domain model, renderer interface, SVG renderer, mock provider, live OpenAI and Gemini adapters, CLI wiring, local config persistence, provider runtime-config injection, and baseline tests.

## Next implementation targets
1. Add GUI shell, validation state, and native picker integration.
2. Add golden files and broader renderer/provider tests.
3. Reconcile the accepted spec decisions back into `SPEC.md`.
4. Harden provider behavior around parse failures, refusals, and larger-image handling as real usage reveals gaps.

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
