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
	SleepDuration time.Duration   `toml:"sleep_duration"`
	ServiceGroups []service.Group `toml:"service_groups"`
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
	if len(cfg.ServiceGroups) == 0 {
		return apperrors.ErrNoServiceGroups
	}

	for _, grp := range cfg.ServiceGroups {
		const MinHeartbeatFreq = time.Second * 60
		if grp.MaxHeartbeatFreq < MinHeartbeatFreq {
			return fmt.Errorf("%w (min: %v): %v", apperrors.ErrHeartbeatTooShort, MinHeartbeatFreq, grp.MaxHeartbeatFreq)
		}

		if len(grp.Services) == 0 {
			return fmt.Errorf("%w", apperrors.ErrNoServices)
		}

		const MinNameLen = 2
		const MaxNameLen = 64
		for _, service := range grp.Services {
			if len(service.Name) < MinNameLen || len(service.Name) > MaxNameLen {
				return fmt.Errorf("%w (min: %d, max: %d): %s", apperrors.ErrInvalidServiceName, MinNameLen, MaxNameLen, service.Name)
			}
		}
	}

	return nil
}
