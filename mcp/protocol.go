// Package mcp implements a minimal MCP (Model Context Protocol) layer
// using JSON-RPC 2.0 over a pair of io.Reader / io.Writer (e.g., stdin/stdout
// or in-process io.Pipe connections).
package mcp

import "encoding/json"

// ---------------------------------------------------------------------------
// JSON-RPC 2.0 base types
// ---------------------------------------------------------------------------

// Request is a JSON-RPC 2.0 request message.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response message.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError carries a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string { return e.Message }

// ---------------------------------------------------------------------------
// MCP domain types
// ---------------------------------------------------------------------------

// Tool describes a callable tool exposed by the MCP server.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToolsListResult is returned by the tools/list method.
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// ToolCallParams is the parameter block for tools/call.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ContentItem is a single piece of content inside a tool response.
type ContentItem struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// ToolCallResult is returned by tools/call.
type ToolCallResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// InitializeParams carries the client's capabilities during handshake.
type InitializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

// InitializeResult is the server's response to initialize.
type InitializeResult struct {
	ProtocolVersion string `json:"protocolVersion"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
	Capabilities map[string]interface{} `json:"capabilities"`
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	JSONRPCVersion = "2.0"

	MethodInitialize    = "initialize"
	MethodToolsList     = "tools/list"
	MethodToolsCall     = "tools/call"
	MethodNotifications = "notifications/initialized"
)
