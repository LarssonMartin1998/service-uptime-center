{
  config,
  lib,
  pkgs,
  ...
}:

with lib;

let
  cfg = config.services.service-uptime-center;
  configFile = pkgs.writeText "service-uptime-center.yaml" (
    lib.generators.toYAML { } {
      notifiers = cfg.notifiers;
      fallback_notifiers = cfg.fallbackNotifiers;

      time_settings = {
        incident_poll_frequency = cfg.timeSettings.incidentPollFrequency;
        successful_report_cooldown = cfg.timeSettings.successfulReportCooldown;
        problematic_report_cooldown = cfg.timeSettings.problematicReportCooldown;
      };

      notification_settings = cfg.notificationSettings;

      service_settings = {
        services = map (service: {
          name = service.name;
          heartbeat_timeout_duration = service.heartbeatTimeoutDuration;
        }) cfg.services;
      };
    }
  );
in
{
  options.services.service-uptime-center = {
    enable = mkEnableOption "service-uptime-center monitoring service";

    package = mkOption {
      type = types.package;
      description = "The service-uptime-center package to use";
    };

    port = mkOption {
      type = types.port;
      default = 8080;
      description = "The port to run the server on";
    };

    notifiers = mkOption {
      type = types.listOf types.str;
      default = [ "mail" ];
      description = "List of notifiers to use";
    };

    fallbackNotifiers = mkOption {
      type = types.listOf types.str;
      default = [ ];
      description = "List of fallback notifiers";
    };

    timeSettings = {
      incidentPollFrequency = mkOption {
        type = types.str;
        default = "30m";
        description = "How often to poll for incidents";
      };

      successfulReportCooldown = mkOption {
        type = types.str;
        default = "1h";
        description = "Cooldown period for successful reports";
      };

      problematicReportCooldown = mkOption {
        type = types.str;
        default = "1h";
        description = "Cooldown period for problematic reports";
      };
    };

    notificationSettings = mkOption {
      type = types.attrsOf types.attrs;
      default = { };
      description = "Notification settings for each notifier";
      example = {
        mail = {
          from = "Service Uptime Center <me@example.com>";
          to = "me@example.com";
          smtp = {
            outgoing = "smtp.example.com";
            port = 587;
            user = "me@example.com";
            password_file = "smtp_pw";
          };
        };
      };
    };

    services = mkOption {
      type = types.listOf (
        types.submodule {
          options = {
            name = mkOption {
              type = types.str;
              description = "Name of the service to monitor";
            };
            heartbeatTimeoutDuration = mkOption {
              type = types.str;
              default = "30m";
              description = "Timeout duration for service heartbeat";
            };
          };
        }
      );
      default = [ ];
      description = "List of services to monitor";
    };

    pwFilePath = mkOption {
      type = types.str;
      description = "Path to the file that contains the auth token to be used";
    };
  };

  config = mkIf cfg.enable {
    users.users.service-uptime-center = {
      isSystemUser = true;
      group = "service-uptime-center";
      home = "/var/lib/service-uptime-center";
      createHome = true;
    };

    users.groups.service-uptime-center = { };

    systemd.services.service-uptime-center = {
      description = "Service Uptime Center";
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" ];

      serviceConfig = {
        Type = "simple";
        User = "service-uptime-center";
        Group = "service-uptime-center";
        WorkingDirectory = "/var/lib/service-uptime-center";
        ExecStart = "${cfg.package}/bin/service-uptime-center --config-path ${configFile} --port ${toString cfg.port} --pw-file ${cfg.pwFilePath}";
        Restart = "always";
        RestartSec = "10s";
      };
    };
  };
}
