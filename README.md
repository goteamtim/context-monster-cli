# context-monster-cli

A minimal, local-first CLI agent that connects to a running [Ollama](https://ollama.com) instance and extends the model's capabilities through dynamically loaded, language-agnostic "skills".

Zero external Go dependencies — pure stdlib.

Built for developers who want capable, customizable AI agents without API costs or data leaving their machine.

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
cd skills/file_search   && go build -o search search.go && cd ../..
cd skills/read_file     && go build -o read   read.go   && cd ../..
cd skills/list_directory && go build -o list  list.go   && cd ../..

# 3. Run the agent
go run ./cmd/agent
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--model` | `qwen3.5:4b` | Ollama model to use |
| `--skills-dir` | `./skills` | Directory containing skill subdirectories |
| `--skill` | _(none)_ | Run a named standalone persona (see [Standalone Personas](#standalone-personas)) |
| `--debug` | `false` | Print raw Ollama response details to stderr |

```bash
go run ./cmd/agent --model qwen3.5:7b --skills-dir ./skills
go run ./cmd/agent --skill running_coach
```

## Project Structure

```
context-monster-cli/
├── cmd/agent/
│   └── main.go              # Entry point — flag parsing, wires packages together
├── pkg/
│   ├── ollama/
│   │   └── client.go        # HTTP client for /api/chat; all Ollama API types
│   ├── skills/
│   │   ├── types.go         # Manifest/Skill structs, ToOllamaTool() conversion
│   │   └── manager.go       # LoadSkills(), Execute() with 30s timeout
│   └── agent/
│       └── engine.go        # REPL loop, multi-turn tool-call orchestration
├── skills/
│   ├── file_search/         # Search a directory for files by extension
│   ├── read_file/           # Read the contents of a file
│   ├── list_directory/      # List immediate contents of a directory
│   ├── build_skill/         # Meta-skill: scaffolds new skills at runtime
│   ├── wiki_search/         # Search a local markdown wiki (index.md + pages)
│   └── running_coach/       # Example wiki-backed standalone persona
├── go.mod
└── README.md
```

## Bundled Skills

| Skill | Parameters | Description |
|---|---|---|
| `file_search` | `dir`, `ext` | Recursively finds files matching an extension |
| `read_file` | `path` | Returns the full text contents of a file |
| `list_directory` | `path` | Lists entries in a directory with file/dir labels |
| `wiki_search` | `wiki_dir`, `query` | Searches a wiki's `index.md` by keyword score, returns top matching pages in one call |

## Adding Your Own Skill

Create a subdirectory under `skills/` with two files:

For agent-generated implementations, use the contract and prompt template in `skills/SKILL_BUILDER.md`.

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

**`skills/my_skill/my_skill.py`** (or any executable)
```python
import sys, json

args = json.loads(sys.argv[1])
print(f"You passed: {args['input']}")
```

The agent will discover it automatically on next startup. The `command` field is executed with the JSON arguments appended as a single string argument. Skills can be written in any language.

## Standalone Personas

Any skill can include a `standalone` block in its manifest to become a self-contained persona. When launched with `--skill <name>`, the agent:

- Injects the persona's `system_prompt` instead of the default one
- Restricts available tools to the curated `tools` list
- Prints a shell alias hint to stderr so you can set up a permanent shortcut

**`skills/running_coach/manifest.json`**
```json
{
  "name": "running_coach",
  "description": "A standalone running coach persona.",
  "parameters": { "type": "object", "properties": {}, "required": [] },
  "command": "",
  "standalone": {
    "system_prompt": "You are an expert running coach...",
    "tools": ["read_file", "file_search"]
  }
}
```

```bash
go run ./cmd/agent --skill running_coach
# Prints to stderr:
# Alias hint: alias running_coach='context-monster-cli --skill running_coach'
```

- `command` can be an empty string for pure-persona skills that are never called as a tool
- A skill can be both a callable tool AND a persona simultaneously
- The `tools` list controls exactly which other skills the persona can invoke — omit `build_skill` unless you want the persona to be able to create new skills

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

**`skills/running_coach/manifest.json`** (abbreviated)
```json
{
  "standalone": {
    "system_prompt": "You are an expert running coach...\n\n## Wiki\n\nYou maintain a persistent knowledge wiki at skills/running_coach/wiki/. Before answering any running question, call wiki_search...",
    "tools": ["wiki_search", "read_file", "write_file", "file_search"]
  }
}
```

The wiki directory lives alongside the persona's skill directory:

```
skills/
└── running_coach/
    ├── manifest.json
    └── wiki/
        ├── index.md        ← catalog of all pages (LLM maintains)
        └── pages/          ← one markdown file per topic (LLM writes)
```

The `index.md` uses simple category headers with one link per line:

```markdown
## Training Plans
- [12-Week Marathon Plan](pages/marathon-12-week.md) — beginner-friendly base-building plan

## Injury Prevention
- [IT Band Syndrome](pages/it-band.md) — causes, treatment, and return-to-run protocol
```

`wiki_search` parses these lines, scores them against your query by keyword match, and returns the top 5 pages' full content in a single tool call. As the wiki grows, answers get more grounded — the model cites its own prior work rather than improvising.

## How It Works

1. On startup, the agent scans `./skills/` for `manifest.json` files and registers each as an Ollama tool.
2. User input is appended to the conversation history and sent to Ollama along with the tool definitions.
3. If Ollama returns `tool_calls`, each skill is executed as a subprocess with a 30-second timeout. Results are appended to history as `tool` role messages.
4. Ollama is prompted again with the tool results to produce a final synthesis response.
5. The agent prints `Thinking...` and `Running tool: <name>...` to stderr as progress feedback.
