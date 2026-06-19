package skills

type SkillMetadata struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	AllowedTools []string `json:"allowed_tools"`
}

// ParseSkill parses the content of a skill file and returns the markdown content and the extracted metadata.
func ParseSkill(content string) (string, *SkillMetadata, error) {
	// implement me
}
