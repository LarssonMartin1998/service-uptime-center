// Package service
package service

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type Manager struct {
	services []Service
	lookup   map[string]*Service
	mutex    sync.RWMutex
}

type Service struct {
	Name                     string        `toml:"name"`
	HeartbeatTimeoutDuration time.Duration `toml:"heartbeat_timeout_duration"`
	LastPulse                time.Time
}

func NewManager(services []Service) (*Manager, error) {
	now := time.Now()
	lookup := make(map[string]*Service, len(services))

	for i := range services {
		services[i].LastPulse = now

		_, ok := lookup[services[i].Name]
		if ok {
			return nil, fmt.Errorf("found services with the same name when creating lookup, this is not allowed")
		}

		lookup[services[i].Name] = &services[i]
	}

	return &Manager{
		services: services,
		lookup:   lookup,
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

func (m *Manager) StartMonitoring(pollFreq time.Duration) {
	go func() {
		for {
			problematicServices := m.getProblematicServices()
			if len(problematicServices) > 0 {
				go handleProblematicServices(problematicServices)
			}
			time.Sleep(pollFreq)
		}
	}()
}

func (m *Manager) getProblematicServices() []*Service {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var problematic []*Service
	for i := range m.services {
		if time.Since(m.services[i].LastPulse) >= m.services[i].HeartbeatTimeoutDuration {
			problematic = append(problematic, &m.services[i])
		}
	}
	return problematic
}

func handleProblematicServices(services []*Service) {
	slog.Info("Detected problematic services", "services", services)
}
