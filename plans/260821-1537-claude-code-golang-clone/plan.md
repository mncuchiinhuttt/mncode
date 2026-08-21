# Claude Code Go Clone Implementation Plan

## Overview
Build a high-performance, modular CLI AI coding assistant in Golang (named `mncode`) that fully replicates Claude Code core functionalities and seamlessly integrates the `.claude` skill catalog, ClaudeKit engineer agents, custom rules, and multi-provider LLM backends (Anthropic Claude 3.5/3.7, Gemini, OpenAI/OpenRouter).

## Phases

| Phase | Description | Status |
|-------|-------------|--------|
| [Phase 01](phase-01-core-architecture-and-config.md) | Setup Go module, config loader, environment & CLI boilerplate | Completed |
| [Phase 02](phase-02-skills-and-agent-system.md) | Implement Claude Skills, ClaudeKit Agents & Rules parser & loader | Completed |
| [Phase 03](phase-03-builtin-tools-suite.md) | Implement Claude Code built-in tools (Bash, View, Edit, Write, Grep, Glob, ListDir, Web, Ask) | Completed |
| [Phase 04](phase-04-llm-provider-engine.md) | Implement LLM Provider Layer (Anthropic Claude API with streaming, tool calling & thinking, OpenRouter/Gemini) | Completed |
| [Phase 05](phase-05-agent-loop-and-subagents.md) | Implement Agent execution loop, system prompt composer, context manager & subagent dispatcher | Completed |
| [Phase 06](phase-06-interactive-terminal-ui.md) | Implement Rich Terminal UI (REPL, streaming markdown, diffs, slash commands, permissions) | Completed |
| [Phase 07](phase-07-write-tests.md) | Unit tests, integration tests, build verification & CLI execution | Completed |
