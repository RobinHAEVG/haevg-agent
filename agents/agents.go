package agents

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/RobinHAEVG/haevg-agent/configuration"
	"github.com/RobinHAEVG/haevg-agent/llm"
	"github.com/RobinHAEVG/haevg-agent/mcp"
	"github.com/RobinHAEVG/haevg-agent/skills"
)

const (
	maxIterations            = 10
	maxConsecutiveToolErrors = 2
	maxTotalToolErrors       = 5
)

type Agent struct {
	Config      *configuration.AppConfig
	LLMClient   *llm.Client
	LoadedSkill *skills.Skill
	MCPClient   *mcp.Client
	Verbose     bool
	Logger      *slog.Logger
}

// ---------------------------------------------------------------------------
// executeTool
// ---------------------------------------------------------------------------

func (a *Agent) executeTool(ctx context.Context, tc llm.ToolCall) (string, error) {
	name := tc.Function.Name
	rawArgs := tc.Function.Arguments

	// Parse the arguments so we can pretty-print them.
	var argsMap map[string]interface{}
	_ = json.Unmarshal([]byte(rawArgs), &argsMap)

	if a.Verbose {
		pretty, _ := json.MarshalIndent(argsMap, "  ", "  ")
		a.logf("[TOOL CALL] %s\n  args: %s\n", name, pretty)
	}

	// Delegate to the MCP client which forwards to the MCP server.
	result, err := a.MCPClient.CallTool(name, json.RawMessage(rawArgs))
	if err != nil {
		if a.Verbose {
			a.logf("[TOOL RESULT] %s → ERROR: %s\n", name, err)
		}
		return "", err
	}

	if a.Verbose {
		a.logf("[TOOL RESULT] %s → %s\n", name, result)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// convertTools — MCP → OpenAI format
// ---------------------------------------------------------------------------

// convertTools converts a slice of MCP tool definitions to the OpenAI Tool
// format expected by the chat-completions API.
func convertTools(mcpTools []mcp.Tool) []llm.Tool {
	out := make([]llm.Tool, 0, len(mcpTools))
	for _, t := range mcpTools {
		out = append(out, llm.Tool{
			Type: "function",
			Function: llm.Function{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (a *Agent) logf(format string, args ...interface{}) {
	if a.Logger != nil {
		a.Logger.Info(format, args...)
	}
}

func toolNames(tools []llm.Tool) string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Function.Name
	}
	return strings.Join(names, ", ")
}
