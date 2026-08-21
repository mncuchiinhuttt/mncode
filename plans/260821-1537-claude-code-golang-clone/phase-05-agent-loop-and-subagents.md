# Phase 05: Agent Loop and Subagents

## Context Links
- [plan.md](file:///Users/vominhlong/mncode/plans/260821-1537-claude-code-golang-clone/plan.md)

## Overview
- **Priority**: High
- **Current status**: Pending
- **Description**: Build the ReAct agent execution loop, system prompt composer (injecting skills catalog, active rules, workspace info), context window manager, and subagent orchestration (planner, researcher, reviewer, etc.).

## Requirements
- System prompt assembly including environment metadata, rules, available skills, and tool usage guidance.
- Iterative tool execution loop: LLM response -> Parse tool calls -> Execute tools -> Feed results back -> Loop until completion.
- Subagent runner capable of executing dedicated roles in sub-contexts.
- Token budget management & conversation compaction.

## Related Code Files
- Create: `pkg/agent/types.go`
- Create: `pkg/agent/prompt-builder.go`
- Create: `pkg/agent/engine.go`
- Create: `pkg/agent/subagent-runner.go`
- Create: `pkg/agent/history.go`
