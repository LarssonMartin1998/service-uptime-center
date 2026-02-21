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
	protocols map[string]NotifyProtocol
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
		if _, ok := manager.protocols[protocol]; !ok {
			return ErrInvalidProtocol
		}
		switch protocol {
		case "mail":
			if err := m.Mail.Validate(); err != nil {
				return err
			}
		case "ntfy":
			if err := m.Ntfy.Validate(); err != nil {
				return err
			}
		}
	}

	return nil
}

func NewManager(cfg *ManagerConfig) *Manager {
	return &Manager{
		protocols: map[string]NotifyProtocol{
			"mail": newMailNotifier(&cfg.Mail),
			"ntfy": newNtfyNotifier(&cfg.Ntfy),
		},
	}
}

func (p *Manager) Send(protocols []string, data SendData) error {
	for _, protocol := range protocols {
		found, ok := p.protocols[protocol]
		if !ok {
			return ErrInvalidProtocol
		}

		if err := found.send(data); err != nil {
			return err
		}
	}

	return nil
}
