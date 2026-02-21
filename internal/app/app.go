// Package app
package app

import (
	"fmt"

	"service-uptime-center/internal/app/apperror"
	"service-uptime-center/internal/app/timings"
	"service-uptime-center/internal/service"
	"service-uptime-center/notification"
)

type managerLocator struct {
	NotificationManager *notification.Manager
	ServiceManager      *service.Manager
}

func NewManagerLocator(cfg *Config) (*managerLocator, error) {
	notificationManager := notification.NewManager(&cfg.Notification)

	serviceManager, err := service.NewManager(&cfg.Service)
	if err != nil {
		return nil, err
	}

	return &managerLocator{
		NotificationManager: notificationManager,
		ServiceManager:      serviceManager,
	}, nil
}

type Config struct {
	Notification      notification.ManagerConfig `toml:"notification_settings"`
	Service           service.Config             `toml:"service_settings"`
	Timings           timings.Timings            `toml:"time_settings"`
	Notifiers         []string                   `toml:"notifiers"`
	FallbackNotifiers []string                   `toml:"fallback_notifiers"`
}

func (a *Config) Validate() error {
	if len(a.Notifiers) == 0 {
		return apperror.ErrNoNotifiers
	}

	if err := a.Service.Validate(); err != nil {
		return err
	}

	notificationManager := notification.NewManager(&a.Notification)
	if err := a.Notification.ValidateFor(a.Notifiers, notificationManager); err != nil {
		return err
	}
	if err := a.Notification.ValidateFor(a.FallbackNotifiers, notificationManager); err != nil {
		return err
	}

	if len(a.FallbackNotifiers) != 0 {
		seen := make(map[string]struct{}, len(a.Notifiers))
		for _, protocol := range a.Notifiers {
			seen[protocol] = struct{}{}
		}
		for _, protocol := range a.FallbackNotifiers {
			if _, ok := seen[protocol]; ok {
				return fmt.Errorf("%w: %s", notification.ErrDuplicateFallback, protocol)
			}
		}
	}

	return nil
}
