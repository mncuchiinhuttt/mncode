# mncode

<p align="center">
  <img src="assets/logo-transparent.svg" alt="mncode Logo" width="240" />
</p>

<p align="center">
  <strong>High-Performance Autonomous AI Coding Assistant CLI in Golang</strong><br>
  <em>Next-Generation Pair Programmer with Multi-Account Smart Routing, MCP Support, ClaudeKit Skills & Web Telemetry Hub</em>
</p>

<p align="center">
  <a href="#-quick-install"><img src="https://img.shields.io/badge/Install-1--Line%20Script-brightgreen" alt="Install Script"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go" alt="Go Version"></a>
  <a href="https://github.com/mncuchiinhuttt/mncode/releases"><img src="https://img.shields.io/github/v/release/mncuchiinhuttt/mncode?color=f472b6" alt="Release"></a>
  <a href="https://github.com/mncuchiinhuttt/mncode-web"><img src="https://img.shields.io/badge/Web%20Hub-mncode--web-a855f7" alt="Web Platform"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
</p>

---

## ⚡️ Key Highlights

- 🚀 **Blazing-Fast Golang Core**: Single static binary with sub-20ms cold startup and lightweight RAM footprint. No Node.js runtime, Python virtualenvs, or heavy dependencies needed.
- 🔄 **Multi-Account Smart Routing & Failover**:
  - Connect unlimited **Antigravity (Gemini)**, **Codex (OpenAI)**, **Anthropic Claude**, and **OpenCode / Stealth** accounts simultaneously.
  - Automatic round-robin load balancing and instant cooldown rotation on rate limits (`429`) or quota exhaustion.
  - Auto-import existing credentials directly from `~/.gemini/` and `~/.openai/`.
- 🧠 **Cutting-Edge Reasoning Models & Thinking Budget**:
  - Full support for **`ox-alpha` / `stealth/ox-alpha`** (1,048,576 context tokens reasoning model with 131k output tokens).
  - Extended thinking budgets (`/effort` up to **PRO MAX 64,000 tokens**) for Claude 3.7 Sonnet, Gemini 3.7 Flash Thinking, and o3-mini.
- 🔌 **Model Context Protocol (MCP) Integration**:
  - MCP client support over stdio with interactive server manager (`/mcp`).
  - Dynamic tool discovery from connected servers with explicit restart/reload controls; HTTP/SSE transports are not enabled yet.
- 🌐 **Web Platform & Analytics Hub Sync**:
  - Connect with [`mncode-web`](https://github.com/mncuchiinhuttt/mncode-web) via API Key or CLI login (`/login web`).
  - Automatic daily telemetry synchronization (token consumption, model distribution, request volume).
  - Web dashboard, user role management, and built-in feedback reporting (`/feedback`).
- 🛠️ **Autonomous Multi-Agent Orchestration & Skills**:
  - Discover and execute 60+ modular skills from `.claude/skills/*/SKILL.md`.
  - Specialized built-in subagents: `planner`, `researcher`, `scout`, `tester`, `debugger`, `code-reviewer`, `docs-manager`.
  - 4 Orchestration workflows (`/workflow`): `auto`, `ultra-workflow`, `plan-first`, and `direct`.
- 🎨 **Modern Interactive TUI & Overlays**:
  - Ephemeral status card overlay (`/status`) dismissable with `Esc` without cluttering terminal history.
  - Live fuzzy autocomplete dropdown with argument isolation.
  - Ephemeral side questions (`/btw <question>`) that answer inquiries without polluting main conversation context.
  - Session persistence, history viewer, and interactive resume (`/resume`).
  - Native mouse text selection with smooth copy toast notifications.
  - Gen-Z / Sigma 10x Developer Persona toggle (`/brainrot`).

---

## 🚀 Quick Install

### macOS & Linux (Universal 1-Line Script)

```bash
curl -fsSL https://raw.githubusercontent.com/mncuchiinhuttt/mncode/main/install.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/mncuchiinhuttt/mncode/main/install.ps1 | iex
```

### Build from Source

```bash
git clone https://github.com/mncuchiinhuttt/mncode.git
cd mncode
go build -ldflags="-s -w" -o bin/mncode ./cmd/mncode
```

---

## 🕹️ Interactive Slash Commands

| Command | Description |
| :--- | :--- |
| `/resolve` | **Autonomous Git Merge Conflict Resolver**: Merges conflicting branches cleanly |
| `/db` | **Database Explorer & SQL Query**: Inspect tables, schemas, and run queries |
| `/api` | **In-Terminal REST / HTTP Tester**: Send requests with syntax JSON & latency meter |
| `/tree` | **Interactive ASCII File Tree**: Visualize codebase structure and file sizes |
| `/share` | **Web Session Share**: Export & publish public transcripts to `mncode.dev/share/[id]` |
| `/export [path]` | **ShareGPT Trajectory**: Export frozen session history as private `0600` JSON |
| `/doctor` | **Workspace Health Audit**: Diagnostic scorecard, runtimes, and file size limits |
| `/commit [-p]` | **AI Semantic Commit**: Conventional commit generation, auto-stage & push |
| `/review` | **Pre-Commit Code Auditor**: Scan uncommitted diffs for secrets, SQLi, and smells |
| `/drift [init\|check]` | **Architectural Drift Sentinel**: Compare AST, exported symbols, imports, cycles, and optional policy rules to a private baseline |
| `/sandbox [init\|run\|test]` | **Fixture Harness**: Run allowlisted argv cases in a bounded temporary copy; `/shadow` is intentionally rejected |
| `/index [build\|query]` | **Local Code Index**: Search normalized source terms and AST symbols with BM25 ranking and stale-index checks |
| `/arena [review]` | **Red-Team PR Arena**: Fan out bounded diff review to security, correctness, and maintainability adversaries |
| `/replay [start\|show\|export]` / `/fork` | **Flight Recorder**: Record scrubbed lifecycle events and fork conversation context without replaying tool effects |
| `/spec [new\|check\|matrix]` | **Spec-First Contracts**: Define deterministic invariants/cases and verify them without mutating the workspace |
| `/symbol <name>` | **AST Symbol Search**: Find functions, structs, interfaces, and classes instantly |
| `/scratch [ext]` | **Code Sandbox**: Open, evaluate, and test temporary code snippets |
| `/undo` / `/rewind` | **Checkpoint Rollback**: Revert last agent turn and restore workspace files |
| `/diff` | **Diff Inspector**: View syntax-highlighted uncommitted workspace changes |
| `/steer <msg>` | **Real-Time Steering**: Inject high-priority course corrections into thought loop |
| `/queue <msg>` | **Prompt Queue**: Enqueue prompts for hands-free sequential execution |
| `/changelog` | **Release Notes Generator**: Synthesize categorized notes from git commits |
| `/skills [install]` | Browse, activate, or install skills directly from `skills.sh` |
| `/status` | Open interactive session status overlay (`Esc` / `q` to dismiss) |
| `/model [name]` | Interactive model catalog selector (`ox-alpha`, `claude-3.7-sonnet`, `gemini-3.7-flash`, `o3-mini`, etc.) |
| `/effort [level]` | Configure thinking token budget (`None`, `Low`, `Medium`, `High`, `Max`, `PRO MAX`) |
| `/workflow [mode]` | Switch orchestration mode (`auto`, `ultra-workflow`, `plan-first`, `direct`) |
| `/mcp` | Model Context Protocol (MCP) server manager and tool inspector |
| `/search [engine]` | **Configuration only**: choose the backend or run `/search setup`; prompts trigger `search_web` automatically |
| `/agents` | Inspect active subagents and multi-agent execution pipeline |
| `/account` | Manage multi-account pool, health checks, and cooldown status |
| `/quota` | Check real-time rate limit headroom and remaining token quotas |
| `/btw <question>` | Ask a side question without consuming conversation context |
| `/theme` | Switch UI color theme (Pastel Pink, Dark, Light, Cyberpunk, Monokai) |
| `/context` | Display context window usage bar and token breakdown |
| `/compact` | Summarize and compress conversation history to free context tokens |
| `/resume` | Browse and resume previous conversation sessions |
| `/feedback <msg>` | Submit direct feedback or bug report to the web admin portal |
| `/brainrot` | Toggle Gen-Z / Sigma 10x developer personality on/off |
| `/clear` | Clear conversation history and reset screen |
| `/help` | Display command palette and keybinding cheat sheet |
| `/exit` | Gracefully exit mncode |

The six new command families are local-first and bounded: `/sandbox` uses a temporary copy rather than a kernel container, `/index` is lexical BM25 plus AST ranking rather than embedding search, `/arena` is advisory and never posts or edits a PR, `/replay` never re-executes tools, and `/spec` is a deterministic contract matrix rather than a formal proof.

---

## 🔑 Authentication & Providers

### 1. Web Platform Sync (Recommended)

Generate an API key on your [mncode-web](https://github.com/mncuchiinhuttt/mncode-web) dashboard:

```bash
# Login via API key
mncode /login web

# Or login via email/password
mncode /login web --email user@example.com
```

### 2. Multi-Account OAuth Pool

```bash
# Connect Google Antigravity (Gemini)
mncode /login antigravity

# Connect OpenAI Codex
mncode /login codex

# Auto-import local credentials (~/.gemini, ~/.openai)
mncode /account import
```

### 3. Direct API Keys (`.env` or Environment Variables)

```env
# Anthropic
ANTHROPIC_API_KEY=sk-ant-api03-...
ANTHROPIC_MODEL=claude-3-7-sonnet-20250219

# Google Gemini
GEMINI_API_KEY=AIzaSy...
GEMINI_MODEL=gemini-3.7-flash-high

# OpenRouter / OpenCode (ox-alpha)
OPENROUTER_API_KEY=sk-or-v1-...
OPENROUTER_MODEL=stealth/ox-alpha
```

### 4. Web Search Engines

`search_web` is an internal agent tool. Users write a normal prompt; when the harness decides current web research is needed, the model calls `search_web` automatically. Users do not need to type `/search` to perform a search.

The tool defaults to `auto` and tries engines in this order: Antigravity/Gemini, Tavily, Brave, then DuckDuckGo (skipping keyed engines that are not configured).

Configure keys in the CLI with `/search setup` or in Desktop → Settings → General → Web search. Keys persist locally in `~/.mncode/config.json` (`0700` directory, `0600` file). A `.env` file is optional; `BRAVE_API_KEY`, `TAVILY_API_KEY`, and `SEARCH_ENGINE` remain optional environment fallbacks for development and automation.

---

## 🧠 Advanced Agent Tools

- `/drift`: compare AST and dependency boundaries against a stored baseline.
- `/sandbox`: run allowlisted fixtures in bounded temporary workspace copies.
- `/index`: query a persisted BM25 and AST-aware local code index.
- `/arena`: run bounded security, correctness, and maintainability reviews against a diff.
- `/replay` and `/fork`: record scrubbed lifecycle traces and branch conversation history without replaying tool effects.
- `/spec`: define deterministic command, file, and invariant contracts.
- `lsp_tool`: semantic Go/TypeScript definition, references, hover, diagnostics, and project-wide rename through installed language servers.
- `debugger`: persistent Delve DAP sessions for breakpoints, stack traces, scopes, variables, and expression evaluation.
- `persistent_kernel`: bounded persistent Python or Node namespaces for calculations and data analysis.
- `replace_file_content` accepts an optional `FileHash` to reject edits against stale file bytes.
- Concurrent subagents can coordinate through the in-process `subagent_message` tool.

## 🏗️ Architecture & Modules

```
mncode/
├── cmd/
│   └── mncode/
│       └── main.go                 # Application entrypoint & CLI flag router
├── pkg/
│   ├── accounts/                   # Multi-account pool, auto-rotation & quota checkers
│   ├── agent/                      # ReAct loop, prompt tiers, memory scrubber & subagent coordinator
│   ├── config/                     # Configuration loader, .env & persistent settings store
│   ├── evals/                      # Isolated parallel edit-reliability benchmark harness
│   ├── mcp/                        # Model Context Protocol client & server lifecycle manager
│   ├── provider/                   # LLM backends (Anthropic, Gemini, OpenAI, OpenRouter, OpenCode)
│   ├── skills/                     # Open Agent Skills parser (.claude/skills, rules, agents)
│   ├── stats/                      # Token usage tracking, telemetry sync & visualizer
│   ├── tools/                      # Workspace tools, LSP/DAP, kernels, edits, peer messaging & web
│   └── ui/                         # Interactive terminal REPL, theme selector & overlays
├── dist/                           # Multi-platform compiled release binaries
├── install.sh                      # Universal macOS/Linux 1-line installer
├── install.ps1                     # Universal Windows PowerShell 1-line installer
└── LICENSE                         # MIT License
```

---

## 🌐 Ecosystem

- **CLI Engine**: [mncode](https://github.com/mncuchiinhuttt/mncode) — High-performance Golang CLI.
- **Web Platform**: [mncode-web](https://github.com/mncuchiinhuttt/mncode-web) — Web UI, Docs, Auth & Analytics Hub.

---

## 📄 License

Distributed under the [MIT License](LICENSE). Built with ❤️ by [mncuchiinhuttt](https://github.com/mncuchiinhuttt).
