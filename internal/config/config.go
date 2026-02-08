// Package config
package config

import (
	"github.com/BurntSushi/toml"
)

type ValidatableConfig interface {
	Validate() error
}

func TomlStringDecoder[T ValidatableConfig](data string) (T, error) {
	var cfg T
	_, err := toml.Decode(data, &cfg)
	return cfg, err
}

func TomlFileDecoder[T ValidatableConfig](filePath string) (T, error) {
	var cfg T
	_, err := toml.DecodeFile(filePath, &cfg)
	return cfg, err
}

type TomlDecoder[T ValidatableConfig] func(string) (T, error)

func Parse[T ValidatableConfig](decodeToml TomlDecoder[T], value string) (T, error) {
	cfg, err := decodeToml(value)
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
