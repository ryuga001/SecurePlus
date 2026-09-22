package worker

import (
	"context"
	"errors"
	"fmt"

	"dpdp-backend/internal/notification"
)

var ErrUnprocessable = errors.New("notification is unprocessable")

type Mailer interface {
	Send(ctx context.Context, email notification.Email) error
	AdminEmail(ctx context.Context, customerID int) (string, error)
}

type NotificationProcessor struct {
	mailer Mailer
}

func NewNotificationProcessor(mailer Mailer) *NotificationProcessor {
	return &NotificationProcessor{mailer: mailer}
}

func (p *NotificationProcessor) Process(ctx context.Context, msg notification.NotificationMessage) error {
	switch msg.MessageType {
	case notification.MessageTypeEmail:
		return p.email(ctx, msg)
	default:
		return fmt.Errorf("%w: unknown message type %q", ErrUnprocessable, msg.MessageType)
	}
}

func (p *NotificationProcessor) email(ctx context.Context, msg notification.NotificationMessage) error {
	recipients := msg.To

	if len(recipients) == 0 {
		address, err := p.mailer.AdminEmail(ctx, msg.CustomerID)
		if err != nil {
			if errors.Is(err, notification.ErrRecipientNotFound) {
				return fmt.Errorf("%w: %v", ErrUnprocessable, err)
			}

			return err
		}

		recipients = []string{address}
	}

	err := p.mailer.Send(ctx, notification.Email{
		CustomerID: msg.CustomerID,
		Template:   msg.TemplateTitle,
		To:         recipients,
		Vars:       msg.Body,
	})

	if errors.Is(err, notification.ErrTemplateNotFound) {
		return fmt.Errorf("%w: %v", ErrUnprocessable, err)
	}

	return err
}
