package configuration

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type (
	AppConfig struct {
		LLM         LLM         `yaml:"llm"`
		AzureDevops AzureDevops `yaml:"azure_devops"`
	}

	LLM struct {
		BaseURL string        `yaml:"base_url"`
		Model   string        `yaml:"model"`
		APIKey  string        `yaml:"api_key"`
		Timeout time.Duration `yaml:"timeout"`
	}
	AzureDevops struct {
		Organization string `yaml:"organization"`
		Project      string `yaml:"project"`
		APIKey       string `yaml:"api_key"`
	}
)

func Load(path string) (*AppConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Konnte Konfiguration nicht laden: %w", err)
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("Konnte Konfiguration nicht parsen: %w", err)
	}

	return &cfg, nil
}

func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("Konnte Home-Verzeichnis nicht ermitteln: %v", err))
	}

	return fmt.Sprintf("%s/.haevg-agent", home)
}
