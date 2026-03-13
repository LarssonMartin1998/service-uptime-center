// Package timings
package timings

import "time"

type Timings struct {
	IncidentsPollFreq         time.Duration `yaml:"incident_poll_frequency"`
	SuccessfulReportCooldown  time.Duration `yaml:"successful_report_cooldown"`
	ProblematicReportCooldown time.Duration `yaml:"problematic_report_cooldown"`
}
