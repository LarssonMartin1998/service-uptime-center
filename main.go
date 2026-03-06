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
	"service-uptime-center/notification"
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
		cfg, err := config.Parse(config.YamlFileDecoder[*app.Config], args.ConfigPath)
		cfgChan <- cfgResult{cfg: cfg, err: err}
	}()
	pwRes := <-pwChan
	cfgRes := <-cfgChan

	if pwRes.err != nil {
		slog.Error("failed to read password file", "path", args.PwFilePath, "error", pwRes.err)
		os.Exit(apperror.CodeFailedReadingPasswordFile)
	}
	if cfgRes.err != nil {
		slog.Error("failed to parse yaml config", "error", cfgRes.err)
		os.Exit(apperror.CodeInvalidConfig)
	}
	pw := pwRes.pw
	cfg := cfgRes.cfg

	managerLocator, err := app.NewManagerLocator(cfgRes.cfg)
	if err != nil {
		slog.Error("failed to create manager locator from config", "error", err)
		os.Exit(apperror.CodeInvalidConfig)
	}

	allNotifiers := append(cfg.Notifiers, cfg.FallbackNotifiers...)
	slog.Info("running startup authentication tests", "notifiers", allNotifiers)
	authResults := managerLocator.NotificationManager.TestAuth(allNotifiers)
	for _, r := range authResults {
		if r.Err != nil {
			slog.Error("startup auth test failed", "protocol", r.Protocol, "error", r.Err)
			os.Exit(apperror.CodeAuthTestFailed)
		}
	}

	server.SetupEndpoints(pw, managerLocator.ServiceManager, managerLocator.NotificationManager, allNotifiers)
	managerLocator.ServiceManager.StartMonitoring(managerLocator.NotificationManager, service.MonitoringInstructions{
		Timings: &cfg.Timings,
		Notifiers: notification.ProtocolTargets{
			Primary:  cfg.Notifiers,
			Fallback: cfg.FallbackNotifiers,
		},
	})

	server.ServeAndAwaitTermination(args.Port)
}
