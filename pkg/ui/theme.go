package ui

import (
	"fmt"
	"strings"
)

// Theme represents a complete color palette for UI, Code Diffs, and Tool Executions
type Theme struct {
	ID          string
	Name        string
	Description string
	IsDark      bool
	Primary     string // Main accent (borders, prompt, active tabs)
	Secondary   string // Secondary accent (badges, highlights)
	Success     string // Tool OK, diff additions (+)
	Error       string // Tool error, diff deletions (-)
	Warning     string // Quota, caution, budget
	Info        string // Information, links, thinking
	Muted       string // Borders, line numbers, subtitles
	Text        string // Standard text
	DiffAddBg   string // Full line background for additions (+)
	DiffDelBg   string // Full line background for deletions (-)
	DiffAddFg   string // Text color on addition bg
	DiffDelFg   string // Text color on deletion bg
}

var Themes = map[string]Theme{
	"pastel-pink": {
		ID:          "pastel-pink",
		Name:        "Pastel Pink Dark",
		Description: "Soft aesthetic pastel pink & mint accents for dark terminal",
		IsDark:      true,
		Primary:     "\033[38;5;218m",
		Secondary:   "\033[38;5;212m",
		Success:     "\033[38;5;120m",
		Error:       "\033[38;5;203m",
		Warning:     "\033[38;5;222m",
		Info:        "\033[38;5;225m",
		Muted:       "\033[38;5;243m",
		Text:        "\033[38;5;254m",
		DiffAddBg:   "\033[48;5;22m",
		DiffAddFg:   "\033[1;38;5;120m",
		DiffDelBg:   "\033[48;5;52m",
		DiffDelFg:   "\033[1;38;5;218m",
	},
	"pastel-pink-light": {
		ID:          "pastel-pink-light",
		Name:        "Pastel Pink Light",
		Description: "Aesthetic pastel blush & soft green for light terminal",
		IsDark:      false,
		Primary:     "\033[38;5;168m",
		Secondary:   "\033[38;5;198m",
		Success:     "\033[38;5;28m",
		Error:       "\033[38;5;161m",
		Warning:     "\033[38;5;130m",
		Info:        "\033[38;5;126m",
		Muted:       "\033[38;5;244m",
		Text:        "\033[38;5;235m",
		DiffAddBg:   "\033[48;5;194m",
		DiffAddFg:   "\033[1;38;5;28m",
		DiffDelBg:   "\033[48;5;225m",
		DiffDelFg:   "\033[1;38;5;161m",
	},
	"dark": {
		ID:          "dark",
		Name:        "Claude Dark Mode",
		Description: "Classic Claude Code dark theme with vivid cyan and emerald",
		IsDark:      true,
		Primary:     "\033[38;5;51m",
		Secondary:   "\033[38;5;39m",
		Success:     "\033[38;5;48m",
		Error:       "\033[38;5;196m",
		Warning:     "\033[38;5;220m",
		Info:        "\033[38;5;75m",
		Muted:       "\033[38;5;240m",
		Text:        "\033[38;5;253m",
		DiffAddBg:   "\033[48;5;22m",
		DiffAddFg:   "\033[1;38;5;48m",
		DiffDelBg:   "\033[48;5;52m",
		DiffDelFg:   "\033[1;38;5;203m",
	},
	"light": {
		ID:          "light",
		Name:        "Solarized Light",
		Description: "Clean, high-contrast light theme with navy and crimson",
		IsDark:      false,
		Primary:     "\033[38;5;24m",
		Secondary:   "\033[38;5;31m",
		Success:     "\033[38;5;28m",
		Error:       "\033[38;5;160m",
		Warning:     "\033[38;5;130m",
		Info:        "\033[38;5;25m",
		Muted:       "\033[38;5;244m",
		Text:        "\033[38;5;235m",
		DiffAddBg:   "\033[48;5;194m",
		DiffAddFg:   "\033[1;38;5;28m",
		DiffDelBg:   "\033[48;5;224m",
		DiffDelFg:   "\033[1;38;5;160m",
	},
	"cyberpunk": {
		ID:          "cyberpunk",
		Name:        "Cyberpunk Neon",
		Description: "High-voltage neon magenta, electric cyan, and acid yellow",
		IsDark:      true,
		Primary:     "\033[38;5;201m",
		Secondary:   "\033[38;5;51m",
		Success:     "\033[38;5;46m",
		Error:       "\033[38;5;197m",
		Warning:     "\033[38;5;226m",
		Info:        "\033[38;5;141m",
		Muted:       "\033[38;5;239m",
		Text:        "\033[38;5;255m",
		DiffAddBg:   "\033[48;5;22m",
		DiffAddFg:   "\033[1;38;5;46m",
		DiffDelBg:   "\033[48;5;53m",
		DiffDelFg:   "\033[1;38;5;201m",
	},
	"monokai": {
		ID:          "monokai",
		Name:        "Monokai Pro",
		Description: "Vibrant yellow, candy green, hot magenta, and soft orange",
		IsDark:      true,
		Primary:     "\033[38;5;227m",
		Secondary:   "\033[38;5;197m",
		Success:     "\033[38;5;148m",
		Error:       "\033[38;5;197m",
		Warning:     "\033[38;5;208m",
		Info:        "\033[38;5;81m",
		Muted:       "\033[38;5;241m",
		Text:        "\033[38;5;252m",
		DiffAddBg:   "\033[48;5;22m",
		DiffAddFg:   "\033[1;38;5;148m",
		DiffDelBg:   "\033[48;5;52m",
		DiffDelFg:   "\033[1;38;5;197m",
	},
	"tokyo-night": {
		ID:          "tokyo-night",
		Name:        "Tokyo Night",
		Description: "Deep twilight indigo, lavender rose, and luminous sky blue",
		IsDark:      true,
		Primary:     "\033[38;5;111m",
		Secondary:   "\033[38;5;176m",
		Success:     "\033[38;5;114m",
		Error:       "\033[38;5;204m",
		Warning:     "\033[38;5;221m",
		Info:        "\033[38;5;153m",
		Muted:       "\033[38;5;60m",
		Text:        "\033[38;5;189m",
		DiffAddBg:   "\033[48;5;23m",
		DiffAddFg:   "\033[1;38;5;114m",
		DiffDelBg:   "\033[48;5;52m",
		DiffDelFg:   "\033[1;38;5;204m",
	},
}

var ThemeList = []string{"pastel-pink", "pastel-pink-light", "dark", "light", "cyberpunk", "monokai", "tokyo-night"}

// GetThemeCatalog exposes theme metadata to other clients such as the desktop app.
func GetThemeCatalog() []Theme {
	result := make([]Theme, 0, len(ThemeList))
	for _, id := range ThemeList {
		if theme, ok := Themes[id]; ok {
			result = append(result, theme)
		}
	}
	return result
}

var currentTheme = Themes["pastel-pink"]

func GetCurrentTheme() Theme {
	return currentTheme
}

func SetTheme(themeID string) Theme {
	id := strings.ToLower(strings.TrimSpace(themeID))
	id = strings.ReplaceAll(id, " ", "-")
	if t, ok := Themes[id]; ok {
		currentTheme = t
		return t
	}
	for k, t := range Themes {
		if strings.Contains(k, id) || strings.Contains(strings.ToLower(t.Name), id) {
			currentTheme = t
			return t
		}
	}
	currentTheme = Themes["pastel-pink"]
	return currentTheme
}

func RenderThemeSwatch(t Theme) string {
	return fmt.Sprintf("%s██%s██%s██%s██%s██%s",
		t.Primary, t.Secondary, t.Success, t.Warning, t.Error, Reset)
}
