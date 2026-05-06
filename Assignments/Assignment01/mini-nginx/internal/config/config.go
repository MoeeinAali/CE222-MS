package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenOn int              `yaml:"listen-on"`
	Paths    map[string]Route `yaml:"paths"`
}

type Route struct {
	Type   string `yaml:"type"`
	Target string `yaml:"target"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
