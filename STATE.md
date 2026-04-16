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

## Architecture constraints
- Domain model must stay provider-agnostic and renderer-agnostic.
- GUI and CLI must share one application core.
- Renderer interface must support SVG now and FODG later.
- Provider adapters must normalize into one internal `DocumentPage` / `TextBlock` model.
- No business logic inside GUI widgets.

## Repository conventions
- `cmd/app` is the application entrypoint.
- `internal/domain` holds core models.
- `internal/app` holds use cases / orchestration.
- `internal/provider/openai` and `internal/provider/gemini` hold adapters.
- `internal/renderer/svg` is the first renderer.
- `internal/renderer/fodg` is the next renderer.
- `internal/gui` contains Fyne UI only.
- `internal/cli` contains CLI wiring only.

## Validation rules
- Prefer the repository `make` targets as the primary workflow entrypoints.
- Run `make fmt-check` on changed Go files.
- Run `make test` before finalizing code changes.
- Run `make lint` when available.
- Use `make check` for the standard validation bundle.

## Current milestone
Bootstrap repository and Codex working environment.

## Next implementation targets
1. Create repo skeleton.
2. Add domain model.
3. Add renderer interface.
4. Add SVG renderer.
5. Add mock provider.
6. Add CLI.
7. Add OpenAI provider.
8. Add Gemini provider.
9. Add GUI.
10. Add tests and golden files.

## Open decisions
- Exact native picker package choice
- Exact local config file shape for the app itself
- Final SVG text wrapping strategy
- Exact FODG writer implementation strategy

## Notes
Update this file when:
- architecture changes;
- repo structure changes;
- milestone changes;
- validation policy changes;
- new hard constraints appear.
