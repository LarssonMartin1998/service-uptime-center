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
[timings]
incident_poll_frequency = "3s"
successful_report_cooldown = "168h"
problematic_report_cooldown = "24h"

[[services]]
name = "api-server"
heartbeat_timeout_duration = "90s"
notifiers = ["email"]

[[services]]
name = "database"
heartbeat_timeout_duration = "5m"
notifiers = ["email"]

[[services]]
heartbeat_timeout_duration = "1h"
name = "cache-server"
notifiers = ["email"]
`
	expectedResult = service.Config{
		Timings: service.TimingIntervals{
			IncidentsPollFreq:         time.Second * 3,
			SuccessfulReportCooldown:  time.Hour * 168,
			ProblematicReportCooldown: time.Hour * 24,
		},
		Services: []service.Service{
			{
				Name:                     "api-server",
				HeartbeatTimeoutDuration: time.Second * 90,
				NotifiersStr:             []string{"email"},
			},
			{
				Name:                     "database",
				HeartbeatTimeoutDuration: time.Minute * 5,
				NotifiersStr:             []string{"email"},
			},
			{
				Name:                     "cache-server",
				HeartbeatTimeoutDuration: time.Hour * 1,
				NotifiersStr:             []string{"email"},
			},
		},
	}
)

func TestConfigCreation(t *testing.T) {
	cfg, err := Parse(TomlStringDecoder[*service.Config], mockConfig)
	if err != nil {
		t.Errorf("failed to create config: %v", err)
	}

	if diff := cmp.Diff(expectedResult, *cfg); diff != "" {
		t.Errorf("config mismatch (-want +got):\n%s", diff)
	}
}

func TestValidateConfig(t *testing.T) {
	t.Run("mockConfig should be valid", func(t *testing.T) {
		_, err := Parse(TomlStringDecoder[*service.Config], mockConfig)
		if err != nil {
			t.Fatalf("mockConfig should parse and validate without error: %v", err)
		}
	})

	tests := []struct {
		name        string
		config      service.Config
		expectError error
	}{
		{
			name:        "empty services",
			config:      service.Config{Timings: service.TimingIntervals{IncidentsPollFreq: time.Hour * 2}, Services: nil},
			expectError: apperror.ErrNoServices,
		},
		{
			name: "heartbeat timeout duration too short",
			config: service.Config{
				Services: []service.Service{
					{
						HeartbeatTimeoutDuration: time.Second * 30,
						Name:                     "test-service-1",
						NotifiersStr:             []string{"email"},
					},
				},
			},
			expectError: apperror.ErrHeartbeatTimeoutTooShort,
		},
		{
			name: "service name too short",
			config: service.Config{
				Services: []service.Service{
					{
						HeartbeatTimeoutDuration: time.Hour * 30,
						Name:                     "x",
						NotifiersStr:             []string{"email"},
					},
				},
			},
			expectError: apperror.ErrInvalidServiceName,
		},
		{
			name: "no notifiers configured for service",
			config: service.Config{
				Services: []service.Service{
					{
						HeartbeatTimeoutDuration: time.Hour * 30,
						Name:                     "test-name",
						NotifiersStr:             nil,
					},
				},
			},
			expectError: apperror.ErrNoNotifiers,
		},
		{
			name: "invalid notifyer protocl configured for service",
			config: service.Config{
				Services: []service.Service{
					{
						HeartbeatTimeoutDuration: time.Hour * 30,
						Name:                     "test-name",
						NotifiersStr:             []string{"facetime"},
					},
				},
			},
			expectError: apperror.ErrInvalidNotifProtocol,
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
