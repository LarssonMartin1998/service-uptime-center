package notification

import (
	"fmt"
	"log/slog"
	"service-uptime-center/internal/app/util"

	gomail "gopkg.in/gomail.v2"
)

var (
	ErrMissingMailConfigProperty = fmt.Errorf("missing required property in mail config")
	ErrInvalidSMTPPort           = fmt.Errorf("")
)

type SMTPContext struct {
	Outgoing     string `toml:"outgoing"`
	Port         int    `toml:"port"`
	User         string `toml:"user"`
	PasswordFile string `toml:"password_file"`
	password     string
}

type MailConfig struct {
	From string      `toml:"from"`
	To   string      `toml:"to"`
	SMTP SMTPContext `toml:"smtp"`
}

func (m *MailConfig) Validate() error {
	if len(m.From) == 0 {
		return fmt.Errorf("%w: From", ErrMissingMailConfigProperty)
	}

	if len(m.To) == 0 {
		return fmt.Errorf("%w: To", ErrMissingMailConfigProperty)
	}

	if len(m.SMTP.Outgoing) == 0 {
		return fmt.Errorf("%w: SMTP.Outgoing", ErrMissingMailConfigProperty)
	}

	if m.SMTP.Port <= 0 || m.SMTP.Port > 65535 {
		return fmt.Errorf("%w: %d", ErrInvalidSMTPPort, m.SMTP.Port)
	}

	if len(m.SMTP.User) == 0 {
		return fmt.Errorf("%w: SMTP.User", ErrMissingMailConfigProperty)
	}

	if len(m.SMTP.PasswordFile) == 0 {
		return fmt.Errorf("%w: SMTP.PasswordFile", ErrMissingMailConfigProperty)
	}

	if pw, err := util.ParsePasswordFile(m.SMTP.PasswordFile); err != nil {
		return err
	} else {
		m.SMTP.password = pw
	}

	return nil
}

type mailNotifier struct {
	cfg *MailConfig
}

func newMailNotifier(cfg *MailConfig) *mailNotifier {
	return &mailNotifier{
		cfg,
	}
}

func (m *mailNotifier) send(data SendData) error {
	message := gomail.NewMessage()

	message.SetHeader("From", m.cfg.From)
	message.SetHeader("To", m.cfg.To)
	message.SetHeader("Subject", data.Title)
	message.SetBody("text/plain", data.Body)

	smtp := &m.cfg.SMTP
	dialer := gomail.NewDialer(smtp.Outgoing, smtp.Port, smtp.User, smtp.password)
	if err := dialer.DialAndSend(message); err != nil {
		slog.Error("failed to send mail.", "from", smtp.User, "to", m.cfg.To)
		return err
	}

	return nil
}
