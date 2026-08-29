package memory

import (
	"fmt"
	"strings"
)

// ReflectOnFailure parses a failed tool execution and subsequent fix to synthesize an actionable lesson.
func ReflectOnFailure(toolName, inputOrCmd, errorOutput, fixSummary string) ReflectiveLesson {
	category := CategoryGotchaBug
	topic := "tool-gotcha"

	lowerErr := strings.ToLower(errorOutput)
	lowerInput := strings.ToLower(inputOrCmd)

	if strings.Contains(lowerErr, "test") || strings.Contains(lowerInput, "test") {
		category = CategoryGotchaBug
		topic = "test-regression"
	} else if strings.Contains(lowerErr, "compile") || strings.Contains(lowerErr, "build") || strings.Contains(lowerErr, "syntax") {
		category = CategoryToolchain
		topic = "build-compile"
	} else if strings.Contains(lowerErr, "permission") || strings.Contains(lowerErr, "access denied") {
		category = CategoryToolchain
		topic = "permissions"
	} else if strings.Contains(lowerErr, "429") || strings.Contains(lowerErr, "quota") || strings.Contains(lowerErr, "rate limit") {
		category = CategoryToolchain
		topic = "rate-limits"
	} else if strings.Contains(lowerErr, "merge conflict") || strings.Contains(lowerInput, "git") {
		category = CategoryConvention
		topic = "git-workflow"
	}

	mistake := strings.TrimSpace(errorOutput)
	if len(mistake) > 160 {
		mistake = mistake[:157] + "..."
	}

	correction := strings.TrimSpace(fixSummary)
	if correction == "" {
		correction = fmt.Sprintf("Apply verified fix for %s when encountering: %s", toolName, mistake)
	}

	summary := fmt.Sprintf("[%s] %s: %s", strings.ToUpper(string(category)), topic, correction)

	return ReflectiveLesson{
		Topic:      topic,
		Category:   category,
		Summary:    summary,
		Mistake:    mistake,
		Correction: correction,
		Confidence: 5,
		Source:     "auto-reflection",
	}
}

// AutoReflectAndLearn analyzes an error-correction sequence and saves it into the shared memory store.
func AutoReflectAndLearn(store *HierarchicalStore, toolName, inputOrCmd, errorOutput, fixSummary string, tier MemoryTier) (*MemoryItem, bool, error) {
	if store == nil {
		return nil, false, nil
	}
	lesson := ReflectOnFailure(toolName, inputOrCmd, errorOutput, fixSummary)
	return EvolveMemory(store, lesson, tier)
}
