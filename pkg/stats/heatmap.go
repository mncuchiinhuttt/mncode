package stats

import (
	"fmt"
	"sort"
	"time"
)

type StreakInfo struct {
	TotalTokens    int64
	InputTokens    int64
	OutputTokens   int64
	ThinkingTokens int64
	CacheRead      int64
	CacheWrite     int64
	Sessions       int
	LongestSession string
	ActiveDays     int
	TotalDays      int
	CurrentStreak  int
	LongestStreak  int
	MostActiveDay  string
	FavoriteModel  string
}

// GetStreakStats computes streaks, active days, favorite model, and token totals
// from recorded usage only. Unknown values remain zero/empty instead of being
// presented as estimates.
func (t *Tracker) GetStreakStats() StreakInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	info := StreakInfo{}
	if t.store.Lifetime != nil {
		info.TotalTokens = t.store.Lifetime.TotalTokens
		info.InputTokens = t.store.Lifetime.InputTokens
		info.OutputTokens = t.store.Lifetime.OutputTokens
		info.ThinkingTokens = t.store.Lifetime.ThinkingTokens
	}

	var maxModelTokens int64
	for model, summary := range t.store.ByModel {
		if summary != nil && (summary.TotalTokens > maxModelTokens ||
			summary.TotalTokens == maxModelTokens && (info.FavoriteModel == "" || model < info.FavoriteModel)) {
			maxModelTokens = summary.TotalTokens
			info.FavoriteModel = model
		}
	}

	active := make(map[string]bool, len(t.store.Daily))
	for date, summary := range t.store.Daily {
		if summary != nil && summary.TotalTokens > 0 {
			active[date] = true
		}
	}
	info.ActiveDays = len(active)
	info.TotalDays = len(active)

	now := time.Now().UTC()
	current := 0
	for d := 0; ; d++ {
		date := now.AddDate(0, 0, -d).Format("2006-01-02")
		if !active[date] {
			break
		}
		current++
	}
	maxStreak := 0
	streak := 0
	var mostActiveDate string
	var mostActiveTokens int64
	dateKeys := make([]string, 0, len(active))
	for date := range active {
		dateKeys = append(dateKeys, date)
	}
	sort.Strings(dateKeys)
	var previous time.Time
	for _, date := range dateKeys {
		parsed, err := time.ParseInLocation("2006-01-02", date, time.UTC)
		if err != nil {
			continue
		}
		if !previous.IsZero() && parsed.Sub(previous) == 24*time.Hour {
			streak++
		} else {
			streak = 1
		}
		if streak > maxStreak {
			maxStreak = streak
		}
		previous = parsed
		if summary := t.store.Daily[date]; summary != nil &&
			(summary.TotalTokens > mostActiveTokens ||
				summary.TotalTokens == mostActiveTokens && (mostActiveDate == "" || date < mostActiveDate)) {
			mostActiveTokens = summary.TotalTokens
			mostActiveDate = date
		}
	}
	info.LongestStreak = maxStreak
	if mostActiveDate != "" {
		if parsed, err := time.ParseInLocation("2006-01-02", mostActiveDate, time.UTC); err == nil {
			info.MostActiveDay = parsed.Format("Jan 02")
		}
	}

	// History entries represent recorded requests; do not invent a session
	// when no record exists. Duration is unavailable in the token schema.
	info.Sessions = len(t.store.History)
	info.LongestSession = ""
	return info
}

// GetHeatmapMatrix returns a 7x52 matrix of rune shades (·, ░, ▒, ▓, █)
func (t *Tracker) GetHeatmapMatrix() ([7][52]rune, []string) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var matrix [7][52]rune
	now := time.Now().UTC()
	weekday := int(now.Weekday()) // 0 = Sun, 1 = Mon ... 6 = Sat

	// Generate month labels
	monthLabels := make([]string, 12)
	for m := 11; m >= 0; m-- {
		monthLabels[11-m] = now.AddDate(0, -m, 0).Format("Jan")
	}

	for col := 51; col >= 0; col-- {
		for row := 6; row >= 0; row-- {
			daysAgo := (51-col)*7 + (weekday - row)
			if daysAgo < 0 {
				matrix[row][col] = ' '
				continue
			}
			dateKey := now.AddDate(0, 0, -daysAgo).Format("2006-01-02")
			if sum, ok := t.store.Daily[dateKey]; ok && sum != nil && sum.TotalTokens > 0 {
				if sum.TotalTokens > 500000 {
					matrix[row][col] = '█'
				} else if sum.TotalTokens > 200000 {
					matrix[row][col] = '▓'
				} else if sum.TotalTokens > 50000 {
					matrix[row][col] = '▒'
				} else {
					matrix[row][col] = '░'
				}
			} else {
				matrix[row][col] = '·'
			}
		}
	}

	return matrix, monthLabels
}

// FormatCompactTokens formats token counts to 12.3k, 715.4M, 4.3B
func FormatCompactTokens(tokens int64) string {
	if tokens >= 1_000_000_000 {
		return fmt.Sprintf("%.1fb", float64(tokens)/1_000_000_000.0)
	} else if tokens >= 1_000_000 {
		return fmt.Sprintf("%.1fm", float64(tokens)/1_000_000.0)
	} else if tokens >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(tokens)/1_000.0)
	}
	return fmt.Sprintf("%d", tokens)
}

// DailyTokenPoint holds date and total tokens
type DailyTokenPoint struct {
	DateKey string
	Label   string
	Tokens  int64
}

func (t *Tracker) GetDailyHistory(days int) []DailyTokenPoint {
	if days <= 0 {
		return []DailyTokenPoint{}
	}
	points := make([]DailyTokenPoint, days)
	now := time.Now().UTC()
	for i := 0; i < days; i++ {
		d := now.AddDate(0, 0, -(days - 1 - i))
		key := d.Format("2006-01-02")
		tokens := int64(0)
		if sum, ok := t.store.Daily[key]; ok && sum != nil {
			tokens = sum.TotalTokens
		}
		points[i] = DailyTokenPoint{
			DateKey: key,
			Label:   d.Format("Jan 02"),
			Tokens:  tokens,
		}
	}
	return points
}
