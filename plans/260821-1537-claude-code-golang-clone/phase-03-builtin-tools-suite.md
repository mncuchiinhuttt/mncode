# Phase 03: Built-in Tools Suite

## Context Links
- [plan.md](file:///Users/vominhlong/mncode/plans/260821-1537-claude-code-golang-clone/plan.md)

## Overview
- **Priority**: High
- **Current status**: Pending
- **Description**: Recreate the complete toolset of Claude Code in pure, high-performance Go with safety checks and clean schemas for LLM tool calling.

## Tools to Implement
1. `bash` (`run_command`): Execute command with timeout, working directory, streaming output.
2. `view_file` (`read_file`): Read file with line numbers, range slicing (`start_line`, `end_line`), binary detection.
3. `replace_file_content` (`edit_file`): Precise substring replacement with validation.
4. `write_to_file` (`write_file`): Create or overwrite files, auto-create parent directories.
5. `grep_search` (`grep`): Fast regex/literal search across files with line numbers.
6. `find_by_name` (`glob` / `find`): File name/glob pattern search with depth and ignore filters.
7. `list_dir` (`ls`): Directory listing with file metadata.
8. `fetch_url` (`read_url`): HTTP request with markdown extraction.
9. `ask_user`: Interactive prompts to clarify questions with user.

## Related Code Files
- Create: `pkg/tools/tool.go` (interfaces and registry)
- Create: `pkg/tools/bash-tool.go`
- Create: `pkg/tools/view-tool.go`
- Create: `pkg/tools/edit-tool.go`
- Create: `pkg/tools/write-tool.go`
- Create: `pkg/tools/grep-tool.go`
- Create: `pkg/tools/find-tool.go`
- Create: `pkg/tools/list-tool.go`
- Create: `pkg/tools/web-tool.go`
- Create: `pkg/tools/ask-tool.go`
