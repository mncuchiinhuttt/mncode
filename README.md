# mncode

<p align="center">
  <strong>High-Performance AI Coding Assistant CLI in Golang</strong><br>
  <em>Claude Code Alternative with Multi-Account Smart Routing, ClaudeKit Skills & Autonomous Multi-Agent Orchestration</em>
</p>

<p align="center">
  <a href="#-quick-install"><img src="https://img.shields.io/badge/Install-1--Line%20Script-brightgreen" alt="Install Script"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
</p>

---

## ⚡️ Key Features

- **Blazing-Fast Golang Core**: Single static binary with near-instant startup (< 20ms) and minimal memory footprint — no heavy Node.js runtime or Python virtualenv required.
- **Multi-Account Smart Routing & Failover**:
  - Connect unlimited **Antigravity (Gemini)** and **Codex (OpenAI)** accounts simultaneously.
  - Automatic round-robin load balancing and instant cooldown rotation on rate limits (`429`) or auth errors (`401`).
  - Auto-import existing credentials from `~/.gemini/` and `~/.openai/`.
- **Full ClaudeKit & Open Agent Skills Support**:
  - Automatically discovers and executes 60+ Claude Skills from `.claude/skills/*/SKILL.md` (Open Agent Skills Specification).
  - Built-in multi-agent delegation: `planner`, `researcher`, `scout`, `tester`, `debugger`, `code-reviewer`, `fullstack-developer`, `docs-manager`.
  - Enforces workspace rules (`.claude/rules/*.md`) and project conventions.
- **Full-Featured Tool Calling Suite**:
  - `run_command`: Shell command execution with real-time output streaming.
  - `view_file` & `replace_file_content`: Precise line-slice inspection and atomic chunk replacement.
  - `write_to_file`: Atomic file creation with auto-parent directory generation.
  - `grep_search` & `find_by_name`: Ripgrep-speed pattern searching and glob matching.
  - `list_dir` & `read_url_content`: Directory exploration and web content extraction.
  - `ask_user`: Interactive clarifying questions.
  - `invoke_subagent`: Autonomous subagent lifecycle coordination.
- **Multi-Provider LLM Engine**:
  - **Anthropic Claude**: Claude 3.7 Sonnet with Extended Thinking budget control.
  - **Google Gemini / Antigravity**: Gemini 2.5 Flash/Pro and Gemini 3.0.
  - **OpenRouter & OpenAI**: GPT-4o, o3-mini, DeepSeek R1/V3, and Ollama local models.
- **Modern Interactive Terminal REPL**:
  - Clean full-width header card and status bar.
  - Live slash command autocomplete with relevance-ranked search.
  - Ephemeral side questions (`/btw <question>`) that answer inquiries without polluting main task history.
  - Session persistence and interactive restore (`/resume`).
  - Native mouse text selection with non-disruptive copy toast notifications (`✓ Copied X characters`).
  - Dedicated Brainrot / Gen-Z 10x Developer Persona toggle (`/brainrot`).

---

## 🚀 Quick Install

### macOS & Linux (Terminal)

```bash
curl -fsSL https://raw.githubusercontent.com/mncuchiinhuttt/mncode/main/install.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/mncuchiinhuttt/mncode/main/install.ps1 | iex
```

### Build from Source (All Platforms)

```bash
git clone https://github.com/mncuchiinhuttt/mncode.git
cd mncode
go build -o bin/mncode ./cmd/mncode
```

---

## 🔑 Configuration & Authentication

### 1. Interactive Multi-Account Login

```bash
# Add Antigravity (Gemini) OAuth account
mncode /login antigravity

# Add Codex (OpenAI) account
mncode /login codex

# Auto-import credentials from local directories
mncode /account import
```

### 2. Environment Variables (Optional)

Create a `.env` file in your workspace or set system environment variables:

```env
# Anthropic Claude
ANTHROPIC_API_KEY=your_anthropic_api_key
ANTHROPIC_MODEL=claude-3-7-sonnet-20250219

# Google Gemini
GEMINI_API_KEY=your_gemini_api_key
GEMINI_MODEL=gemini-3.7-flash-high

# OpenRouter
OPENROUTER_API_KEY=your_openrouter_api_key
OPENROUTER_MODEL=anthropic/claude-3.7-sonnet
```

---

## 💻 Usage

### Interactive Mode (Default)
```bash
mncode
```

### Resume Previous Session
```bash
mncode --resume
# or
mncode -r
```

### Single Command Execution (Non-Interactive)
```bash
mncode -e "Analyze the repository structure and list available skills"
```

### Auto-Approve Permissions
```bash
mncode -y
```

---

## 🕹️ Slash Commands Cheat Sheet

| Command | Description |
| :--- | :--- |
| `/config` | Interactive settings manager (Language, Themes, Permissions, Editor mode) |
| `/btw <question>` | Ask a quick side question without consuming main task history |
| `/brainrot` | Toggle Gen Z / Sigma 10x developer personality on/off |
| `/resume` | Browse and resume previous conversation sessions with history viewer |
| `/model [name]` | Select or switch the active LLM model |
| `/effort [level]` | Configure thinking budget (None, Low, Medium, High, Max, PRO MAX) |
| `/workflow [mode]` | Switch agent orchestration mode (`auto`, `ultra-workflow`, `plan-first`, `direct`) |
| `/agents` | Open autonomous subagents & workflow monitor |
| `/skills` | Browse, search, and activate workspace skills |
| `/account list` | View multi-account pool health, active accounts, and cooldown status |
| `/theme` | Switch UI color theme (Pastel Pink, Dark, Light, Cyberpunk, Monokai) |
| `/context` | Display context window usage bar and token breakdown |
| `/compact` | Summarize and compress conversation history to free context tokens |
| `/usage` | View token consumption and request stats |
| `/clear` | Clear conversation history and reset screen |
| `/help` | Show command palette and help guide |
| `/exit` | Exit mncode assistant |

---

## 🏗️ Project Architecture

```
mncode/
├── cmd/
│   └── mncode/
│       └── main.go                 # Application entrypoint & CLI flag router
├── pkg/
│   ├── accounts/                   # Multi-account pool, 9router rotation & quota checkers
│   ├── agent/                      # ReAct agent loop, prompt builder & subagent coordinator
│   ├── config/                     # Configuration loader, .env & settings store
│   ├── provider/                   # LLM backends (Anthropic, Gemini, OpenAI, OpenRouter)
│   ├── skills/                     # Open Agent Skills parser (.claude/skills, rules, agents)
│   ├── stats/                      # Token usage tracking & heatmap visualizer
│   ├── tools/                      # Tool implementations (Bash, File, Grep, Subagent, Web)
│   └── ui/                         # Interactive terminal REPL, markdown renderer & themes
├── install.sh                      # 1-Line universal curl installer
└── LICENSE                         # MIT License
```

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).
