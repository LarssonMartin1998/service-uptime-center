// Package notification
package notification

import (
	"errors"
)

var (
	ErrInvalidProtocol = errors.New("invalid notification protocol provieded")
)

type SendData struct {
	Title string
	Body  string
}

type NotifyProtocol interface {
	send(SendData) error
}

type Manager struct {
	protocols map[string]protocolEntry
}

type protocolEntry struct {
	notify   NotifyProtocol
	validate func() error
}

type ManagerConfig struct {
	Mail MailConfig `toml:"mail"`
	Ntfy NtfyConfig `toml:"ntfy"`
}

func (m *ManagerConfig) ValidateFor(notifiers []string, manager *Manager) error {
	seen := make(map[string]struct{}, len(notifiers))
	for _, protocol := range notifiers {
		if _, ok := seen[protocol]; ok {
			continue
		}
		seen[protocol] = struct{}{}
		entry, ok := manager.protocols[protocol]
		if !ok {
			return ErrInvalidProtocol
		}
		if entry.validate == nil {
			return ErrInvalidProtocol
		}
		if err := entry.validate(); err != nil {
			return err
		}
	}

	return nil
}

func NewManager(cfg *ManagerConfig) *Manager {
	return &Manager{
		protocols: map[string]protocolEntry{
			"mail": {
				notify:   newMailNotifier(&cfg.Mail),
				validate: cfg.Mail.Validate,
			},
			"ntfy": {
				notify:   newNtfyNotifier(&cfg.Ntfy),
				validate: cfg.Ntfy.Validate,
			},
		},
	}
}

func (p *Manager) Send(protocols []string, data SendData) error {
	for _, protocol := range protocols {
		entry, ok := p.protocols[protocol]
		if !ok {
			return ErrInvalidProtocol
		}

		if err := entry.notify.send(data); err != nil {
			return err
		}
	}

	return nil
}
}
