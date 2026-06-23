# context-monster-cli

A local-first CLI agent harness that runs entirely on your own hardware via [Ollama](https://ollama.com). Build focused, specialist agents — called **monsters** — that execute real tasks through composable, language-agnostic skills. No API keys. No cloud. No data leaving your machine.

Zero external Go dependencies — pure stdlib.

---

## Why Context Monster?

Most AI agent frameworks assume you're willing to send your data to a third-party API and pay for every token. Context Monster assumes the opposite.

**Privacy by default.** Every inference runs locally through Ollama. Your prompts, your documents, your tool outputs — none of it touches an external server. This makes Context Monster a strong fit for regulated industries, sensitive workflows, and anyone who simply prefers to keep their data where it belongs.

**Zero marginal cost.** There's no per-token billing, no subscription tier limiting your usage, and no vendor lock-in. Once your model is pulled and your skill is built, you can run it as many times as you want.

**Designed for modest hardware.** Context Monster is optimized for smaller, lightweight models running on CPU or a single consumer GPU. It's practical on a laptop, not just a cloud VM with 8 GPUs.

**Composable, not monolithic.** Skills are standalone executables — write them in Go, Python, Bash, or anything that runs on your machine. Personas give each agent a distinct identity, model configuration, and curated toolset. The two concepts stay cleanly separated so you can mix and match.

**Simple enough to understand completely.** The entire harness is pure Go stdlib. No framework magic, no hidden abstractions. If something breaks, you can read the source and fix it.

---

## Requirements

- [Go 1.21+](https://go.dev/dl/)
- [Ollama](https://ollama.com) running locally on port `11434`
- Your chosen model pulled: `ollama pull qwen3.5:4b`

## Quick Start

```bash
# 1. Clone and enter the project
git clone <repo-url> context-monster-cli
cd context-monster-cli

# 2. Compile the bundled skill binaries (one-time)
make skills
# 3. Run the agent
go run ./cmd/agent
```

When the chat opens, use slash commands locally instead of asking the model to interpret control input:

| Command | Description |
|---|---|
| `/help` | Show the available local commands |
| `/tools` | List the tools available in the current mode or persona |
| `/clear` | Reset the current chat back to the system prompt |
| `/exit` | Leave the chat immediately |
| `/quit` | Alias for `/exit` |

## Flags

| Flag | Default | Description |
|---|---|---|
| `--model` | `qwen3.5:4b` | Ollama model to use |
| `--skills-dir` | `./skills` | Directory containing skill subdirectories |
| `--personas-dir` | `./personas` | Directory containing persona subdirectories |
| `--persona` | _(none)_ | Run a named persona (see [Personas](#personas)) |
| `--debug` | `false` | Print raw Ollama response details to stderr |
| `--record` | `false` | Record trajectories to JSONL for training data (see [Trajectory Logging](#trajectory-logging)) |

```bash
go run ./cmd/agent --model qwen3.5:7b --skills-dir ./skills
go run ./cmd/agent --persona dev_journal
```

## Project Structure

```
context-monster-cli/
├── cmd/agent/
│   └── main.go              # Entry point — flag parsing, wires packages together
├── internal/                # Private packages; only importable within this module
│   ├── ollama/
│   │   └── client.go        # HTTP client for /api/chat; all Ollama API types
│   ├── skills/
│   │   ├── types.go         # Manifest/Skill structs
│   │   └── manager.go       # Load(), Execute() with 30s timeout
│   ├── personas/
│   │   ├── types.go         # PersonaManifest/Persona structs
│   │   └── manager.go       # Load(), FindByName()
│   └── agent/
│       ├── engine.go        # REPL loop, multi-turn tool-call orchestration
│       ├── pathguard.go     # Pre-flight path access validation for tool calls
│       └── commands.go      # Slash-command parsing helpers
│   └── training/
│       ├── types.go         # Trajectory, TrajectoryMessage, TrajectoryMetadata types
│       └── logger.go        # JSONL trajectory logger
├── skills/
│   ├── file_search/         # Search a directory for files by extension
│   ├── grep/                # Regex search with line-range scoping and context lines
│   ├── read_file/           # Read the contents of a file
│   ├── list_directory/      # List immediate contents of a directory
│   ├── build_skill/         # Meta-skill: scaffolds new skills at runtime
│   └── wiki_search/         # Search a local markdown wiki (index.md + pages)
├── personas/
│   └── dev_journal/         # Example wiki-backed persona
├── go.mod
├── Makefile             # `make skills` builds all bundled skill binaries
└── README.md
```

## Bundled Skills

| Skill | Parameters | Description |
|---|---|---|
| `file_search` | `dir`, `ext` | Recursively finds files matching an extension |
| `grep` | `path`, `pattern`?, `start_line`?, `end_line`?, `context_lines`? | Searches a file for a regex pattern; returns matches with line numbers and optional surrounding context. Reads a line range when no pattern is given. |
| `read_file` | `path` | Returns the full text contents of a file |
| `list_directory` | `path` | Lists entries in a directory with file/dir labels |
| `wiki_search` | `wiki_dir`, `query` | Searches a wiki's `index.md` by keyword score, returns top matching pages in one call |
| `write_file` | `path`, `content`, `overwrite`?, `append`? | Writes text to a file. Refuses to overwrite an existing file unless `overwrite: true`. Use `append: true` to add content to the end of a file without replacing it; creates the file if it does not exist. Parent directories are created automatically. |

## Bundled Personas

| Persona | Model | Description |
|---|---|---|
| `dev_journal` | `qwen3.5:9b` | Engineering journal with a persistent wiki |

## Adding Your Own Skill

Create a subdirectory under `skills/` with two files:

For agent-generated implementations, use the contract and prompt template in `skills/SKILL_BUILDER.md`.

You can also place an `AGENTS.md` file in a skill directory. If present, its content overrides the `description` field from `manifest.json`. This is useful for writing richer tool descriptions that guide the model's behaviour without cluttering the manifest:

```
skills/
└── my_skill/
    ├── manifest.json  ← keep description short or leave it empty
    └── AGENTS.md      ← detailed description/instructions for the model
```

**`skills/my_skill/manifest.json`**
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

If your skill accepts file or directory path arguments, add `path_params` so the agent can enforce persona access control on them:

```json
{
  "name": "my_skill",
  "description": "What this skill does.",
  "parameters": {
    "type": "object",
    "properties": {
      "path": { "type": "string", "description": "Path to operate on." }
    },
    "required": ["path"]
  },
  "command": "./my_skill",
  "path_params": ["path"]
}
```

**`skills/my_skill/my_skill.py`** (or any executable)
```python
import sys, json

args = json.loads(sys.argv[1])
print(f"You passed: {args['input']}")
```

The agent will discover it automatically on next startup. The `command` field is executed with the JSON arguments appended as a single string argument. Skills can be written in any language.

## Personas

**Skills** are what the agent *can do* — callable tools that execute code. **Personas** are *who* the agent *is* — an identity with a custom system prompt, a curated set of tools, and optionally its own model and context settings.

Create a subdirectory under `personas/` with a `persona.json`:

```json
{
  "name": "my_persona",
  "description": "What this persona is for.",
  "system_prompt": "You are a ...",
  "model": "qwen3:7b",
  "context_window": 8192,
  "max_tokens": 2048,
  "tools": ["read_file", "file_search"]
}
```

| Field | Required | Description |
|---|---|---|
| `name` | yes | Identifier used with `--persona <name>` |
| `description` | yes | Human-readable description |
| `system_prompt` | yes* | Injected as the first system message. Overridden by `AGENTS.md` if present |
| `model` | no | Overrides `--model`; omit to use the flag value |
| `context_window` | no | Sets Ollama's `num_ctx` option |
| `max_tokens` | no | Sets Ollama's `num_predict` option |
| `tools` | yes | List of skill names the persona can call |
| `allowed_paths` | no | List of file/directory patterns the persona may access (see [File Access Control](#file-access-control)) |
| `record` | no | Set to `true` to always record trajectories for this persona (see [Trajectory Logging](#trajectory-logging)) |

```bash
go run ./cmd/agent --persona dev_journal
```

#### AGENTS.md system prompt

Instead of writing the system prompt inline in `persona.json`, you can place an `AGENTS.md` file in the persona directory. If it exists, its content is used as the system prompt and the `system_prompt` field in `persona.json` is ignored. This makes long prompts easier to read and edit:

```
personas/
└── my_persona/
    ├── persona.json   ← set system_prompt to "" or omit it
    └── AGENTS.md      ← actual system prompt lives here
```

A shell alias hint is printed to stderr on startup:

```
# Alias hint: alias dev_journal='context-monster-cli --persona dev_journal'
```

### Wiki-Backed Personas

A persona can maintain a persistent knowledge wiki — a directory of markdown files that accumulates understanding across sessions, following the pattern described by [Andrej Karpathy](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f). Instead of re-deriving answers from scratch each time, the model writes what it learns and reads it back on future questions.

**The key point:** the wiki only stays current because you tell it to. The persona's `system_prompt` is the schema — the equivalent of Karpathy's `AGENTS.md`. It is your job as the persona author to write a prompt that explicitly instructs the model to:

1. **Query the wiki first** — call `wiki_search` before answering relevant questions
2. **Save knowledge back** — call `write_file` to create a page, then update `index.md`

Without those instructions in the prompt, the model will ignore the wiki entirely.

**Bundled skill:** `wiki_search` reads the wiki's `index.md`, scores pages by keyword match against your query, and returns the top results in one tool call — avoiding the multi-step chain of `list_directory` → `read index.md` → `read_file` × N that would otherwise burn through a small model's context.

**What you need in the prompt:**

```
## Wiki

You maintain a wiki at <wiki_dir>/. Before answering questions about <domain>,
call wiki_search with wiki_dir=<wiki_dir>. After giving detailed advice, offer
to save it. When the user agrees:
1. Call write_file with path=<wiki_dir>/pages/<topic>.md and the content.
2. Call write_file to update <wiki_dir>/index.md, appending a line under the
   right category: `- [Title](pages/<topic>.md) — one-line summary`
   Use overwrite=true to replace the existing index.md.
```

**`personas/dev_journal/persona.json`** (abbreviated)
```json
{
  "name": "dev_journal",
  "system_prompt": "You are a quiet, sharp engineering journal...\n\n## Wiki\n\nYou maintain a persistent journal wiki at personas/dev_journal/wiki/. Before responding to questions about past work, call wiki_search...",
  "tools": ["wiki_search", "read_file", "write_file", "file_search"]
}
```

The wiki directory lives inside the persona's directory:

```
personas/
└── dev_journal/
    ├── persona.json
    └── wiki/
        ├── index.md        ← catalog of all pages (LLM maintains)
        └── decisions/      ← architectural and design choices
        └── sessions/       ← dated notes on what was worked on
        └── problems/       ← bugs, blockers, and resolutions
```

The `index.md` uses simple category headers with one link per line:

```markdown
## Decisions
- [Chose SQLite over Postgres](decisions/chose-sqlite-over-postgres.md) — lightweight, no server needed for this use case

## Problems
- [Scanner blocking on large input](problems/scanner-large-input.md) — switched to bufio.Reader with a larger buffer
```

`wiki_search` parses these lines, scores them against your query by keyword match, and returns the top 5 pages' full content in a single tool call. As the wiki grows, answers get more grounded — the model cites its own prior work rather than improvising.

## Trajectory Logging

The agent can record every user turn as a structured trajectory for later use as training data, benchmarking, or evaluation. Each trajectory captures the full conversation as an OpenAI-format `messages` array — directly consumable by SFT pipelines (Axolotl, LLaMA-Factory, unsloth) without any conversion step — plus a `metadata` object for analysis and eval tooling.

Recording is enabled in two ways:

**CLI flag** — useful for one-off sessions or personas without the config set:
```bash
go run ./cmd/agent --record
go run ./cmd/agent --persona git_agent --record
```

**Persona config** — set `"record": true` in `persona.json` to always record for that persona:
```json
{
  "name": "git_agent",
  "record": true,
  ...
}
```

Trajectories are appended as newline-delimited JSON (JSONL) to:
- `personas/<name>/training/trajectories.jsonl` when a persona is active
- `./training/trajectories.jsonl` for no-persona runs

The directory is created automatically if it does not exist.

### Trajectory Schema

Each line in the JSONL file is a single JSON object with two top-level keys: `messages` and `metadata`.

```json
{
  "messages": [
    {
      "role": "system",
      "content": "You are a ..."
    },
    {
      "role": "user",
      "content": "What changed in the last commit?"
    },
    {
      "role": "assistant",
      "reasoning_content": "I need to run git log to check...",
      "tool_calls": [
        {
          "id": "call_3f1a2b",
          "type": "function",
          "function": {
            "name": "grep",
            "arguments": "{\"path\":\".\",\"pattern\":\"...\"}"
          }
        }
      ]
    },
    {
      "role": "tool",
      "tool_call_id": "call_3f1a2b",
      "name": "grep",
      "content": "..."
    },
    {
      "role": "assistant",
      "content": "The last commit added X..."
    }
  ],
  "metadata": {
    "id": "3f1a2b...",
    "model": "qwen3.5:9b",
    "provider": "ollama",
    "persona_name": "git_agent",
    "context_window": 8192,
    "input_tokens": 312,
    "output_tokens": 87,
    "total_tokens": 399,
    "success": false,
    "success_reason": "",
    "started_at": "2026-06-17T10:00:00Z",
    "completed_at": "2026-06-17T10:00:03Z"
  }
}
```

**`messages`** follows the OpenAI messages format. Each tool-call round produces an `assistant` message (with `tool_calls`) followed by one `tool` message per call. The `tool_call_id` on a `tool` message links the result back to the `assistant` tool call that requested it; `name` identifies which function produced the result.

**`reasoning_content`** on assistant messages captures chain-of-thought output from reasoning models (e.g. `qwen3`). The field name matches the convention used by DeepSeek and Qwen APIs. It is not part of the OpenAI spec and is ignored by SFT pipelines, but preserved for trajectory analysis.

**`metadata.success`** and **`metadata.success_reason`** default to `false` and `""` and are intended for manual or LLM-assisted annotation after the fact.

## File Access Control

By default a persona can read and write any path that the agent process can reach — constrained only by the OS file permissions of the user running it. The `allowed_paths` field narrows this further to an explicit allow-list you define per persona.

```json
{
  "name": "my_persona",
  "tools": ["read_file", "write_file", "grep"],
  "allowed_paths": [
    "./src",
    "./docs/**",
    "/absolute/path/to/data"
  ]
}
```

When `allowed_paths` is set, the agent validates every path-type argument in a tool call **before spawning any subprocess**. If the resolved absolute path of the argument falls outside the allow-list, the tool call is rejected and the LLM receives an error message like:

```
access denied: "/etc/passwd" is outside the persona's allowed paths
```

The LLM can then reason about the denial and respond accordingly — typically by explaining why it cannot fulfil the request or trying a different path.

If `allowed_paths` is absent or empty, all paths are permitted (backward-compatible default).

### Pattern Syntax

Each entry in `allowed_paths` is resolved relative to the **current working directory** at startup time (usually the project root) unless it is already absolute.

| Pattern form | Matches |
|---|---|
| `"./src"` or `"./src/**"` | Any file anywhere inside `./src/` |
| `"/absolute/dir"` | Any file anywhere inside that directory |
| `"/absolute/dir/file.txt"` | Exactly that one file |
| `"./logs/*.log"` | Any `.log` file directly inside `./logs/` (no subdirectories) |
| `"./src/**/*.go"` | _Not_ supported as a recursive glob — use a directory prefix instead |

> **Note:** Glob patterns use Go's `filepath.Match` semantics, which does not support `**` as a multi-segment wildcard. For recursive directory access, use the plain directory form (`"./src"`) rather than a glob.

The trailing `/**` suffix is treated as sugar for the directory prefix form and is stripped before comparison, so `"./src/**"` and `"./src"` are equivalent.

### How Enforcement Works

Path checking is a **pre-flight step** in the Go agent itself, not inside the skill binaries. The flow for every tool call:

1. The LLM emits a tool call with its arguments.
2. The agent looks up which parameters are path-type for that skill (declared via `path_params` in `manifest.json`).
3. Each path argument is resolved to an absolute path via `filepath.Abs`.
4. The resolved path is tested against `allowed_paths`. First match → allow. No match → deny.
5. **On deny:** the agent never spawns the subprocess. The denial message is injected into history as the tool result, and Ollama is re-prompted to synthesise a response.
6. **On allow:** the subprocess is spawned as normal. The `CM_ALLOWED_PATHS` environment variable is also set to the comma-joined allow-list for shell-script skills that want to perform secondary enforcement.

### Declaring Path Parameters in a Skill

For the agent to know which arguments to check, each skill manifest lists the names of its path-type parameters in `path_params`:

```json
{
  "name": "read_file",
  "parameters": {
    "type": "object",
    "properties": {
      "path": { "type": "string", "description": "Path to the file." }
    },
    "required": ["path"]
  },
  "command": "./read",
  "path_params": ["path"]
}
```

All bundled skills (`read_file`, `write_file`, `grep`, `file_search`, `list_directory`, `wiki_search`) already declare their path parameters. When writing a custom skill, add `path_params` to its manifest for access control to apply.

### Example: Scoped Code-Review Persona

A persona that is only allowed to read files in `./src` and `./tests`, and write nothing:

```json
{
  "name": "code_reviewer",
  "description": "Reviews code in ./src and ./tests only.",
  "system_prompt": "You are a meticulous code reviewer. Read source files, then provide feedback.",
  "model": "qwen3.5:9b",
  "tools": ["read_file", "grep", "list_directory", "file_search"],
  "allowed_paths": [
    "./src",
    "./tests"
  ]
}
```

If the LLM tries to read a file outside those directories — say `../secrets/.env` or `/etc/passwd` — the attempt is blocked before any subprocess runs and the error is returned to the model.

---

## How It Works

1. On startup, the agent scans `./skills/` for `manifest.json` files and registers each as an Ollama tool. In persona mode (`--persona`), it also loads `./personas/` and restricts tools and system prompt to those declared by the persona. If `allowed_paths` is set, path enforcement is armed for all subsequent tool calls.
2. User input is appended to the conversation history and sent to Ollama along with the tool definitions.
3. If Ollama returns `tool_calls`, each skill's path arguments are validated against `allowed_paths` before any subprocess is started. Denied calls return an error to the LLM; allowed calls execute as a subprocess with a 30-second timeout. Results are appended to history as `tool` role messages.
4. Ollama is prompted again with the tool results to produce a final synthesis response.
5. The agent prints `Thinking...` and `Running tool: <name>...` to stderr as progress feedback.
