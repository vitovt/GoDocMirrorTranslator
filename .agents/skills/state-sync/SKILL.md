---
name: state-sync
description: Use this skill when starting a non-trivial task, resuming after a pause, or finishing a task that changes architecture, milestones, repository layout, or operating rules. Do not use it for tiny isolated edits that do not affect project state.
---

# State synchronization workflow

## Read first
- `SPEC.md`
- `STATE.md`
- `.codex/local/SNAPSHOT.md` if it exists

## Update `STATE.md` when
- architecture changes;
- repo structure changes;
- milestones change;
- new constraints appear;
- validation policy changes.

## Do not put in `STATE.md`
- transient shell output;
- long logs;
- one-off debug notes;
- secrets.

## Preferred update style
- keep it concise;
- keep it durable;
- preserve existing headings where possible;
- update only the sections affected by the task.
