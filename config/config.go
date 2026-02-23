// Package config
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ValidatableConfig interface {
	Validate() error
}

func YamlStringDecoder[T ValidatableConfig](data string) (T, error) {
	var cfg T
	err := yaml.Unmarshal([]byte(data), &cfg)
	return cfg, err
}

func YamlFileDecoder[T ValidatableConfig](filePath string) (T, error) {
	var cfg T
	data, err := os.ReadFile(filePath)
	if err != nil {
		return cfg, err
	}
	err = yaml.Unmarshal(data, &cfg)
	return cfg, err
}

type YamlDecoder[T ValidatableConfig] func(string) (T, error)

func Parse[T ValidatableConfig](decodeYaml YamlDecoder[T], value string) (T, error) {
	cfg, err := decodeYaml(value)
	if err != nil {
		var zero T
		return zero, err
	}

	if err := cfg.Validate(); err != nil {
		var zero T
		return zero, err
	}

	return cfg, nil
}
