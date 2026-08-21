package ui

import (
	"mncode/pkg/agent"
	"sort"
	"strings"
)

type scoredSlashOption struct {
	opt   SlashOption
	score int
}

func getMatchingSlashOptions(s *agent.Session, query string) []SlashOption {
	q := strings.ToLower(strings.TrimPrefix(query, "/"))
	var candidates []SlashOption
	seen := make(map[string]bool)

	for _, opt := range slashOptions {
		if !seen[opt.Command] {
			seen[opt.Command] = true
			candidates = append(candidates, opt)
		}
	}

	if s != nil && s.Catalog != nil {
		for _, sk := range s.Catalog.Skills {
			baseName := strings.TrimPrefix(sk.Name, "ck:")
			cmd := "/ck:" + baseName
			if !seen[cmd] {
				seen[cmd] = true
				candidates = append(candidates, SlashOption{
					Command:     cmd,
					Description: sk.Description,
					Category:    "Skills",
				})
			}
		}
	}

	if q == "" {
		return candidates
	}

	var scored []scoredSlashOption
	for _, opt := range candidates {
		cmdName := strings.ToLower(strings.TrimPrefix(opt.Command, "/"))
		desc := strings.ToLower(opt.Description)
		score := 0

		if cmdName == q || cmdName == "ck:"+q {
			score = 1000
		} else if strings.HasPrefix(cmdName, q) {
			score = 500 + (50 - len(cmdName))
		} else if strings.HasPrefix(cmdName, "ck:"+q) {
			score = 450 + (50 - len(cmdName))
		} else if strings.Contains(cmdName, q) {
			score = 200 + (50 - len(cmdName))
		} else if strings.HasPrefix(desc, q) {
			score = 50
		} else if strings.Contains(desc, q) {
			score = 10
		}

		if score > 0 {
			scored = append(scored, scoredSlashOption{opt: opt, score: score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].opt.Command < scored[j].opt.Command
	})

	var result []SlashOption
	for _, sc := range scored {
		result = append(result, sc.opt)
	}
	return result
}
