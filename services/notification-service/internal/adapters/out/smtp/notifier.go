package smtp

import (
	"context"
	"fmt"
	"net"
	"net/smtp"

	"github.com/fiapx/notification-service/internal/domain"
)

type SMTPNotifier struct {
	host string
	port string
	from string
}

func NewSMTPNotifier(host, port, from string) *SMTPNotifier {
	return &SMTPNotifier{host: host, port: port, from: from}
}

func (n *SMTPNotifier) Send(ctx context.Context, notif domain.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	msg := buildMessage(n.from, notif.Email, notif.Subject, notif.Body)
	addr := net.JoinHostPort(n.host, n.port)
	return smtp.SendMail(addr, nil, n.from, []string{notif.Email}, []byte(msg))
}

func buildMessage(from, to, subject, body string) string {
	headers := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n",
		from, to, subject,
	)
	return headers + body
}
