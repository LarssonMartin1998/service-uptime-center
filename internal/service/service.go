// Package service
package service

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	apperror "service-uptime-center/internal/error"
)

type Manager struct {
	cfg    *Config
	lookup map[string]*Service
	mutex  sync.RWMutex
}

type TimingIntervals struct {
	IncidentsPollFreq         time.Duration `toml:"incident_poll_frequency"`
	SuccessfulReportCooldown  time.Duration `toml:"successful_report_cooldown"`
	ProblematicReportCooldown time.Duration `toml:"problematic_Report_cooldown"`
}

type Config struct {
	Timings  TimingIntervals `toml:"timings"`
	Services []Service       `toml:"services"`
}

type Service struct {
	Name                     string        `toml:"name"`
	HeartbeatTimeoutDuration time.Duration `toml:"heartbeat_timeout_duration"`
	NotifiersStr             []string      `toml:"notifiers"`
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

func (m *Manager) StartMonitoring() {
	go func() {
		for {
			categorizedServices := m.categorizeServices()

			if len(categorizedServices.Problematic) > 0 {
				go m.handleProblematicServices(categorizedServices.Problematic)
			}
			if len(categorizedServices.ReadyToReportSuccess) > 0 {
				go m.handleReadyToReportSuccessServices(categorizedServices.ReadyToReportSuccess)
			}

			time.Sleep(m.cfg.Timings.IncidentsPollFreq)
		}
	}()
}

type ServiceCategories struct {
	Problematic          []*Service
	ReadyToReportSuccess []*Service
}

func (m *Manager) categorizeServices() ServiceCategories {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var categories ServiceCategories
	for i := range m.cfg.Services {
		service := &m.cfg.Services[i]

		if time.Since(service.LastPulse) >= service.HeartbeatTimeoutDuration {
			categories.Problematic = append(categories.Problematic, service)
		} else if time.Since(service.LastSuccessReport) >= m.cfg.Timings.SuccessfulReportCooldown &&
			time.Since(service.LastProblem) >= m.cfg.Timings.SuccessfulReportCooldown {
			categories.ReadyToReportSuccess = append(categories.ReadyToReportSuccess, service)
		}
	}
	return categories
}

func (m *Manager) handleProblematicServices(services []*Service) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for _, service := range services {
		service.LastProblem = now
	}
	slog.Info("Detected problematic services", "services", services)
}

func (m *Manager) handleReadyToReportSuccessServices(services []*Service) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for _, service := range services {
		service.LastSuccessReport = now
	}
	slog.Info("Reporting successful services", "services", services)
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

		if len(service.NotifiersStr) == 0 {
			return fmt.Errorf("%w: 'Service=%s'", apperror.ErrNoNotifiers, service.Name)
		}
	}

	return nil
}
