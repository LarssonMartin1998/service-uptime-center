// Package service
package service

import "time"

type Mapper map[string]*Service

type Group struct {
	Services         []Service     `toml:"services"`
	MaxHeartbeatFreq time.Duration `toml:"max_heartbeat_freq"`
}

type Service struct {
	Name      string `toml:"name"`
	LastPulse time.Time
}
