---
name: go-repo-workflow
description: Use this skill when changing Go code, package structure, tests, CLI wiring, provider adapters, or renderers in this repository. Do not use it for pure documentation-only tasks.
---

# Go repository workflow

## Purpose
Apply the repository's Go engineering rules consistently.

## Required steps
1. Inspect `SPEC.md`, `STATE.md`, and relevant package boundaries before editing.
2. Keep changes small and localized.
3. Prefer explicit, idiomatic Go.
4. Keep interfaces minimal.
5. Keep provider adapters isolated.
6. Keep renderers isolated.
7. Do not put business logic in CLI or Fyne widgets.
8. Run:
   - `gofmt -w` on modified Go files
   - `go test ./...`
   - `golangci-lint run` when available
9. If validation fails, summarize the exact failure and do not hide it.
