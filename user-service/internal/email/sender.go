package email

import (
	"fmt"
	"log/slog"

	"gopkg.in/gomail.v2"
)

type Sender interface {
	SendPasswordResetCode(email, code string, ttlMinutes int) error
}

type SMTPSender struct {
	host     string
	port     int
	user     string
	password string
	from     string
	fromName string
	tls      bool
	logger   *slog.Logger
}

func NewSMTPSender(host string, port int, user, password, from, fromName string, tls bool, logger *slog.Logger) *SMTPSender {
	return &SMTPSender{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		from:     from,
		fromName: fromName,
		tls:      tls,
		logger:   logger,
	}
}

func (s *SMTPSender) SendPasswordResetCode(email, code string, ttlMinutes int) error {
	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.fromName, s.from))
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Password reset code")
	m.SetBody("text/plain", fmt.Sprintf("Your password reset code: %s\n\nThis code expires in %d minutes.\n\nIf you didn't request this, please ignore this email.", code, ttlMinutes))

	d := gomail.NewDialer(s.host, s.port, s.user, s.password)
	if !s.tls {
		d.TLSConfig = nil
	}

	if err := d.DialAndSend(m); err != nil {
		s.logger.Error("Failed to send email", "error", err, "email", email)
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("Password reset code sent", "email", email)
	return nil
}
