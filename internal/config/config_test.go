package config

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	apperror "service-uptime-center/internal/error"
	"service-uptime-center/internal/service"
)

var (
	mockConfig = `
incident_poll_frequency = "3s"

[[services]]
name = "api-server"
heartbeat_timeout_duration = "90s"

[[services]]
name = "database"
heartbeat_timeout_duration = "5m"

[[services]]
heartbeat_timeout_duration = "1h"
name = "cache-server"
`
	expectedResult = TomlConfig{
		IncidentsPollFreq: time.Second * 3,
		Services: []service.Service{
			{
				Name:                     "api-server",
				HeartbeatTimeoutDuration: time.Second * 90,
			},
			{
				Name:                     "database",
				HeartbeatTimeoutDuration: time.Minute * 5,
			},
			{
				Name:                     "cache-server",
				HeartbeatTimeoutDuration: time.Hour * 1,
			},
		},
	}
)

func TestConfigCreation(t *testing.T) {
	cfg, err := Parse(TomlStringDecoder, mockConfig)
	if err != nil {
		t.Errorf("failed to create config: %v", err)
	}

	if diff := cmp.Diff(expectedResult, *cfg); diff != "" {
		t.Errorf("config mismatch (-want +got):\n%s", diff)
	}
}

func TestValidateConfig(t *testing.T) {
	t.Run("mockConfig should be valid", func(t *testing.T) {
		_, err := Parse(TomlStringDecoder, mockConfig)
		if err != nil {
			t.Fatalf("mockConfig should parse and validate without error: %v", err)
		}
	})

	tests := []struct {
		name        string
		config      TomlConfig
		expectError error
	}{
		{
			name:        "empty services",
			config:      TomlConfig{IncidentsPollFreq: time.Hour * 2, Services: nil},
			expectError: apperror.ErrNoServices,
		},
		{
			name: "heartbeat timeout duration too short",
			config: TomlConfig{
				Services: []service.Service{
					{
						HeartbeatTimeoutDuration: time.Second * 30,
						Name:                     "test-service-1",
					},
				},
			},
			expectError: apperror.ErrHeartbeatTimeoutTooShort,
		},
		{
			name: "service name too short",
			config: TomlConfig{
				Services: []service.Service{
					{
						HeartbeatTimeoutDuration: time.Hour * 30,
						Name:                     "x",
					},
				},
			},
			expectError: apperror.ErrInvalidServiceName,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.Validate()

			if test.expectError == nil {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error %v but got none", test.expectError)
				} else if !errors.Is(err, test.expectError) {
					t.Errorf("expected error %v but got %v", test.expectError, err)
				}
			}
		})
	}
}
