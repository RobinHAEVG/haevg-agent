package skills

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type SkillMetadata struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	AllowedTools []string `json:"allowed_tools"`
}

type Skill struct {
	Metadata SkillMetadata
	Content  string
}

const delim = "---"

// ParseSkill parses the content of a skill file and returns the markdown content and the extracted metadata.
func ParseSkill(name string) (*Skill, error) {
	rawContent, err := skillFiles.ReadFile("skillfiles/" + name + ".md")
	if err != nil {
		return nil, fmt.Errorf("skills: read skill file '%s': %w", name, err)
	}
	content := string(rawContent)

	content = strings.TrimPrefix(content, "\ufeff")

	if !strings.HasPrefix(content, delim+"\n") && content != delim {
		return nil, fmt.Errorf("skills: missing frontmatter delimiter")
	}

	startLen := len(delim)
	if len(content) >= startLen+1 && content[startLen] == '\r' {
		startLen++
	}
	if len(content) > startLen && content[startLen] == '\n' {
		startLen++
	}

	rest := content[startLen:]
	closeIdx := strings.Index(rest, "\n"+delim)
	if closeIdx < 0 {
		if strings.HasPrefix(rest, delim) {
			closeIdx = 0
		} else {
			return nil, fmt.Errorf("skills: missing closing frontmatter delimiter")
		}
	}

	header := rest[:closeIdx]
	after := rest[closeIdx:]

	if strings.HasPrefix(after, "\n"+delim) {
		after = after[1+len(delim):]
	} else if strings.HasPrefix(after, delim) {
		after = after[len(delim):]
	}

	if strings.HasPrefix(after, "\r\n") {
		after = after[2:]
	} else if strings.HasPrefix(after, "\n") {
		after = after[1:]
	}

	meta := &SkillMetadata{}
	if err := yaml.Unmarshal([]byte(header), meta); err != nil {
		return nil, fmt.Errorf("skills: parse frontmatter: %w", err)
	}

	return &Skill{
		Metadata: *meta,
		Content:  after,
	}, nil
}
