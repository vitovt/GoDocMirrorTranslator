#!/usr/bin/env python3
import datetime as dt
import json
import os
import pathlib
import subprocess
import sys

def run(cmd, cwd):
    try:
        return subprocess.check_output(cmd, cwd=cwd, text=True, stderr=subprocess.DEVNULL).strip()
    except Exception:
        return ""

def git_root(cwd):
    root = run(["git", "rev-parse", "--show-toplevel"], cwd)
    return pathlib.Path(root) if root else pathlib.Path(cwd)

def main():
    payload = json.load(sys.stdin)
    cwd = payload.get("cwd") or os.getcwd()
    root = git_root(cwd)

    branch = run(["git", "branch", "--show-current"], root)
    status_short = run(["git", "status", "--short"], root)
    changed_files = run(["git", "diff", "--name-only"], root)
    last_msg = (payload.get("last_assistant_message") or "").strip()

    snapshot_dir = root / ".codex" / "local"
    snapshot_dir.mkdir(parents=True, exist_ok=True)
    snapshot_path = snapshot_dir / "SNAPSHOT.md"

    now = dt.datetime.utcnow().replace(microsecond=0).isoformat() + "Z"

    content = f"""# SNAPSHOT

## Updated
{now}

## Branch
{branch or "(unknown)"}

## Working tree status
```text
{status_short or "(clean)"}
```

## Changed files
```text
{changed_files or "(none)"}
```

## Last assistant summary
{last_msg or "(none)"}
"""

    snapshot_path.write_text(content, encoding="utf-8")

    print(json.dumps({
        "continue": True,
        "systemMessage": "Local snapshot saved to .codex/local/SNAPSHOT.md"
    }, ensure_ascii=False))
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
