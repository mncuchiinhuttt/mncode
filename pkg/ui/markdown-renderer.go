package ui

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	boldRe       = regexp.MustCompile(`\*\*(.*?)\*\*`)
	italicRe     = regexp.MustCompile(`\*([^\*]+)\*`)
	inlineCodeRe = regexp.MustCompile("`([^`]+)`")
)

// RenderMarkdown parses markdown text and returns beautiful ANSI terminal styled text
func RenderMarkdown(md string) string {
	t := GetCurrentTheme()
	lines := strings.Split(md, "\n")
	var result []string
	inCodeBlock := false
	codeLang := ""
	var codeLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 1. Code Block Fence (```lang)
		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				// End code block
				result = append(result, renderCodeBox(codeLang, codeLines, t)...)
				inCodeBlock = false
				codeLang = ""
				codeLines = nil
			} else {
				// Start code block
				inCodeBlock = true
				codeLang = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				codeLines = nil
			}
			continue
		}

		if inCodeBlock {
			codeLines = append(codeLines, line)
			continue
		}

		// 2. Horizontal Rule (---, ***, ___)
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			result = append(result, GrayText(strings.Repeat("─", 60)))
			continue
		}

		// 3. Headers (# Header, ## Header, ### Header)
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimPrefix(trimmed, "# ")
			result = append(result, Colorize(AttrBold+t.Primary, "# "+title))
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			title := strings.TrimPrefix(trimmed, "## ")
			result = append(result, Colorize(AttrBold+t.Secondary, "## "+title))
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			title := strings.TrimPrefix(trimmed, "### ")
			result = append(result, Colorize(AttrBold+t.Info, "### "+title))
			continue
		}

		// 4. Blockquotes (> quote)
		if strings.HasPrefix(trimmed, ">") {
			quoteText := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
			styledQuote := formatInlineMarkdown(quoteText, t)
			result = append(result, fmt.Sprintf("  %s %s", Colorize(t.Primary, "│"), Colorize(AttrDim, styledQuote)))
			continue
		}

		// 5. Unordered Lists (- item, * item)
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			itemText := trimmed[2:]
			styledItem := formatInlineMarkdown(itemText, t)
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			result = append(result, fmt.Sprintf("%s  %s %s", indent, Colorize(AttrBold+t.Primary, "•"), styledItem))
			continue
		}

		// 6. Ordered Lists (1. item, 2. item)
		if matched, _ := regexp.MatchString(`^\d+\.\s+`, trimmed); matched {
			parts := strings.SplitN(trimmed, " ", 2)
			num := parts[0]
			itemText := ""
			if len(parts) > 1 {
				itemText = parts[1]
			}
			styledItem := formatInlineMarkdown(itemText, t)
			result = append(result, fmt.Sprintf("  %s %s", Colorize(AttrBold+t.Secondary, num), styledItem))
			continue
		}

		// 7. Normal paragraph with inline formatting
		result = append(result, formatInlineMarkdown(line, t))
	}

	if inCodeBlock {
		result = append(result, renderCodeBox(codeLang, codeLines, t)...)
	}

	return strings.Join(result, "\n")
}

func formatInlineMarkdown(s string, t Theme) string {
	// Inline Code: `code`
	s = inlineCodeRe.ReplaceAllStringFunc(s, func(m string) string {
		inner := m[1 : len(m)-1]
		return fmt.Sprintf("\033[48;5;236m\033[38;5;218m %s \033[0m", inner)
	})

	// Bold: **text**
	s = boldRe.ReplaceAllStringFunc(s, func(m string) string {
		inner := m[2 : len(m)-2]
		return Colorize(AttrBold+t.Text, inner)
	})

	// Italic: *text*
	s = italicRe.ReplaceAllStringFunc(s, func(m string) string {
		inner := m[1 : len(m)-1]
		return Colorize(AttrDim+t.Primary, inner)
	})

	return s
}

func renderCodeBox(lang string, codeLines []string, t Theme) []string {
	if lang == "" {
		lang = "code"
	}
	maxLen := 50
	for _, l := range codeLines {
		if len([]rune(l)) > maxLen {
			maxLen = len([]rune(l))
		}
	}
	if maxLen > 80 {
		maxLen = 80
	}

	topBorder := fmt.Sprintf("  %s %s %s",
		Colorize(t.Muted, "╭──"),
		Colorize(AttrBold+t.Primary, lang),
		Colorize(t.Muted, strings.Repeat("─", maxLen-len(lang))))

	var res []string
	res = append(res, topBorder)
	for _, l := range codeLines {
		res = append(res, fmt.Sprintf("  %s %s", Colorize(t.Muted, "│"), Colorize(t.Text, l)))
	}
	res = append(res, fmt.Sprintf("  %s", Colorize(t.Muted, "╰"+strings.Repeat("─", maxLen+3))))
	return res
}
