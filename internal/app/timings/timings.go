// Package timings
package timings

import "time"

type Timings struct {
	IncidentsPollFreq        time.Duration `toml:"incident_poll_frequency"`
	SuccessfulReportCooldown time.Duration `toml:"successful_report_cooldown"`
}
