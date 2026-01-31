package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

var (
	ErrNoServiceGroups    = errors.New("config doesn't contain any service groups, this is not allowed")
	ErrInvalidGroupName   = errors.New("invalid service group name length")
	ErrAuthTokenTooLong   = errors.New("auth token too long")
	ErrHeartbeatTooShort  = errors.New("heartbeat frequency too short")
	ErrNoServices         = errors.New("service group has no services")
	ErrInvalidServiceName = errors.New("invalid service name length")
)

type cliArgs struct {
	configPath *string
}

type service struct {
	Name string `toml:"name"`
}

type serviceGroup struct {
	Name             string        `toml:"name"`
	Services         []service     `toml:"services"`
	AuthToken        string        `toml:"auth_token"`
	MaxHeartbeatFreq time.Duration `toml:"max_heartbeat_freq"`
}

type config struct {
	ServiceGroups []serviceGroup `toml:"service_groups"`
}

func handleCliArgs() cliArgs {
	var args cliArgs
	args.configPath = flag.String("config-path", "config.toml", "description")

	flag.Parse()
	return args
}

func createConfigFromPath(configPath string) (config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return config{}, err
	}

	return createConfig(string(data))
}

func createConfig(data string) (config, error) {
	var cfg config
	_, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return cfg, err
	}

	err = validateConfig(&cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, err
}

func validateConfig(cfg *config) error {
	if len(cfg.ServiceGroups) == 0 {
		return ErrNoServiceGroups
	}

	for _, grp := range cfg.ServiceGroups {
		const MinNameLen = 2
		const MaxNameLen = 64
		if len(grp.Name) < MinNameLen || len(grp.Name) > MaxNameLen {
			return fmt.Errorf("%w (min: %d, max: %d): %s", ErrInvalidGroupName, MinNameLen, MaxNameLen, grp.Name)
		}

		const MaxAuthLen = 255
		if len(grp.AuthToken) > MaxAuthLen {
			return fmt.Errorf("%w (max: %d): %s", ErrAuthTokenTooLong, MaxAuthLen, grp.AuthToken)
		}

		const MinHeartbeatFreq = time.Second * 60
		if grp.MaxHeartbeatFreq < MinHeartbeatFreq {
			return fmt.Errorf("%w (min: %v): %v", ErrHeartbeatTooShort, MinHeartbeatFreq, grp.MaxHeartbeatFreq)
		}

		if len(grp.Services) == 0 {
			return fmt.Errorf("%w: %s", ErrNoServices, grp.Name)
		}

		for _, service := range grp.Services {
			if len(service.Name) < MinNameLen || len(service.Name) > MaxNameLen {
				return fmt.Errorf("%w (min: %d, max: %d): %s", ErrInvalidServiceName, MinNameLen, MaxNameLen, service.Name)
			}
		}
	}

	return nil
}

func main() {
	args := handleCliArgs()
	_, err := createConfigFromPath(*args.configPath)
	if err != nil {
		slog.Error("failed to parse toml config", "error", err)
		return
	}
}
