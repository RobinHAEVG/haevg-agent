package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	Out         io.Writer
}

func (a *Agent) Run(ctx context.Context, userPrompt string) (string, error) {
	// -----------------------------------------------------------------------
	// 1. Fetch tools from the MCP server and convert to OpenAI format.
	// -----------------------------------------------------------------------
	mcpTools, err := a.MCPClient.ListTools()
	if err != nil {
		return "", fmt.Errorf("agent: list tools: %w", err)
	}

	llmTools := convertTools(mcpTools)
	a.logf("Loaded %d tool(s): %s\n", len(llmTools), toolNames(llmTools))

	// -----------------------------------------------------------------------
	// 2. Build the initial message list.
	// -----------------------------------------------------------------------
	messages := []llm.Message{}

	if a.LoadedSkill.Content != "" {
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: a.LoadedSkill.Content,
		})
	}

	messages = append(messages, llm.Message{
		Role:    "user",
		Content: userPrompt,
	})

	// -----------------------------------------------------------------------
	// 3. Tool-calling loop.
	// -----------------------------------------------------------------------
	iterations := 0
	totalToolErrors := 0
	consecutiveToolErrors := 0

	for {
		if ctx.Err() != nil {
			return "", fmt.Errorf("agent: context error: %w", ctx.Err())
		}

		iterations++
		if iterations > maxIterations {
			return "", fmt.Errorf("agent: max iterations exceeded (%d)", maxIterations)
		}

		req := llm.ChatRequest{
			Messages: messages,
			Tools:    llmTools,
		}

		a.logf("[LLM] Sending request with %d message(s)...\n", len(messages))

		resp, err := a.LLMClient.Chat(ctx, req)
		if err != nil {
			return "", fmt.Errorf("agent: LLM call: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("agent: LLM returned no choices")
		}

		choice := resp.Choices[0]
		finishing := choice.FinishReason
		assistantMsg := choice.Message

		a.logf("[LLM] finish_reason=%q\n", finishing)

		switch finishing {
		case "tool_calls":
			// Append the assistant message (which carries the tool_calls).
			messages = append(messages, assistantMsg)

			// Execute each tool call via the MCP client.
			for _, tc := range assistantMsg.ToolCalls {
				result, err := a.executeTool(ctx, tc)
				if err != nil {
					totalToolErrors++
					consecutiveToolErrors++

					// Propagate the error as a tool result so the LLM can react.
					result = fmt.Sprintf("Error: %s", err.Error())

					if consecutiveToolErrors > maxConsecutiveToolErrors {
						return "", fmt.Errorf("agent: consecutive tool error budget exceeded (%d)", maxConsecutiveToolErrors)
					}

					if totalToolErrors > maxTotalToolErrors {
						return "", fmt.Errorf("agent: total tool error budget exceeded (%d)", maxTotalToolErrors)
					}
				} else {
					consecutiveToolErrors = 0
				}

				// Append the tool result.
				messages = append(messages, llm.Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Name:       tc.Function.Name,
					Content:    result,
				})
			}

			// Continue the loop so the LLM can react to the tool results.

		case "stop", "":
			// The LLM has produced a final answer.
			return assistantMsg.Content, nil

		case "length":
			// The response was cut short by the token limit; return what we have.
			a.logf("[WARN] LLM response was truncated (finish_reason=length).\n")
			return assistantMsg.Content, nil

		default:
			// Unknown finish reason — treat like stop.
			a.logf("[WARN] Unknown finish_reason=%q; treating as stop.\n", finishing)
			return assistantMsg.Content, nil
		}
	}
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
	if a.Out != nil {
		fmt.Fprintf(a.Out, format, args...)
	}
}

func toolNames(tools []llm.Tool) string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Function.Name
	}
	return strings.Join(names, ", ")
}
