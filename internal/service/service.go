// Package service
package service

import (
	"encoding/json"
	"time"
)

type Service struct {
	Name                     string        `toml:"name"`
	HeartbeatTimeoutDuration time.Duration `toml:"heartbeat_timeout_duration"`
	LastPulse                time.Time
	LastProblem              time.Time
	LastSuccessReport        time.Time
}

func (s *Service) MarshalJSON() ([]byte, error) {
	result := map[string]any{
		"name":                       s.Name,
		"is_problematic":             s.isProblematic(),
		"heartbeat_timeout_duration": s.HeartbeatTimeoutDuration.String(),
	}

	if !s.LastPulse.IsZero() {
		result["last_pulse"] = s.LastPulse.Format(time.RFC3339)
	}
	if !s.LastProblem.IsZero() {
		result["last_problem"] = s.LastProblem.Format(time.RFC3339)
	}
	if !s.LastSuccessReport.IsZero() {
		result["last_success_report"] = s.LastSuccessReport.Format(time.RFC3339)
	}

	return json.Marshal(result)
}

func (s *Service) isProblematic() bool {
	return time.Since(s.LastPulse) >= s.HeartbeatTimeoutDuration
}
