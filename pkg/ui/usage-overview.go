package ui

import (
	"fmt"
	"mncode/pkg/agent"
	"mncode/pkg/stats"
	"strings"
)

// RenderUsageOverview renders Tab 1: Heatmap matrix, streaks, and token metrics
func RenderUsageOverview(s *agent.Session, timeFilterIdx int) []string {
	var lines []string
	tracker, _ := s.Tracker.(*stats.Tracker)
	if tracker == nil {
		lines = append(lines, "  No token metrics recorded yet.")
		return lines
	}

	matrix, months := tracker.GetHeatmapMatrix()
	info := tracker.GetStreakStats()

	// 1. Months header line
	monthHeader := "    "
	for _, m := range months {
		monthHeader += fmt.Sprintf("%-4s", m)
	}
	lines = append(lines, GrayText(monthHeader))

	// 2. 7 Rows (Sun - Sat)
	rowLabels := []string{"   ", "Mon", "   ", "Wed", "   ", "Fri", "   "}
	for r := 0; r < 7; r++ {
		var rowSb strings.Builder
		for c := 0; c < 52; c++ {
			ch := matrix[r][c]
			switch ch {
			case '█':
				rowSb.WriteString(BoldPastelPink(string(ch)))
			case '▓':
				rowSb.WriteString("\033[38;5;218m" + string(ch) + "\033[0m")
			case '▒':
				rowSb.WriteString("\033[38;5;225m" + string(ch) + "\033[0m")
			case '░':
				rowSb.WriteString(GrayText(string(ch)))
			default:
				rowSb.WriteString(GrayText(string(ch)))
			}
		}
		lines = append(lines, fmt.Sprintf("%-4s%s", GrayText(rowLabels[r]), rowSb.String()))
	}
	lines = append(lines, "")

	// 3. Legend
	lines = append(lines, fmt.Sprintf("    %s %s %s %s %s %s",
		GrayText("Less"), GrayText("░"), "\033[38;5;225m▒\033[0m", "\033[38;5;218m▓\033[0m", BoldPastelPink("█"), GrayText("More")))
	lines = append(lines, "")

	// 4. Timeframe filter tabs
	filters := []string{"All time", "Last 7 days", "Last 30 days"}
	var filterStrs []string
	for i, f := range filters {
		if i == timeFilterIdx {
			filterStrs = append(filterStrs, BoldPastelPink(f))
		} else {
			filterStrs = append(filterStrs, GrayText(f))
		}
	}
	lines = append(lines, strings.Join(filterStrs, " · "))
	lines = append(lines, "")

	// 5. Favorite model & Total tokens
	lines = append(lines, fmt.Sprintf("%-32s %s",
		fmt.Sprintf("Favorite model: %s", Bold(info.FavoriteModel)),
		fmt.Sprintf("Total tokens: %s", BoldCyan(stats.FormatCompactTokens(info.TotalTokens)))))
	lines = append(lines, "")

	// 6. Streaks & Sessions grid
	lines = append(lines, fmt.Sprintf("%-32s %s",
		fmt.Sprintf("Sessions: %s", Bold(fmt.Sprintf("%d", info.Sessions))),
		fmt.Sprintf("Longest session: %s", Bold(info.LongestSession))))
	lines = append(lines, fmt.Sprintf("%-32s %s",
		fmt.Sprintf("Active days: %s", Bold(fmt.Sprintf("%d/%d", info.ActiveDays, info.TotalDays))),
		fmt.Sprintf("Longest streak: %s", Bold(fmt.Sprintf("%d days", info.LongestStreak)))))
	lines = append(lines, fmt.Sprintf("%-32s %s",
		fmt.Sprintf("Most active day: %s", Bold(info.MostActiveDay)),
		fmt.Sprintf("Current streak: %s", Bold(fmt.Sprintf("%d days", info.CurrentStreak)))))
	lines = append(lines, "")

	// 7. Token breakdowns
	lines = append(lines, fmt.Sprintf("Input %s · Output %s · Cache read %s · Cache write %s",
		BoldGreen(stats.FormatCompactTokens(info.InputTokens)),
		BoldCyan(stats.FormatCompactTokens(info.OutputTokens)),
		GrayText(stats.FormatCompactTokens(info.CacheRead)),
		GrayText(stats.FormatCompactTokens(info.CacheWrite))))

	return lines
}
