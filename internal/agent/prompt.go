package agent

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const defaultPromptPath = "prompt.yaml"

type promptConfig struct {
	Instruction string `yaml:"instruction"`
}

func loadInstruction(path string) (string, error) {
	if path == "" {
		path = defaultPromptPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading prompt %q: %w", path, err)
	}

	var pc promptConfig
	if err := yaml.Unmarshal(data, &pc); err != nil {
		return "", fmt.Errorf("parsing prompt.yaml: %w", err)
	}
	if pc.Instruction == "" {
		return "", fmt.Errorf("instruction is empty in %q", path)
	}

	return pc.Instruction, nil
}
