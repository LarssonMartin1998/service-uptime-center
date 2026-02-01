package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

var (
	mockConfig = `
[[service_groups]]
name = "production-group"
auth_token = "token-for-group-1"
max_heartbeat_freq = "5m"

  [[service_groups.services]]
  name = "api-server"

  [[service_groups.services]]
  name = "database"

[[service_groups]]
name = "staging-group"
auth_token = "token-for-group-2"  
max_heartbeat_freq = "1h"

  [[service_groups.services]]
  name = "cache-server"
`
	expectedResult = config{
		ServiceGroups: []serviceGroup{
			{
				Name:             "production-group",
				AuthToken:        "token-for-group-1",
				MaxHeartbeatFreq: time.Minute * 5,
				Services: []service{
					{
						Name: "api-server",
					},
					{
						Name: "database",
					},
				},
			},
			{
				Name:             "staging-group",
				AuthToken:        "token-for-group-2",
				MaxHeartbeatFreq: time.Hour * 1,
				Services: []service{
					{
						Name: "cache-server",
					},
				},
			},
		},
	}
)

func TestConfigCreation(t *testing.T) {
	cfg, err := createConfig(mockConfig)
	if err != nil {
		t.Errorf("failed to create config: %v", err)
	}

	if diff := cmp.Diff(expectedResult, cfg); diff != "" {
		t.Errorf("config mismatch (-want +got):\n%s", diff)
	}
}

func TestValidateConfig(t *testing.T) {
	t.Run("mockConfig should be valid", func(t *testing.T) {
		cfg, err := createConfig(mockConfig)
		if err != nil {
			t.Fatalf("mockConfig should parse without error: %v", err)
		}

		err = validateConfig(&cfg)
		if err != nil {
			t.Errorf("mockConfig should validate without error: %v", err)
		}
	})

	baseGroup := serviceGroup{
		Name:             "test-group",
		AuthToken:        "valid-token",
		MaxHeartbeatFreq: time.Minute * 2,
		Services:         []service{{Name: "test-service"}},
	}

	tests := []struct {
		name        string
		config      config
		expectError error
	}{
		{
			name:   "valid config",
			config: config{ServiceGroups: []serviceGroup{baseGroup}},
		},
		{
			name:        "empty service groups",
			config:      config{ServiceGroups: []serviceGroup{}},
			expectError: ErrNoServiceGroups,
		},
		{
			name: "service group name too short",
			config: config{ServiceGroups: []serviceGroup{
				{
					Name:             "a",
					AuthToken:        baseGroup.AuthToken,
					MaxHeartbeatFreq: baseGroup.MaxHeartbeatFreq,
					Services:         baseGroup.Services,
				},
			}},
			expectError: ErrInvalidGroupName,
		},
		{
			name: "service group name too long",
			config: config{ServiceGroups: []serviceGroup{
				{
					Name:             "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					AuthToken:        baseGroup.AuthToken,
					MaxHeartbeatFreq: baseGroup.MaxHeartbeatFreq,
					Services:         baseGroup.Services,
				},
			}},
			expectError: ErrInvalidGroupName,
		},
		{
			name: "auth token too long",
			config: config{ServiceGroups: []serviceGroup{
				{
					Name:             baseGroup.Name,
					AuthToken:        strings.Repeat("a", 256),
					MaxHeartbeatFreq: baseGroup.MaxHeartbeatFreq,
					Services:         baseGroup.Services,
				},
			}},
			expectError: ErrAuthTokenTooLong,
		},
		{
			name: "heartbeat frequency too short",
			config: config{ServiceGroups: []serviceGroup{
				{
					Name:             baseGroup.Name,
					AuthToken:        baseGroup.AuthToken,
					MaxHeartbeatFreq: time.Second * 30,
					Services:         baseGroup.Services,
				},
			}},
			expectError: ErrHeartbeatTooShort,
		},
		{
			name: "no services in group",
			config: config{ServiceGroups: []serviceGroup{
				{
					Name:             baseGroup.Name,
					AuthToken:        baseGroup.AuthToken,
					MaxHeartbeatFreq: baseGroup.MaxHeartbeatFreq,
					Services:         []service{},
				},
			}},
			expectError: ErrNoServices,
		},
		{
			name: "service name too short",
			config: config{ServiceGroups: []serviceGroup{
				{
					Name:             baseGroup.Name,
					AuthToken:        baseGroup.AuthToken,
					MaxHeartbeatFreq: baseGroup.MaxHeartbeatFreq,
					Services:         []service{{Name: "x"}},
				},
			}},
			expectError: ErrInvalidServiceName,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateConfig(&test.config)

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
