package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"service-uptime-center/internal/cli"
	"service-uptime-center/internal/config"
	apperrors "service-uptime-center/internal/error"
	"service-uptime-center/internal/server"
	"service-uptime-center/internal/service"
)

func parsePasswordFile(path string) (string, error) {
	if len(path) == 0 {
		slog.Warn("Running without a password file, this is supported but might not be what you intended to do, see --help for more info")
		return "", nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	pw := strings.TrimSpace(string(data))

	const MaxPasswordLen = 255
	if len(pw) > MaxPasswordLen {
		return "", fmt.Errorf("%w (max: %d)", apperrors.ErrPasswordTooLong, MaxPasswordLen)
	}

	return pw, nil
}

func main() {
	args := cli.ParseArgs()

	cfg, err := config.Parse(config.TomlFileDecoder, args.ConfigPath)
	if err != nil {
		slog.Error("failed to parse toml config", "error", err)
		os.Exit(apperrors.CodeInvalidConfig)
	}

	if pw, err := parsePasswordFile(args.PwFilePath); err != nil {
		slog.Error("failed to read password file", "path", args.PwFilePath, "error", err)
		os.Exit(apperrors.CodeFailedReadingPasswordFile)
	} else {
		serviceManager, err := service.NewManager(cfg.Services)
		if err != nil {
			slog.Error("failed to create service mapper from config", "error", err)
			os.Exit(apperrors.CodeInvalidConfig)
		}

		server.SetupEndpoints(pw, serviceManager)
		serviceManager.StartMonitoring(cfg.IncidentsPollFreq)
	}

	server.ServeAndAwaitTermination(args.Port)
}
