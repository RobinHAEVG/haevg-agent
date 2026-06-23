package tools

import (
	"encoding/json"
	"os"

	"github.com/RobinHAEVG/haevg-agent/mcp"
)

func writeFileTool() mcp.Tool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"file_path": {
				"type": "string",
				"description": "Path to the file to write."
			},
			"content": {
				"type": "string",
				"description": "Content to write to the file."
			}
		},
		"required": ["file_path", "content"]
	}`)

	return mcp.Tool{
		Name:        "write_file",
		Description: "Writes content to a file.",
		InputSchema: schema,
	}
}

type writeFileArgs struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func (s *Store) writeFile(raw json.RawMessage) (string, error) {
	var args writeFileArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	root, err := os.OpenRoot(s.workDir)
	if err != nil {
		return "", err
	}
	defer root.Close()

	file, err := root.Create(args.FilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = file.Write([]byte(args.Content))
	if err != nil {
		return "", err
	}

	return "", nil
}
