package memory

import "time"

// MemoryCategory classifies the type of knowledge learned by the agent.
type MemoryCategory string

const (
	CategoryConvention     MemoryCategory = "convention"      // Repository conventions and coding style
	CategoryArchitecture   MemoryCategory = "architecture"    // System components and data flow
	CategoryGotchaBug      MemoryCategory = "gotcha_bug"      // Subtle pitfalls, build errors & bug lessons
	CategoryUserPreference MemoryCategory = "user_preference" // Personal preferences and instructions
	CategoryToolchain      MemoryCategory = "toolchain"       // Flags, test scripts, ports & environment
)

// MemoryTier defines the sharing scope of the memory.
type MemoryTier string

const (
	TierGlobal    MemoryTier = "global"    // Machine-wide personal knowledge (~/.mncode/memory/global.json)
	TierWorkspace MemoryTier = "workspace" // Shared across all sessions in this repo (<workspace>/.mncode/memory/workspace.json)
)

// MemoryItem represents a structured, self-reflective unit of knowledge.
type MemoryItem struct {
	ID           string         `json:"id"`
	Topic        string         `json:"topic"`                  // Normalized slug (e.g. "auth-token-prefix")
	Category     MemoryCategory `json:"category"`
	Tier         MemoryTier     `json:"tier"`
	Summary      string         `json:"summary"`
	Mistake      string         `json:"mistake,omitempty"`      // What was tried and failed
	Correction   string         `json:"correction,omitempty"`   // The verified working approach
	Source       string         `json:"source,omitempty"`       // e.g. "auto-reflection", "user", "session"
	Confidence   int            `json:"confidence"`            // 1 to 5 rating
	HitCount     int            `json:"hitCount"`              // Times recalled
	SupersedesID string         `json:"supersedesId,omitempty"` // ID of the outdated memory this replaces
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

// ReflectiveLesson represents an uncommitted lesson extracted from an error/fix sequence.
type ReflectiveLesson struct {
	Topic      string         `json:"topic"`
	Category   MemoryCategory `json:"category"`
	Summary    string         `json:"summary"`
	Mistake    string         `json:"mistake"`
	Correction string         `json:"correction"`
	Confidence int            `json:"confidence"`
	Source     string         `json:"source"`
}
