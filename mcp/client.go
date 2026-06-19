package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"
)

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client is an MCP client that communicates with an MCP server over a pair of
// io.Reader / io.Writer.  Communication is sequential (one request at a time).
type Client struct {
	r       *bufio.Reader
	w       io.Writer
	counter atomic.Int64
}

// NewClient wraps a reader (from the server) and a writer (to the server).
func NewClient(r io.Reader, w io.Writer) *Client {
	return &Client{
		r: bufio.NewReader(r),
		w: w,
	}
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// Initialize performs the MCP handshake.
func (c *Client) Initialize() error {
	params := InitializeParams{ProtocolVersion: "2024-11-05"}
	params.ClientInfo.Name = "qualitaetssicherungsagent-client"
	params.ClientInfo.Version = "1.0.0"

	var result InitializeResult
	if err := c.call(MethodInitialize, params, &result); err != nil {
		return fmt.Errorf("mcp initialize: %w", err)
	}

	// Send the initialized notification (fire and forget — server echoes a null).
	_ = c.notify(MethodNotifications, nil)

	return nil
}

// ListTools returns all tools registered on the MCP server.
func (c *Client) ListTools() ([]Tool, error) {
	var result ToolsListResult
	if err := c.call(MethodToolsList, map[string]interface{}{}, &result); err != nil {
		return nil, fmt.Errorf("mcp list tools: %w", err)
	}

	return result.Tools, nil
}

// CallTool executes the named tool with the given arguments.
// arguments must be a JSON-marshalable value matching the tool's input schema.
func (c *Client) CallTool(name string, arguments interface{}) (string, error) {
	argsRaw, err := json.Marshal(arguments)
	if err != nil {
		return "", fmt.Errorf("mcp call tool marshal args: %w", err)
	}

	params := ToolCallParams{Name: name, Arguments: argsRaw}

	var result ToolCallResult
	if err := c.call(MethodToolsCall, params, &result); err != nil {
		return "", fmt.Errorf("mcp call tool %s: %w", name, err)
	}

	if result.IsError {
		errMsg := ""
		for _, ci := range result.Content {
			errMsg += ci.Text
		}
		return "", fmt.Errorf("tool %s returned error: %s", name, errMsg)
	}

	text := ""
	for _, ci := range result.Content {
		if ci.Type == "text" {
			text += ci.Text
		}
	}
	return text, nil
}

// ---------------------------------------------------------------------------
// Internal transport helpers
// ---------------------------------------------------------------------------

func (c *Client) nextID() int64 { return c.counter.Add(1) }

// call sends a request and decodes the result into out.
func (c *Client) call(method string, params interface{}, out interface{}) error {
	id := c.nextID()

	req := Request{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  method,
	}

	if params != nil {
		raw, err := json.Marshal(params)
		if err != nil {
			return err
		}
		req.Params = raw
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(c.w, "%s\n", data); err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	// Read exactly one response line.
	line, err := c.r.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if resp.Error != nil {
		return resp.Error
	}

	if out != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, out); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}
	}

	return nil
}

// notify sends a notification (no response expected from server in MCP spec,
// but our server echoes null so we still read one line).
func (c *Client) notify(method string, params interface{}) error {
	return c.call(method, params, nil)
}
