# Phase 01: Core Architecture and Configuration

## Context Links
- [plan.md](file:///Users/vominhlong/mncode/plans/260821-1537-claude-code-golang-clone/plan.md)
- [.ck.json](file:///Users/vominhlong/mncode/.claude/.ck.json)

## Overview
- **Priority**: High
- **Current status**: In Progress
- **Description**: Initialize the Go module `github.com/vominhlong/mncode` (or `mncode`), create the project structure, configuration parser (supporting `.env`, `.claude/.ck.json`, `.claude/settings.json`, and environment variables), and base CLI command runner.

## Key Insights
- Clean dependency management using standard libraries and battle-tested Go packages (e.g. `gopkg.in/yaml.v3` for frontmatter/YAML, `github.com/peterh/liner` or `golang.org/x/term` / bubbletea / promptui for rich interactive CLI).
- Support loading API keys: `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`, `OPENROUTER_API_KEY`.

## Requirements
- Go module initialization with Go 1.24.
- Configuration struct mapping settings from `.claude` folder and runtime flags.
- Logging and error handling with clear user-facing messages.

## Related Code Files
- Create: `go.mod`
- Create: `pkg/config/types.go`
- Create: `pkg/config/loader.go`
- Create: `pkg/config/dotenv.go`

## Implementation Steps
1. Run `go mod init mncode`.
2. Add necessary dependencies (`yaml.v3`, terminal utilities).
3. Implement `pkg/config/types.go` for configuration models.
4. Implement `pkg/config/loader.go` to parse configurations from flags, environment, `.claude/settings.json`, and `.ck.json`.
5. Implement `.env` file loader in `pkg/config/dotenv.go`.

## Todo List
- [ ] Initialize `go.mod`
- [ ] Implement configuration structures and loaders
- [ ] Validate config loading with unit test

## Success Criteria
- `go build` runs without error on `pkg/config`.
