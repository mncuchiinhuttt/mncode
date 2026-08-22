package ui

import (
	"encoding/json"
	"fmt"
	"mncode/pkg/provider"
	"strings"
)

// RenderToolCallFormatted prints a clean syntax for tool calls
func RenderToolCallFormatted(tc *provider.ToolCall) string {
	t := GetCurrentTheme()
	argBytes, _ := json.Marshal(tc.Arguments)

	switch tc.Name {
	case "run_command", "bash":
		var args struct {
			CommandLine string `json:"CommandLine"`
			Cwd         string `json:"Cwd"`
		}
		_ = json.Unmarshal(argBytes, &args)
		cmd := strings.TrimSpace(args.CommandLine)
		if cmd == "" {
			cmd = string(argBytes)
		}
		return fmt.Sprintf("\n%s%s%s\n",
			Colorize(AttrBold+t.Primary, "Bash("),
			Colorize(AttrBold+t.Text, cmd),
			Colorize(AttrBold+t.Primary, ")"))

	case "view_file", "read_file":
		var args struct {
			AbsolutePath string `json:"AbsolutePath"`
			StartLine    int    `json:"StartLine"`
			EndLine      int    `json:"EndLine"`
		}
		_ = json.Unmarshal(argBytes, &args)
		target := args.AbsolutePath
		if target == "" {
			target = formatCleanArgs(tc.Arguments)
		}
		lineInfo := ""
		if args.StartLine > 0 && args.EndLine > 0 {
			lineInfo = fmt.Sprintf(":%d-%d", args.StartLine, args.EndLine)
		}
		return fmt.Sprintf("\n%s%s%s\n",
			Colorize(AttrBold+t.Primary, tc.Name+"("),
			Colorize(AttrBold+t.Text, target+lineInfo),
			Colorize(AttrBold+t.Primary, ")"))

	case "replace_file_content", "edit_file":
		var args struct {
			TargetFile  string `json:"TargetFile"`
			Instruction string `json:"Instruction"`
			StartLine   int    `json:"StartLine"`
			EndLine     int    `json:"EndLine"`
		}
		_ = json.Unmarshal(argBytes, &args)
		lineRange := ""
		if args.StartLine > 0 && args.EndLine > 0 {
			lineRange = fmt.Sprintf(":%d-%d", args.StartLine, args.EndLine)
		}
		return fmt.Sprintf("\n%s%s%s\n",
			Colorize(AttrBold+t.Primary, tc.Name+"("),
			Colorize(AttrBold+t.Text, args.TargetFile+lineRange),
			Colorize(AttrBold+t.Primary, ")"))

	case "write_to_file", "create_file":
		var args struct {
			TargetFile  string `json:"TargetFile"`
			Description string `json:"Description"`
		}
		_ = json.Unmarshal(argBytes, &args)
		target := args.TargetFile
		if target == "" {
			target = formatCleanArgs(tc.Arguments)
		}
		return fmt.Sprintf("\n%s%s%s\n",
			Colorize(AttrBold+t.Success, tc.Name+"("),
			Colorize(AttrBold+t.Text, target),
			Colorize(AttrBold+t.Success, ")"))

	case "list_dir":
		var args struct {
			DirectoryPath string `json:"DirectoryPath"`
		}
		_ = json.Unmarshal(argBytes, &args)
		path := args.DirectoryPath
		if path == "" {
			path = "."
		}
		return fmt.Sprintf("\n%s%s%s\n",
			Colorize(AttrBold+t.Primary, "list_dir("),
			Colorize(AttrBold+t.Text, path),
			Colorize(AttrBold+t.Primary, ")"))

	case "grep_search":
		var args struct {
			Query      string `json:"Query"`
			SearchPath string `json:"SearchPath"`
		}
		_ = json.Unmarshal(argBytes, &args)
		return fmt.Sprintf("\n%s%s, in=%s%s\n",
			Colorize(AttrBold+t.Warning, "grep_search("),
			Colorize(AttrBold+t.Text, fmt.Sprintf("%q", args.Query)),
			Colorize(t.Secondary, args.SearchPath),
			Colorize(AttrBold+t.Warning, ")"))

	case "find_by_name":
		var args struct {
			Pattern         string `json:"Pattern"`
			SearchDirectory string `json:"SearchDirectory"`
		}
		_ = json.Unmarshal(argBytes, &args)
		return fmt.Sprintf("\n%s%s, in=%s%s\n",
			Colorize(AttrBold+t.Warning, "find_by_name("),
			Colorize(AttrBold+t.Text, fmt.Sprintf("%q", args.Pattern)),
			Colorize(t.Secondary, args.SearchDirectory),
			Colorize(AttrBold+t.Warning, ")"))

	default:
		argsFormatted := formatCleanArgs(tc.Arguments)
		return fmt.Sprintf("\n%s%s%s\n",
			Colorize(AttrBold+t.Primary, tc.Name+"("),
			Colorize(t.Muted, argsFormatted),
			Colorize(AttrBold+t.Primary, ")"))
	}
}

func formatCleanArgs(args map[string]interface{}) string {
	if len(args) == 0 {
		return ""
	}
	var pairs []string
	for k, v := range args {
		switch val := v.(type) {
		case string:
			if len([]rune(val)) > 60 {
				val = string([]rune(val)[:57]) + "…"
			}
			pairs = append(pairs, fmt.Sprintf("%s=%q", k, val))
		default:
			b, _ := json.Marshal(val)
			pairs = append(pairs, fmt.Sprintf("%s=%s", k, string(b)))
		}
	}
	return strings.Join(pairs, ", ")
}

// RenderToolResultFormatted prints highlighted diffs, stdout, or search previews
func RenderToolResultFormatted(name string, result string, isError bool) string {
	t := GetCurrentTheme()
	clean := strings.TrimSpace(result)
	if clean == "" {
		if isError {
			return fmt.Sprintf("%s %s (empty error output)\n", Colorize(AttrBold+t.Error, "[Tool Error]"), name)
		}
		return fmt.Sprintf("%s %s (success)\n", Colorize(AttrBold+t.Success, "[Tool Done]"), name)
	}

	// 1. If Code Diff (contains + / - or diff headers)
	if strings.Contains(clean, "[diff_block_start]") || strings.Contains(clean, "@@ ") || (strings.Contains(clean, "+") && strings.Contains(clean, "-")) {
		return formatDiffBlock(clean, t, true)
	}

	// 2. Standard Output folding
	lines := strings.Split(clean, "\n")
	previewLines := lines
	hasFolded := false
	if len(lines) > 8 {
		previewLines = lines[:8]
		hasFolded = true
	}

	var sb strings.Builder
	badge := Colorize(AttrBold+t.Success, "  [OK]")
	if isError {
		badge = Colorize(AttrBold+t.Error, "  [Error]")
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", badge, Colorize(t.Muted, name)))

	for _, l := range previewLines {
		sb.WriteString(fmt.Sprintf("    %s %s\n", Colorize(t.Muted, "│"), Colorize(t.Text, l)))
	}
	if hasFolded {
		sb.WriteString(fmt.Sprintf("    %s %s\n", Colorize(t.Muted, "└"), Colorize(t.Muted, fmt.Sprintf("... [%d more lines folded]", len(lines)-8))))
	}
	return sb.String()
}

func formatDiffBlock(diff string, t Theme, fullLineBg bool) string {
	lines := strings.Split(diff, "\n")
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s %s\n", Colorize(AttrBold+t.Primary, "▼"), Colorize(AttrBold+t.Text, "Code Diff & Edits:")))

	maxLines := 15
	shownCount := 0

	for _, l := range lines {
		if strings.Contains(l, "[diff_block_start]") || strings.Contains(l, "[diff_block_end]") {
			continue
		}
		if shownCount >= maxLines {
			sb.WriteString(fmt.Sprintf("    %s %s\n", Colorize(t.Muted, "└"), Colorize(t.Muted, fmt.Sprintf("... [%d more diff lines]", len(lines)-shownCount))))
			break
		}

		if strings.HasPrefix(l, "+") {
			content := l[1:]
			if fullLineBg && t.DiffAddBg != "" {
				pad := 60 - len([]rune(content))
				if pad < 2 {
					pad = 2
				}
				sb.WriteString(fmt.Sprintf("    %s\n", Colorize(t.DiffAddBg+t.DiffAddFg, fmt.Sprintf(" + %s%s", content, strings.Repeat(" ", pad)))))
			} else {
				sb.WriteString(fmt.Sprintf("    %s %s\n", Colorize(t.Success, "+"), Colorize(t.Success, content)))
			}
		} else if strings.HasPrefix(l, "-") {
			content := l[1:]
			if fullLineBg && t.DiffDelBg != "" {
				pad := 60 - len([]rune(content))
				if pad < 2 {
					pad = 2
				}
				sb.WriteString(fmt.Sprintf("    %s\n", Colorize(t.DiffDelBg+t.DiffDelFg, fmt.Sprintf(" - %s%s", content, strings.Repeat(" ", pad)))))
			} else {
				sb.WriteString(fmt.Sprintf("    %s %s\n", Colorize(t.Error, "-"), Colorize(t.Error, content)))
			}
		} else if strings.HasPrefix(l, "@@") {
			sb.WriteString(fmt.Sprintf("    %s %s\n", Colorize(t.Info, "≈"), Colorize(t.Info, l)))
		} else {
			sb.WriteString(fmt.Sprintf("    %s %s\n", Colorize(t.Muted, " "), Colorize(t.Muted, l)))
		}
		shownCount++
	}
	return sb.String()
}
