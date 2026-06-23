package agents

import (
	"context"
	"fmt"

	"github.com/RobinHAEVG/haevg-agent/llm"
)

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
