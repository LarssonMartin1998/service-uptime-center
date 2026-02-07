// Package service
package service

import "time"

type Mapper map[string]*Service

type Service struct {
	Name                     string        `toml:"name"`
	HeartbeatTimeoutDuration time.Duration `toml:"heartbeat_timeout_duration"`
	LastPulse                time.Time
}
}
