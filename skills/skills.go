package skills

import (
	"embed"
)

//go:embed skillfiles/*
var skillFiles embed.FS

func Get(name string) (string, error) {
	content, err := skillFiles.ReadFile("skillfiles/" + name)
	return string(content), err
}

func List() ([]string, error) {
	entries, err := skillFiles.ReadDir("skillfiles")
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}

	return names, nil
}
