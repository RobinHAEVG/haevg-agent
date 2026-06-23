package tools

import (
	"encoding/json"
	"io"
	"os"

	"github.com/RobinHAEVG/haevg-agent/mcp"
)

func readFileTool() mcp.Tool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"file_path": {
				"type": "string",
				"description": "Path to the file to read."
			}
		},
		"required": ["file_path"]
	}`)

	return mcp.Tool{
		Name:        "read_file",
		Description: "Reads the content of a file.",
		InputSchema: schema,
	}
}

type readFileArgs struct {
	FilePath string `json:"file_path"`
}

func (s *Store) readFile(raw json.RawMessage) (string, error) {
	var args readFileArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	root, err := os.OpenRoot(s.workDir)
	if err != nil {
		return "", err
	}
	defer root.Close()

	file, err := root.Open(args.FilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
