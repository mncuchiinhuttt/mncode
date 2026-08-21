package mcp

// ServerConfig defines the launch configuration for an MCP server
type ServerConfig struct {
	Command  string            `json:"command"`
	Args     []string          `json:"args,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	Disabled bool              `json:"disabled,omitempty"`
}

// Config represents the root mcp.json file structure
type Config struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

// MCPToolInfo holds metadata about a tool exposed by an MCP server
type MCPToolInfo struct {
	ServerName  string
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ServerStatus describes the runtime health and tool count of an MCP server
type ServerStatus struct {
	Name      string
	Command   string
	Args      []string
	Connected bool
	Error     string
	Tools     []MCPToolInfo
}

// JSON-RPC 2.0 structures
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
