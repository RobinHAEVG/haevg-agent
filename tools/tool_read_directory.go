package tools

import (
	"encoding/json"
	"os"

	"github.com/RobinHAEVG/haevg-agent/mcp"
)

func readDirectoryTool() mcp.Tool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"directory_path": {
				"type": "string",
				"description": "Path to the directory to read."
			}
		},
		"required": ["directory_path"]
	}`)

	return mcp.Tool{
		Name:        "read_directory",
		Description: "Reads the contents of a directory recursively.",
		InputSchema: schema,
	}
}

type readDirectoryArgs struct {
	DirectoryPath string `json:"directory_path"`
}

func (s *Store) readDirectory(raw json.RawMessage) (string, error) {
	var args readDirectoryArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	root, err := os.OpenRoot(s.workDir)
	if err != nil {
		return "", err
	}
	defer root.Close()

	var paths []string
	err = walkDir(root, args.DirectoryPath, &paths)
	if err != nil {
		return "", err
	}

	result, err := json.Marshal(paths)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func walkDir(root *os.Root, dirPath string, paths *[]string) error {
	file, err := root.Open(dirPath)
	if err != nil {
		return err
	}
	defer file.Close()

	entries, err := file.ReadDir(-1)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		fullPath := dirPath + "/" + entry.Name()
		*paths = append(*paths, fullPath)

		if entry.IsDir() {
			err := walkDir(root, fullPath, paths)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
