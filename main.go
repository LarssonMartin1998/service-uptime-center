package main

import (
	"log/slog"
	"os"

	"service-uptime-center/config"
	"service-uptime-center/internal/app"
	"service-uptime-center/internal/app/apperror"
	"service-uptime-center/internal/app/util"
	"service-uptime-center/internal/cli"
	"service-uptime-center/internal/server"
	"service-uptime-center/internal/service"
)

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
		pw, err := util.ParsePasswordFile(args.PwFilePath)
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
