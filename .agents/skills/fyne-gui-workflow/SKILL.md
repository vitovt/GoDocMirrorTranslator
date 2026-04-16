---
name: fyne-gui-workflow
description: Use this skill when changing Fyne GUI code, file pickers, UI validation, asynchronous task handling, or GUI tests in this repository. Do not use it for provider-only or renderer-only changes.
---

# Fyne GUI workflow

## Rules
- Keep GUI thin.
- Move business logic into application services.
- Never block the UI thread with network or long-running file operations.
- Prefer native-first file and folder picker adapters.
- Preserve restorable user settings.
- Add or update GUI tests when changing:
  - validation;
  - enabled/disabled state;
  - provider/model selection;
  - settings persistence;
  - success/error flows.

## Validation
- Confirm the GUI still maps cleanly to the shared application core.
- Confirm no provider-specific parsing or renderer-specific formatting leaked into widgets.
