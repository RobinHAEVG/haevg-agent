package skills

import (
	"embed"
	"strings"
)

//go:embed skillfiles/*
var skillFiles embed.FS

func List() ([]string, error) {
	entries, err := skillFiles.ReadDir("skillfiles")
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, strings.TrimSuffix(entry.Name(), ".md"))
		}
	}

	return names, nil
}
