package tools

import (
	"encoding/json"
	"fmt"

	"github.com/RobinHAEVG/haevg-agent/mcp"
)

func getPackageDocumentationTool() mcp.Tool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"language": {
				"type": "string",
				"description": "Programming language to get documentation for (go, csharp)."
			},
			"package_name": {
				"type": "string",
				"description": "Name of the package to get documentation for."
			}
		},
		"required": ["language", "package_name"]
	}`)

	return mcp.Tool{
		Name:        "get_package_documentation",
		Description: "Gets documentation for a specified programming language.",

		InputSchema: schema,
	}
}

type getPackageDocumentationArgs struct {
	Language    string `json:"language"`
	PackageName string `json:"package_name"`
}

func (s *Store) getPackageDocumentation(raw json.RawMessage) (string, error) {
	var args getPackageDocumentationArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}
	var (
		result string
		err    error
	)
	switch args.Language {
	case "go":
		result, err = getGoPackageDocumentation(args.PackageName)
	case "csharp":
		result, err = getCSharpPackageDocumentation(args.PackageName)
	default:
		return "", fmt.Errorf("unsupported language: %s", args.Language)
	}

	return result, err
}

func getGoPackageDocumentation(packageName string) (string, error) {
	panic("getGoPackageDocumentation is not implemented yet")
}

func getCSharpPackageDocumentation(packageName string) (string, error) {
	panic("getCSharpPackageDocumentation is not implemented yet")
}
