---
name: spec-driven-dev
description: Use this skill when planning, reviewing, or implementing work against SPEC.md. Do not use it for unrelated one-off environment tasks.
---

# Specification-driven workflow

## Rules
- Treat `SPEC.md` as the primary source of truth.
- Before coding, map the requested task to the relevant sections of `SPEC.md`.
- If the specification is ambiguous, identify the ambiguity explicitly.
- Do not silently invent requirements.
- If implementation reveals a mismatch with the specification, propose a concrete SPEC update.
- Keep code, tests, and documentation aligned with the specification.

## Deliverables
For non-trivial work:
1. State which SPEC sections apply.
2. List the concrete implementation steps.
3. Implement in small increments.
4. Update tests.
5. Update `STATE.md` if milestones or architecture changed.
