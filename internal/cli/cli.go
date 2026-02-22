// Package cli
package cli

import (
	"flag"
	"log/slog"
	"math"
	"os"
	"service-uptime-center/internal/app/apperror"
)

type CliArgs struct {
	PwFilePath    string
	ConfigPath string
	Port       uint16
}

func ParseArgs() *CliArgs {
	configPath := flag.String("config-path", "config.toml", "Path to the configuration file, defaults to './config.toml'")
	pwFilePath := flag.String("pw-file", "", "Path to the password file, if run without a password file, auth token middleware will be disabled.")

	portFlag := *flag.Uint64("port", 8080, "The port that the HTTP server will listen on")
	flag.Parse()

	if portFlag > math.MaxUint16 {
		slog.Error("--port flag is too high.", "max value", math.MaxUint16, "got", portFlag)
		os.Exit(apperror.CodeInvalidCliArgument)
	}

	return &CliArgs{
		PwFilePath:    *pwFilePath,
		ConfigPath: *configPath,
		Port:       uint16(portFlag),
	}
}
