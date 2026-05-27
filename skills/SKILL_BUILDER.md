# Skill Builder Guide (Agent-Focused)

This guide is the canonical checklist for coding agents that generate new skills for context-monster-cli.

Use this when you create a skill manually or when you call the `build_skill` meta-skill.

## Quick Contract

A valid skill is a directory under `skills/` that contains:

1. `manifest.json`
2. One executable implementation target referenced by `command`

Minimum `manifest.json` shape:

```json
{
  "name": "my_skill",
  "description": "What this skill does.",
  "parameters": {
    "type": "object",
    "properties": {
      "input": { "type": "string", "description": "The input value." }
    },
    "required": ["input"]
  },
  "command": "./my_skill"
}
```

Contract requirements:

- `name` must match the skill directory name.
- `parameters` must be a valid JSON object matching Ollama tool schema.
- The skill process receives one argument: JSON payload in `argv[1]`.
- The skill must parse JSON from `argv[1]`, perform work, and print result to stdout.

## build_skill Inputs

The `build_skill` tool expects:

- `name` (snake_case)
- `description`
- `parameters` (JSON object as a string)
- `language` (`go`, `python`, or `bash`)
- `code` (full source)

Language mapping used by scaffolder:

- `go`: writes `main.go`, sets `command` to `./<name>`, then runs `go build -o <name> main.go`
- `python`: writes `run.py`, sets `command` to `python3 run.py`
- `bash`: writes `run.sh`, sets `command` to `bash run.sh`

## Agent Prompt Template

Use this as the working prompt when generating a skill:

```text
Create a new context-monster-cli skill with these inputs:
- name: <snake_case_name>
- description: <one sentence>
- parameters schema JSON: <valid JSON object>
- language: <go|python|bash>
- behavior: <what the skill should do>

Requirements:
1. Parse JSON from argv[1] (Go/Bash os.Args[1], Python sys.argv[1]).
2. Validate required keys before use.
3. Print final result to stdout only.
4. Keep output deterministic and concise.
5. Ensure manifest `command` matches generated file/runtime.

Return:
- manifest.json
- implementation source file
- brief test command
```

## Minimal Implementations

### Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
)

type Args struct {
    Input string `json:"input"`
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("error: missing JSON args")
        os.Exit(1)
    }

    var a Args
    if err := json.Unmarshal([]byte(os.Args[1]), &a); err != nil {
        fmt.Printf("error: invalid JSON: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("You passed: %s\n", a.Input)
}
```

### Python

```python
import json
import sys

if len(sys.argv) < 2:
    print("error: missing JSON args")
    sys.exit(1)

try:
    args = json.loads(sys.argv[1])
except Exception as exc:
    print(f"error: invalid JSON: {exc}")
    sys.exit(1)

print(f"You passed: {args.get('input', '')}")
```

### Bash

```bash
#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "error: missing JSON args"
  exit 1
fi

json="$1"
# Keep shell skills simple; use python3 for reliable JSON parsing.
input="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1]).get("input",""))' "$json")"
echo "You passed: $input"
```

## Preflight Checklist (Before Build)

1. Skill name is snake_case and filesystem-safe.
2. `parameters` string is valid JSON.
3. `required` fields exist in `properties`.
4. Command in manifest matches generated language target.
5. Code reads `argv[1]` and writes final output to stdout.

## Verification

1. Build or create the skill.
2. Restart the agent so skill discovery reloads manifests.
3. Invoke the skill once with representative inputs.
4. Confirm output is useful and does not include debug noise.

## Common Failure Modes

- Invalid `parameters` JSON string: scaffold fails before writing usable manifest.
- Missing `argv[1]` parsing: skill runs but cannot decode tool-call arguments.
- Command mismatch: manifest points to non-existent binary/script.
- Overly verbose output: tool responses bloat context for small models.

## Standalone Persona Note

If this skill is a pure persona (not callable as a tool), `command` may be empty and `standalone` can define:

- `system_prompt`
- `tools` (allowed skill names)

For persona-specific wiki patterns, follow the wiki guidance in README.
