// Package app
package app

import (
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
	if err := cfg.Notification.ValidateFor(cfg.Notifiers, notificationManager); err != nil {
		return nil, err
	}

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
	Notification notification.ManagerConfig `toml:"notification_settings"`
	Service      service.Config             `toml:"service_settings"`
	Timings      timings.Timings            `toml:"time_settings"`
	Notifiers    []string                   `toml:"notifiers"`
}

func (a *Config) Validate() error {
	if len(a.Notifiers) == 0 {
		return apperror.ErrNoNotifiers
	}

	if err := a.Service.Validate(); err != nil {
		return err
	}

	return nil
}
