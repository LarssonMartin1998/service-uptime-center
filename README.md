# Service Uptime Center

A lightweight, self-hosted service monitoring system that tracks heartbeats from your services and sends notifications when they go down.

## Features

- **Heartbeat Monitoring**: Services send periodic pulses via HTTP POST
- **Configurable Timeouts**: Set individual timeout thresholds per service
- **Notification Channels**: Email and ntfy.sh
- **Fallback Notifications**: Optional secondary notifiers when primary ones fail
- **Self-Monitoring**: The system monitors itself and reports its own health

## Quick Start

### 1. Configuration

Create a `config.toml` file, you only need to configure the notifiers that you plan on using, the rest can be left blank:

```toml
notifiers = ["mail"]
fallback_notifiers = ["ntfy"]

[notification_settings.mail]
from = "alerts@yourdomain.com"
to = "you@yourdomain.com"

[notification_settings.mail.smtp]
outgoing = "smtp.yourdomain.com"
port = 587
user = "alerts@yourdomain.com"
password_file = "path/to/file"

[notification_settings.ntfy]
server = "https://ntfy.sh"
topic = "service-alerts"
token_file = "path/to/token-file" # optional

[[service_settings.services]]
name = "web-app"
heartbeat_timeout_duration = "12h"

[[service_settings.services]]
name = "api-server"
heartbeat_timeout_duration = "12h"

[time_settings]
incident_poll_frequency = "2h"
successful_report_cooldown = "24h"
```

### 2. Create Password Files

Create an authentication token file:
```bash
echo "your-secret-token" > auth-token.txt
chmod 600 auth-token.txt
```

If using email notifications, create an SMTP password file:
```bash
echo "your-smtp-password" > smtp-password.txt
chmod 600 smtp-password.txt
```

If using ntfy with a token, create a token file:
```bash
echo "your-ntfy-token" > ntfy-token.txt
chmod 600 ntfy-token.txt
```

### 3. Run the Service

```bash
./service-uptime-center --config-path config.toml --pw-file password/file/path.txt --port 8080
```

### 4. Configure Your Services

Have your services send heartbeat pulses:

```bash
curl -X POST http://localhost:8080/api/v1/pulse \
  -H "Authorization: Bearer your-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"service_name": "web-app"}'
```

## API Endpoints

### POST `/api/v1/pulse`
Send a heartbeat for a service.

**Headers:**
- `Authorization: Bearer <token>`
- `Content-Type: application/json`

**Body:**
```json
{
  "service_name": "your-service-name"
}
```

### GET `/api/v1/health`
Check if the monitoring service is running.

## License

MIT License - see LICENSE file for details.
