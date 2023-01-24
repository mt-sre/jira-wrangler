package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

var ErrUnknownFileType = errors.New("unknown file type")

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	var config Config

	switch strings.TrimPrefix(filepath.Ext(path), ".") {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("unmarshalling config: %w", err)
		}
	default:
		return nil, ErrUnknownFileType
	}

	return &config, nil
}

type Config struct {
	Title   string         `json:"title"`
	Reports []ReportConfig `json:"reports"`
}

type ReportConfig struct {
	Title string `json:"title"`
	Label string `json:"label"`
}
