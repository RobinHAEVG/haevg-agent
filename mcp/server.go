package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// ---------------------------------------------------------------------------
// ToolHandler
// ---------------------------------------------------------------------------

// ToolHandler is a function that executes a registered tool.
// The arguments are the raw JSON object from the client; the return value is
// plain text that will be wrapped into a ContentItem.
type ToolHandler func(args json.RawMessage) (string, error)

// registration bundles the tool metadata with its handler.
type registration struct {
	tool    Tool
	handler ToolHandler
}

// ---------------------------------------------------------------------------
// Server
// ---------------------------------------------------------------------------

// Server is a minimal MCP server that handles JSON-RPC 2.0 messages.
// Call RegisterTool to define tools, then Serve to start processing.
type Server struct {
	tools map[string]registration
}

// NewServer creates an empty, ready-to-use MCP server.
func NewServer() *Server {
	return &Server{tools: make(map[string]registration)}
}

// RegisterTool adds a tool to the server.
// schema must be a valid JSON Schema object (e.g. `{"type":"object","properties":{...}}`).
func (s *Server) RegisterTool(t Tool, handler ToolHandler) {
	s.tools[t.Name] = registration{tool: t, handler: handler}
}

// Serve reads newline-delimited JSON-RPC requests from r and writes
// responses to w until r is exhausted or an error occurs.
// It is safe to call this in a goroutine.
func (s *Server) Serve(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	// Increase the default buffer to handle large wiki pages.
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		resp := s.dispatch(line)
		data, err := json.Marshal(resp)
		if err != nil {
			return fmt.Errorf("mcp server: marshal response: %w", err)
		}

		if _, err := fmt.Fprintf(w, "%s\n", data); err != nil {
			return fmt.Errorf("mcp server: write response: %w", err)
		}
	}

	return scanner.Err()
}

// ---------------------------------------------------------------------------
// dispatch
// ---------------------------------------------------------------------------

func (s *Server) dispatch(raw []byte) Response {
	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return errorResponse(0, -32700, "parse error")
	}

	switch req.Method {
	case MethodInitialize:
		return s.handleInitialize(req)
	case MethodNotifications:
		// notifications have no response in JSON-RPC, but we send a null result
		// so the client's scanner doesn't hang waiting.
		return Response{JSONRPC: JSONRPCVersion, ID: req.ID, Result: json.RawMessage(`null`)}
	case MethodToolsList:
		return s.handleToolsList(req)
	case MethodToolsCall:
		return s.handleToolsCall(req)
	default:
		return errorResponse(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

// ---------------------------------------------------------------------------
// handlers
// ---------------------------------------------------------------------------

func (s *Server) handleInitialize(req Request) Response {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities:    map[string]interface{}{"tools": map[string]interface{}{}},
	}
	result.ServerInfo.Name = "qualitaetssicherungsagent"
	result.ServerInfo.Version = "1.0.0"

	return okResponse(req.ID, result)
}

func (s *Server) handleToolsList(req Request) Response {
	tools := make([]Tool, 0, len(s.tools))
	for _, reg := range s.tools {
		tools = append(tools, reg.tool)
	}

	return okResponse(req.ID, ToolsListResult{Tools: tools})
}

func (s *Server) handleToolsCall(req Request) Response {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, -32602, "invalid params: "+err.Error())
	}

	reg, ok := s.tools[params.Name]
	if !ok {
		return errorResponse(req.ID, -32602, fmt.Sprintf("unknown tool: %s", params.Name))
	}

	text, err := reg.handler(params.Arguments)
	if err != nil {
		result := ToolCallResult{
			Content: []ContentItem{{Type: "text", Text: err.Error()}},
			IsError: true,
		}
		return okResponse(req.ID, result)
	}

	result := ToolCallResult{
		Content: []ContentItem{{Type: "text", Text: text}},
	}
	return okResponse(req.ID, result)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func okResponse(id int64, v interface{}) Response {
	data, _ := json.Marshal(v)
	return Response{JSONRPC: JSONRPCVersion, ID: id, Result: data}
}

func errorResponse(id int64, code int, msg string) Response {
	return Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error:   &RPCError{Code: code, Message: msg},
	}
}
