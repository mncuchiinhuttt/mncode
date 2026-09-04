package artifacts

import (
	"fmt"
	"strings"
)

const (
	// MaxInlineLines is the threshold above which tool outputs are truncated.
	MaxInlineLines = 80
	// MaxInlineBytes is the byte length threshold (12 KB).
	MaxInlineBytes = 12 * 1024
	headRetainLines = 40
	tailRetainLines = 30
)

// TruncateOutput checks if an output exceeds thresholds. If it does, the full content
// is saved to the artifact store and a truncated preview with an artifact:// URI is returned.
func TruncateOutput(output string, store *Store) string {
	if store == nil {
		store = GlobalStore()
	}

	byteLen := len(output)
	lines := strings.Split(output, "\n")
	lineCount := len(lines)

	if lineCount <= MaxInlineLines && byteLen <= MaxInlineBytes {
		return output
	}

	id, err := store.Save(output)
	if err != nil {
		// If saving fails, fall back to simple string trimming
		if lineCount > MaxInlineLines {
			head := strings.Join(lines[:headRetainLines], "\n")
			tail := strings.Join(lines[lineCount-tailRetainLines:], "\n")
			return fmt.Sprintf("%s\n\n[... %d lines truncated ...]\n\n%s", head, lineCount-headRetainLines-tailRetainLines, tail)
		}
		return output
	}

	var head, tail string
	var elidedLines int

	if lineCount > MaxInlineLines {
		head = strings.Join(lines[:headRetainLines], "\n")
		tail = strings.Join(lines[lineCount-tailRetainLines:], "\n")
		elidedLines = lineCount - headRetainLines - tailRetainLines
	} else {
		// Large in bytes but fewer than 80 lines (e.g. huge single line JSON)
		head = output[:4096]
		tail = output[byteLen-2048:]
		elidedLines = 0
	}

	sizeStr := formatBytes(byteLen)
	var sb strings.Builder
	sb.WriteString(head)
	sb.WriteString("\n\n")
	if elidedLines > 0 {
		sb.WriteString(fmt.Sprintf("[... %d lines elided ...]\n\n", elidedLines))
	}
	sb.WriteString(tail)
	sb.WriteString(fmt.Sprintf("\n\n[Output truncated (%d lines, %s). Full output saved to artifact://%s]\n", lineCount, sizeStr, id))
	sb.WriteString(fmt.Sprintf("[Use 'read artifact://%s:1-100' or ':raw' to inspect on demand without polluting context]", id))

	return sb.String()
}

func formatBytes(b int) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f KB", float64(b)/float64(div))
}
