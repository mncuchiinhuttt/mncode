# Phase 04: LLM Provider Engine

## Context Links
- [plan.md](file:///Users/vominhlong/mncode/plans/260821-1537-claude-code-golang-clone/plan.md)

## Overview
- **Priority**: High
- **Current status**: Pending
- **Description**: Implement multi-model LLM abstraction supporting Anthropic Claude (Messages API with streaming, tool calling, and thinking support), OpenAI / OpenRouter compatible API, and Gemini API.

## Requirements
- Streaming token response with callbacks.
- Tool definition conversion and tool call streaming / parsing.
- Support for thinking/reasoning blocks (Claude 3.7 Sonnet extended thinking).
- Automatic retries on rate limits with exponential backoff.

## Related Code Files
- Create: `pkg/provider/provider.go`
- Create: `pkg/provider/types.go`
- Create: `pkg/provider/anthropic.go`
- Create: `pkg/provider/openai.go`
- Create: `pkg/provider/gemini.go`
- Create: `pkg/provider/factory.go`
