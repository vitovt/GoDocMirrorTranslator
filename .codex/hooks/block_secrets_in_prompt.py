#!/usr/bin/env python3
import json
import re
import sys

PATTERNS = [
    r"sk-[A-Za-z0-9]{20,}",
    r"AIza[0-9A-Za-z\-_]{20,}",
    r"-----BEGIN (RSA|EC|OPENSSH|PGP) PRIVATE KEY-----",
    r"(?i)openai[_-]?api[_-]?key\s*[:=]\s*['\"]?[A-Za-z0-9_\-]{16,}",
    r"(?i)gemini[_-]?api[_-]?key\s*[:=]\s*['\"]?[A-Za-z0-9_\-]{16,}"
]

def main():
    payload = json.load(sys.stdin)
    prompt = payload.get("prompt", "") or ""
    for pattern in PATTERNS:
        if re.search(pattern, prompt):
            print(json.dumps({
                "continue": False,
                "stopReason": "Potential secret detected in prompt",
                "systemMessage": "Blocked prompt because it appears to contain a secret."
            }))
            return 0
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
