# Phase 02: Claude Skills & ClaudeKit Agent System

## Context Links
- [plan.md](file:///Users/vominhlong/mncode/plans/260821-1537-claude-code-golang-clone/plan.md)
- [agent_skills_spec.md](file:///Users/vominhlong/mncode/.claude/skills/agent_skills_spec.md)

## Overview
- **Priority**: High
- **Current status**: Pending
- **Description**: Implement loader and parser for Open Agent Skills Specification (`SKILL.md` in `.claude/skills/`), ClaudeKit Agents (`.claude/agents/*.md`), and Claude rules (`.claude/rules/*.md`).

## Requirements
- Parse YAML frontmatter + Markdown body from `.claude/skills/**/SKILL.md`.
- Extract skill metadata: `name`, `description`, `allowed-tools`, `metadata`.
- Parse ClaudeKit agents from `.claude/agents/*.md`.
- Parse rules from `.claude/rules/*.md`.
- Format skills catalog and active skills into system prompt context efficiently.

## Related Code Files
- Create: `pkg/skills/types.go`
- Create: `pkg/skills/frontmatter.go`
- Create: `pkg/skills/skill-loader.go`
- Create: `pkg/skills/agent-loader.go`
- Create: `pkg/skills/rule-loader.go`
- Create: `pkg/skills/catalog.go`

## Success Criteria
- Accurately discover and load all 60+ skills from `.claude/skills/` and all agents from `.claude/agents/`.
