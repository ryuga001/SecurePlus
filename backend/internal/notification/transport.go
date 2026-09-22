package notification

import (
	"context"
	"strings"

	"dpdp-backend/internal/config"
)

type Message struct {
	To      []string
	Subject string
	HTML    string
}

type Transport interface {
	Send(ctx context.Context, msg Message) error
}

func newTransport(cfg config.SMTP) Transport {
	switch cfg.Transport {
	case config.TransportResend:
		return NewResendSender(cfg)
	default:
		return NewSMTPSender(cfg)
	}
}

func formatAddress(name, email string) string {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)

	if name == "" {
		return email
	}

	return name + " <" + email + ">"
}
