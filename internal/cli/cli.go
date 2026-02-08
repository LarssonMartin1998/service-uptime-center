// Package cli
package cli

import (
	"flag"
	"log/slog"
	"math"
	"os"
	"service-uptime-center/internal/app/apperror"
	"time"
)

type CliArgs struct {
	ConfigPath    string
	Port          uint16
	PwFilePath    string
	SleepDuration time.Duration
}

func ParseArgs() *CliArgs {
	configPath := flag.String("config-path", "config.toml", "Path to the configuration file, defaults to './config.toml'")
	pwFilePath := flag.String("pw-file", "", "Path to the password file, if run without a password file, auth token middleware will be disabled.")
	sleepDurationStr := flag.String("sleep-duration", "3h", "How frequently to check for missing pulses from services.")

	portFlag := *flag.Uint64("port", 8080, "The port that the HTTP server will listen on")
	flag.Parse()

	if portFlag > math.MaxUint16 {
		slog.Error("--port flag is too high.", "max value", math.MaxUint16, "got", portFlag)
		os.Exit(apperror.CodeInvalidCliArgument)
	}

	duration, err := time.ParseDuration(*sleepDurationStr)
	if err != nil {
		slog.Error("--sleep-duration has invalid format (supported format: ns, us, ms, s, m, h)")
		os.Exit(apperror.CodeInvalidCliArgument)
	}

	return &CliArgs{
		ConfigPath:    *configPath,
		Port:          uint16(portFlag),
		PwFilePath:    *pwFilePath,
		SleepDuration: duration,
	}
}
