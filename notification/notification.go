// Package notification
package notification

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

var (
	ErrInvalidProtocol         = errors.New("invalid notification protocol provieded")
	ErrNotificationFailed      = errors.New("notification failed")
	ErrDuplicateNotifyProtocol = errors.New("duplicate notification protocol")
	ErrDuplicateFallback       = errors.New("fallback overlaps with primary notifiers")
)

type SendData struct {
	Title string
	Body  string
}

type ProtocolTargets struct {
	Primary  []string
	Fallback []string
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
	if len(notifiers) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(notifiers))
	for _, protocol := range notifiers {
		if _, ok := seen[protocol]; ok {
			return fmt.Errorf("%w: %s", ErrDuplicateNotifyProtocol, protocol)
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
	failures := p.sendAll(protocols, data)
	if len(failures) != 0 {
		logSendFailures(failures)
		return formatSendFailures(failures)
	}

	return nil
}

func (p *Manager) SendWithFallback(targets ProtocolTargets, data SendData) error {
	failures := p.sendAll(targets.Primary, data)
	if len(failures) > 0 && len(targets.Fallback) > 0 {
		fallbackData := SendData{
			Title: "Fallback notification: primary notifier failed",
			Body:  formatFallbackBody(failures, data),
		}
		fallbackFailures := p.sendAll(targets.Fallback, fallbackData)
		for _, fallbackFailure := range fallbackFailures {
			failures = append(failures, sendFailure{
				protocol: fallbackFailure.protocol + " (fallback)",
				err:      fmt.Errorf("%w (%s): %v", ErrNotificationFailed, fallbackFailure.protocol, fallbackFailure.err),
			})
		}
	}

	if len(failures) != 0 {
		logSendFailures(failures)
		return formatSendFailures(failures)
	}

	return nil
}

type sendFailure struct {
	protocol string
	err      error
}

func (p *Manager) sendAll(protocols []string, data SendData) []sendFailure {
	var failures []sendFailure
	for _, protocol := range protocols {
		entry, ok := p.protocols[protocol]
		if !ok {
			failures = append(failures, sendFailure{protocol: protocol, err: ErrInvalidProtocol})
			continue
		}

		if err := entry.notify.send(data); err != nil {
			failures = append(failures, sendFailure{protocol: protocol, err: err})
		}
	}
	return failures
}

func formatFallbackBody(failures []sendFailure, data SendData) string {
	var b strings.Builder
	b.WriteString("One or more notifications failed to send.\n")
	for _, failure := range failures {
		b.WriteString("- ")
		b.WriteString(failure.protocol)
		b.WriteString(": ")
		b.WriteString(failure.err.Error())
		b.WriteString("\n")
	}
	b.WriteString("\nOriginal title: ")
	b.WriteString(data.Title)
	b.WriteString("\n\nOriginal body:\n")
	b.WriteString(data.Body)
	return b.String()
}

func formatSendFailures(failures []sendFailure) error {
	var b strings.Builder
	for _, failure := range failures {
		b.WriteString("\n- ")
		b.WriteString(failure.protocol)
		b.WriteString(": ")
		b.WriteString(failure.err.Error())
	}
	return fmt.Errorf("%w:%s", ErrNotificationFailed, b.String())
}

func logSendFailures(failures []sendFailure) {
	for _, failure := range failures {
		slog.Error("notification failed", "protocol", failure.protocol, "error", failure.err)
	}
}
