package service

import (
	"testing"
	"time"
)

func TestNewManagerDuplicateServiceNames(t *testing.T) {
	services := []Service{
		{Name: "api", HeartbeatTimeoutDuration: time.Minute},
		{Name: "api", HeartbeatTimeoutDuration: time.Minute * 2},
	}

	if _, err := NewManager(services); err == nil {
		t.Error("expected error for duplicate service names")
	}
}

func TestUpdatePulseNonExistentService(t *testing.T) {
	services := []Service{
		{Name: "existing", HeartbeatTimeoutDuration: time.Minute},
	}
	manager, _ := NewManager(services)

	if manager.UpdatePulse("nonexistent") {
		t.Error("UpdatePulse should return false for non-existent service")
	}

	if !manager.UpdatePulse("existing") {
		t.Error("UpdatePulse should return true for existing service")
	}
}

func TestGetProblematicServicesTimeoutEdgeCases(t *testing.T) {
	now := time.Now()
	services := []Service{
		{Name: "justExpired", HeartbeatTimeoutDuration: time.Second},
		{Name: "notYetExpired", HeartbeatTimeoutDuration: time.Minute},
	}

	manager, _ := NewManager(services)

	manager.services[0].LastPulse = now.Add(-time.Second)
	manager.services[1].LastPulse = now.Add(-time.Second + time.Millisecond)

	problematic := manager.getProblematicServices()

	if len(problematic) != 1 || problematic[0].Name != "justExpired" {
		t.Error("should detect exactly expired service but not almost-expired")
	}
}
