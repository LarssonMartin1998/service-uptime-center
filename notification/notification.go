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
}

func (m *ManagerConfig) Validate() error {
	return m.Mail.Validate()
}

func NewManager(cfg *ManagerConfig) *Manager {
	return &Manager{
		protocols: map[string]NotifyProtocol{
			"mail": newMailNotifier(&cfg.Mail),
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
