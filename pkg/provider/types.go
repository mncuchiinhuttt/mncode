package provider

// Role represents chat message role
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolCall represents a model request to execute a tool
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	RawArgs   string                 `json:"rawArgs,omitempty"`
}

// ToolResult represents the output of a tool execution
type ToolResult struct {
	ToolCallID string `json:"toolCallId"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	IsError    bool   `json:"isError"`
}

// ImageData represents a base64 encoded image attachment
type ImageData struct {
	MediaType string `json:"mediaType"` // e.g. "image/png", "image/jpeg"
	Data      string `json:"data"`      // base64 data
	FilePath  string `json:"filePath,omitempty"`
}

// Message represents a conversation turn
type Message struct {
	Role        Role         `json:"role"`
	Content     string       `json:"content,omitempty"`
	Thinking    string       `json:"thinking,omitempty"`
	Images      []ImageData  `json:"images,omitempty"`
	ToolCalls   []ToolCall   `json:"toolCalls,omitempty"`
	ToolResults []ToolResult `json:"toolResults,omitempty"`
}

// ToolDefinition represents tool schema exposed to the LLM
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// EventType represents the type of streaming event
type EventType string

const (
	EventToken            EventType = "token"
	EventThinking         EventType = "thinking"
	EventToolCallStart    EventType = "tool_call_start"
	EventToolCallComplete EventType = "tool_call_complete"
	EventDone             EventType = "done"
	EventError            EventType = "error"
)

// StreamEvent is passed to the streaming callback
type StreamEvent struct {
	Type     EventType `json:"type"`
	Text     string    `json:"text,omitempty"`
	Thinking string    `json:"thinking,omitempty"`
	ToolCall *ToolCall `json:"toolCall,omitempty"`
	Error    error     `json:"error,omitempty"`
}

// CompletionRequest is the request to the LLM provider
type CompletionRequest struct {
	SystemPrompt   string
	Messages       []Message
	Tools          []ToolDefinition
	Model          string
	MaxTokens      int
	ThinkingBudget int
	Temperature    float64
}

// CompletionResponse is the final response from LLM
type CompletionResponse struct {
	Content      string
	Thinking     string
	ToolCalls    []ToolCall
	InputTokens  int
	OutputTokens int
}
