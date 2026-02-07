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
sleep_duration = "3s"

[[service_groups]]
max_heartbeat_freq = "5m"

  [[service_groups.services]]
  name = "api-server"

  [[service_groups.services]]
  name = "database"

[[service_groups]]
max_heartbeat_freq = "1h"

  [[service_groups.services]]
  name = "cache-server"
`
	expectedResult = TomlConfig{
		SleepDuration: time.Second * 3,
		ServiceGroups: []service.Group{
			{
				MaxHeartbeatFreq: time.Minute * 5,
				Services: []service.Service{
					{
						Name: "api-server",
					},
					{
						Name: "database",
					},
				},
			},
			{
				MaxHeartbeatFreq: time.Hour * 1,
				Services: []service.Service{
					{
						Name: "cache-server",
					},
				},
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

	baseGroup := service.Group{
		MaxHeartbeatFreq: time.Minute * 2,
		Services:         []service.Service{{Name: "test-service"}},
	}

	tests := []struct {
		name        string
		config      TomlConfig
		expectError error
	}{
		{
			name:   "valid config",
			config: TomlConfig{ServiceGroups: []service.Group{baseGroup}},
		},
		{
			name:        "empty service groups",
			config:      TomlConfig{ServiceGroups: []service.Group{}},
			expectError: apperror.ErrNoServiceGroups,
		},
		{
			name: "heartbeat frequency too short",
			config: TomlConfig{ServiceGroups: []service.Group{
				{
					MaxHeartbeatFreq: time.Second * 30,
					Services:         baseGroup.Services,
				},
			}},
			expectError: apperror.ErrHeartbeatTooShort,
		},
		{
			name: "no services in group",
			config: TomlConfig{ServiceGroups: []service.Group{
				{
					MaxHeartbeatFreq: baseGroup.MaxHeartbeatFreq,
					Services:         []service.Service{},
				},
			}},
			expectError: apperror.ErrNoServices,
		},
		{
			name: "service name too short",
			config: TomlConfig{ServiceGroups: []service.Group{
				{
					MaxHeartbeatFreq: baseGroup.MaxHeartbeatFreq,
					Services:         []service.Service{{Name: "x"}},
				},
			}},
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
