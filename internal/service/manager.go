package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"service-uptime-center/internal/app/apperror"
	"service-uptime-center/internal/app/timings"
	"service-uptime-center/notification"
	"sync"
	"time"
)

type Manager struct {
	cfg    *Config
	lookup map[string]*Service
	mutex  sync.RWMutex
}

func NewManager(cfg *Config) (*Manager, error) {
	now := time.Now()
	lookup := make(map[string]*Service, len(cfg.Services))

	for i := range cfg.Services {
		cfg.Services[i].LastPulse = now

		_, ok := lookup[cfg.Services[i].Name]
		if ok {
			return nil, apperror.ErrDuplicateServiceNames
		}

		lookup[cfg.Services[i].Name] = &cfg.Services[i]
	}

	return &Manager{
		cfg:    cfg,
		lookup: lookup,
	}, nil
}

func (m *Manager) GetStatusJSON() ([]byte, error) {
	// We could serialize the JSON as soon as any service changes come through and cache it
	// instead of evaluating it each call.
	json, err := json.Marshal(m.cfg)
	if err != nil {
		slog.Error("Error marshaling service config", "error", err)
		return nil, err
	}

	return json, nil
}

func (m *Manager) UpdatePulse(name string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if service, exists := m.lookup[name]; exists {
		service.LastPulse = time.Now()
		return true
	}
	return false
}

type MonitoringInstructions struct {
	Timings   *timings.Timings
	Notifiers notification.ProtocolTargets
}

func (m *Manager) StartMonitoring(notificationManager *notification.Manager, instr MonitoringInstructions) {
	go func() {
		for {
			problematic := m.getProblematicServices()
			if len(problematic) > 0 {
				m.handleProblematicServices(notificationManager, instr.Notifiers, problematic, instr.Timings.ProblematicReportCooldown)
			}

			time.Sleep(instr.Timings.IncidentsPollFreq)
		}
	}()

	start := time.Now()
	go func() {
		for {
			time.Sleep(instr.Timings.SuccessfulReportCooldown)

			if err := notificationManager.SendWithFallback(instr.Notifiers, notification.SendData{
				Title: "Service Uptime Center running without any issues.",
				Body:  "",
			}); err != nil {
				slog.Error("Cannot send notification, monitoring may be compromised", "error", err)
				continue
			}

			slog.Info("Service is still running", "uptime", time.Since(start).String())
		}
	}()
}

func (m *Manager) getProblematicServices() []*Service {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var problematic []*Service
	for i := range m.cfg.Services {
		service := &m.cfg.Services[i]

		if service.isProblematic() {
			problematic = append(problematic, service)
		}
	}
	return problematic
}

func (m *Manager) handleProblematicServices(notificationManager *notification.Manager, targets notification.ProtocolTargets, services []*Service, problematicReportCooldown time.Duration) {
	m.mutex.Lock()

	now := time.Now()
	var buf bytes.Buffer

	reportedCount := 0
	for _, service := range services {
		service.LastProblem = now
		problemDuration := time.Since(service.LastPulse)
		overdue := problemDuration - service.HeartbeatTimeoutDuration

		if service.isProblematicReportCooldownActive(problematicReportCooldown) {
			cooldownEndTime := service.LastProblemReported.Add(problematicReportCooldown)
			remainingCooldown := time.Until(cooldownEndTime)
			slog.Info("Leaving out problematic service from notification because it's on report cooldown.", "service", service.Name, "remaining cooldown", remainingCooldown)
		} else {
			if reportedCount == 0 {
				buf.WriteString("Service Name, Last Pulse, Problem Duration, Overdue\n")
			}
			reportedCount++
			service.LastProblemReported = now
			if _, err := fmt.Fprintf(&buf, "%s, %s, %s, %s\n", service.Name, service.LastPulse.String(), problemDuration.String(), overdue.String()); err != nil {
				slog.Error("Failed to write service data to buffer, notification will be missing this data", "service", service.Name, "error", err)
			}
		}
	}

	m.mutex.Unlock()

	slog.Info("Detected problematic", "services", services)

	body := buf.String()
	if reportedCount == 0 {
		slog.Info("All problematic services are on report cooldown, skipping notification")
		return
	}

	data := notification.SendData{
		Title: fmt.Sprintf("Problem detected with %d services", reportedCount),
		Body:  body,
	}

	if err := notificationManager.SendWithFallback(targets, data); err != nil {
		slog.Error("Failed to send notification - monitoring may be compromised", "error", err)
	}
}
