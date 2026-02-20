package service

import (
	"encoding/json"
	"fmt"
	"service-uptime-center/internal/app/apperror"
	"time"
)

type Config struct {
	Services []Service `toml:"services"`
}

func (c *Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"services": c.Services,
	})
}

func (c *Config) Validate() error {
	if len(c.Services) == 0 {
		return apperror.ErrNoServices
	}

	for _, service := range c.Services {
		const MinHeartbeatFreq = time.Second * 60
		if service.HeartbeatTimeoutDuration < MinHeartbeatFreq {
			return fmt.Errorf("%w (min: %v): %v", apperror.ErrHeartbeatTimeoutTooShort, MinHeartbeatFreq, service.HeartbeatTimeoutDuration)
		}

		const MinNameLen = 2
		const MaxNameLen = 64
		if len(service.Name) < MinNameLen || len(service.Name) > MaxNameLen {
			return fmt.Errorf("%w (min: %d, max: %d): %s", apperror.ErrInvalidServiceName, MinNameLen, MaxNameLen, service.Name)
		}
	}

	return nil
}
