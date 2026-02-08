package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"service-uptime-center/config"
	"service-uptime-center/internal/app"
	"service-uptime-center/internal/app/apperror"
	"service-uptime-center/internal/cli"
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
	if len(pw) == 0 {
		return "", apperror.ErrPasswordFileIsEmpty
	}

	const MaxPasswordLen = 255
	if len(pw) > MaxPasswordLen {
		return "", fmt.Errorf("%w (max: %d)", apperror.ErrPasswordTooLong, MaxPasswordLen)
	}

	return pw, nil
}

func main() {
	args := cli.ParseArgs()

	type pwResult struct {
		pw  string
		err error
	}
	type cfgResult struct {
		cfg *app.Config
		err error
	}
	pwChan := make(chan pwResult, 1)
	cfgChan := make(chan cfgResult, 1)

	go func() {
		pw, err := parsePasswordFile(args.PwFilePath)
		pwChan <- pwResult{pw: pw, err: err}
	}()
	go func() {
		cfg, err := config.Parse(config.TomlFileDecoder[*app.Config], args.ConfigPath)
		cfgChan <- cfgResult{cfg: cfg, err: err}
	}()
	pwRes := <-pwChan
	cfgRes := <-cfgChan

	if pwRes.err != nil {
		slog.Error("failed to read password file", "path", args.PwFilePath, "error", pwRes.err)
		os.Exit(apperror.CodeFailedReadingPasswordFile)
	}
	if cfgRes.err != nil {
		slog.Error("failed to parse toml config", "error", cfgRes.err)
		os.Exit(apperror.CodeInvalidConfig)
	}
	pw := pwRes.pw
	cfg := cfgRes.cfg

	managerLocator, err := app.NewManagerLocator(cfgRes.cfg)
	if err != nil {
		slog.Error("failed to create manager locator from config", "error", err)
		os.Exit(apperror.CodeInvalidConfig)
	}

	server.SetupEndpoints(pw, managerLocator.ServiceManager)
	managerLocator.ServiceManager.StartMonitoring(managerLocator.NotificationManager, service.MonitoringInstructions{
		Timings:   &cfg.Timings,
		Notifiers: cfg.Notifiers,
	})

	server.ServeAndAwaitTermination(args.Port)
}
