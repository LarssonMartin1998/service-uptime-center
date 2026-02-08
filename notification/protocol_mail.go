package notification

import (
	"log/slog"

	gomail "gopkg.in/gomail.v2"
)

type SMTPContext struct {
	Outgoing string `toml:"outgoing"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
}

type MailConfig struct {
	From string      `toml:"from"`
	To   string      `toml:"to"`
	SMTP SMTPContext `toml:"smtp"`
}

func (m *MailConfig) Validate() error {
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
	dialer := gomail.NewDialer(smtp.Outgoing, smtp.Port, smtp.User, smtp.Password)
	if err := dialer.DialAndSend(message); err != nil {
		slog.Error("failed to send mail.", "from", smtp.User, "to", m.cfg.To)
		return err
	}

	return nil
}
