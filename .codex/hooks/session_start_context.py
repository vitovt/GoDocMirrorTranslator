#!/usr/bin/env python3
import json
import os
import pathlib
import subprocess
import sys

MAX_BYTES = 16000

def run(cmd, cwd):
    try:
        return subprocess.check_output(cmd, cwd=cwd, text=True, stderr=subprocess.DEVNULL).strip()
    except Exception:
        return ""

def git_root(cwd):
    root = run(["git", "rev-parse", "--show-toplevel"], cwd)
    return pathlib.Path(root) if root else pathlib.Path(cwd)

def read_text(path):
    try:
        if path.exists():
            return path.read_text(encoding="utf-8")
    except Exception:
        pass
    return ""

def summarize_path(path):
    return "present" if path.exists() else "missing"

def main():
    payload = json.load(sys.stdin)
    cwd = payload.get("cwd") or os.getcwd()
    root = git_root(cwd)

    spec = root / "SPEC.md"
    state = root / "STATE.md"
    snapshot = root / ".codex" / "local" / "SNAPSHOT.md"
    gomod = root / "go.mod"
    gitdir = root / ".git"

    checks = [
        f"- SPEC.md: {summarize_path(spec)}",
        f"- STATE.md: {summarize_path(state)}",
        f"- go.mod: {summarize_path(gomod)}",
        f"- git repo: {summarize_path(gitdir)}",
    ]

    extra = []
    state_text = read_text(state).strip()
    snapshot_text = read_text(snapshot).strip()

    if state_text:
        extra.append("## STATE.md\n" + state_text[:MAX_BYTES])
    if snapshot_text:
        extra.append("## Local snapshot\n" + snapshot_text[:MAX_BYTES])

    bootstrap = "## Bootstrap checks\n" + "\n".join(checks)
    combined = bootstrap
    if extra:
        combined += "\n\n" + "\n\n".join(extra)

    print(json.dumps({
        "hookSpecificOutput": {
            "hookEventName": "SessionStart",
            "additionalContext": combined
        }
    }, ensure_ascii=False))
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
