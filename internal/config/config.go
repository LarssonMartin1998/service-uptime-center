// Package config
package config

import (
	"fmt"
	"time"

	"github.com/BurntSushi/toml"

	apperrors "service-uptime-center/internal/error"
	service "service-uptime-center/internal/service"
)

type TomlConfig struct {
	IncidentsPollFreq time.Duration     `toml:"incident_poll_frequency"`
	Services          []service.Service `toml:"services"`
}

func TomlStringDecoder(data string) (*TomlConfig, error) {
	var cfg TomlConfig
	_, err := toml.Decode(data, &cfg)
	return &cfg, err
}

func TomlFileDecoder(filePath string) (*TomlConfig, error) {
	var cfg TomlConfig
	_, err := toml.DecodeFile(filePath, &cfg)
	return &cfg, err
}

type TomlDecoder func(string) (*TomlConfig, error)

func Parse(decodeToml TomlDecoder, value string) (*TomlConfig, error) {
	cfg, err := decodeToml(value)
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (cfg *TomlConfig) Validate() error {
	if len(cfg.Services) == 0 {
		return fmt.Errorf("%w", apperrors.ErrNoServices)
	}

	for _, service := range cfg.Services {
		const MinHeartbeatFreq = time.Second * 60
		if service.HeartbeatTimeoutDuration < MinHeartbeatFreq {
			return fmt.Errorf("%w (min: %v): %v", apperrors.ErrHeartbeatTimeoutTooShort, MinHeartbeatFreq, service.HeartbeatTimeoutDuration)
		}

		const MinNameLen = 2
		const MaxNameLen = 64
		if len(service.Name) < MinNameLen || len(service.Name) > MaxNameLen {
			return fmt.Errorf("%w (min: %d, max: %d): %s", apperrors.ErrInvalidServiceName, MinNameLen, MaxNameLen, service.Name)
		}
	}

	return nil
}
