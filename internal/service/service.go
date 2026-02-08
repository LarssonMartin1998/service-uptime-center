// Package service
package service

import (
	"bytes"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"service-uptime-center/internal/app/apperror"
	"service-uptime-center/internal/app/timings"
	"service-uptime-center/notification"
)

type Manager struct {
	cfg    *Config
	lookup map[string]*Service
	mutex  sync.RWMutex
}

type Config struct {
	Services []Service `toml:"services"`
}

type Service struct {
	Name                     string        `toml:"name"`
	HeartbeatTimeoutDuration time.Duration `toml:"heartbeat_timeout_duration"`
	LastPulse                time.Time
	LastProblem              time.Time
	LastSuccessReport        time.Time
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
	Notifiers []string
}

func (m *Manager) StartMonitoring(notificationManager *notification.Manager, instr MonitoringInstructions) {
	go func() {
		for {
			problematic := m.getProblematicServices()
			if len(problematic) > 0 {
				m.handleProblematicServices(notificationManager, instr.Notifiers, problematic)
			}

			time.Sleep(instr.Timings.IncidentsPollFreq)
		}
	}()

	start := time.Now()
	go func() {
		time.Sleep(instr.Timings.SuccessfulReportCooldown)

        if err := notificationManager.Send(instr.Notifiers, notification.SendData{
			Title: "Service Uptime Center running without any issues.",
			Body:  "",
		}); err != nil {
            slog.Error("Cannot send notification, monitoring may be compromised", "error", err)
        } else {
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

		if time.Since(service.LastPulse) >= service.HeartbeatTimeoutDuration {
			problematic = append(problematic, service)
		}
	}
	return problematic
}

func (m *Manager) handleProblematicServices(notificationManager *notification.Manager, notifiers []string, services []*Service) {
    m.mutex.Lock()

    now := time.Now()
    var buf bytes.Buffer

    buf.WriteString("Service Name, Last Pulse, Problem Duration, Overdue")
    for _, service := range services {
        service.LastProblem = now
        problemDuration := time.Since(service.LastPulse)
        overdue := problemDuration - service.HeartbeatTimeoutDuration
        if _, err := fmt.Fprintf(&buf, "%s, %s, %s, %s\n", service.Name, service.LastPulse.String(), problemDuration.String(), overdue.String()); err != nil {
            slog.Error("Failed to write service data to buffer, notification will be missing this data", "service", service.Name, "error", err)
        }
    }

    data := notification.SendData{
        Title: fmt.Sprintf("Problem detected with '%d', services", len(services)),
        Body:  buf.String(),
    }

    m.mutex.Unlock()

    slog.Info("Detected problematic services", "services", services)
    if err := notificationManager.Send(notifiers, data); err != nil {
        slog.Error("Failed to send notification - monitoring may be compromised", "error", err)
    }
}

func (cfg *Config) Validate() error {
	if len(cfg.Services) == 0 {
		return apperror.ErrNoServices
	}

	for _, service := range cfg.Services {
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
