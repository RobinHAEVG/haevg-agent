package tools

import (
	"encoding/json"
	"fmt"

	"github.com/RobinHAEVG/haevg-agent/mcp"
)

// get_latest_pipeline_logs

func getLatestPipelineLogsTool() mcp.Tool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"pipelineId": {
				"type": "string",
				"description": "ID of the pipeline to get logs for."
			},
			"runId": {
				"type": "string",
				"description": "ID of the specific run to get logs for."
			}
		},
		"required": ["pipelineId", "runId"]
	}`)

	return mcp.Tool{
		Name:        "get_latest_pipeline_logs",
		Description: "Gets the latest logs for a specified pipeline run.",

		InputSchema: schema,
	}
}

type getLatestPipelineLogsArgs struct {
	PipelineId string `json:"pipelineId"`
	RunId      string `json:"runId"`
}

func (s *Store) getLatestPipelineLogs(raw json.RawMessage) (string, error) {
	var args getLatestPipelineLogsArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}
	
	

	return result, err
}
