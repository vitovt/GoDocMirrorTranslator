#!/usr/bin/env python3
import json
import re
import sys

BLOCK_PATTERNS = [
    r"\brm\s+-rf\s+/\b",
    r"\brm\s+-rf\s+\.\b",
    r"\bgit\s+reset\s+--hard\b",
    r"\bgit\s+clean\s+-fdx\b",
    r"\bsudo\b",
    r":\s*>\s*/dev/sd[a-z]",
    r"\bcurl\b.*\|\s*(bash|sh)\b"
]

def main():
    payload = json.load(sys.stdin)
    tool_input = payload.get("tool_input") or {}
    cmd = (tool_input.get("command") or "").strip()

    for pattern in BLOCK_PATTERNS:
        if re.search(pattern, cmd):
            print(json.dumps({
                "hookSpecificOutput": {
                    "hookEventName": "PreToolUse",
                    "permissionDecision": "deny",
                    "permissionDecisionReason": f"Blocked potentially destructive command: {cmd}"
                },
                "systemMessage": "Blocked a destructive Bash command."
            }))
            return 0
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
