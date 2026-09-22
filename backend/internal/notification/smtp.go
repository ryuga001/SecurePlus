package notification

import (
	"context"

	"github.com/wneessen/go-mail"
)

func (s *Service) deliver(ctx context.Context, to []string, subject, body string) error {
	options := []mail.Option{
		mail.WithPort(s.smtp.Port),
		mail.WithTLSPolicy(mail.NoTLS),
		mail.WithSMTPAuth(mail.SMTPAuthNoAuth),
	}

	if s.smtp.Username != "" {
		options = append(options,
			mail.WithTLSPolicy(mail.TLSOpportunistic),
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(s.smtp.Username),
			mail.WithPassword(s.smtp.Password),
		)
	}

	client, err := mail.NewClient(s.smtp.Host, options...)
	if err != nil {
		return err
	}
	defer client.Close()

	message := mail.NewMsg()

	if err := message.FromFormat(s.smtp.FromName, s.smtp.FromEmail); err != nil {
		return err
	}
	if err := message.To(to...); err != nil {
		return err
	}

	message.Subject(subject)
	message.SetBodyString(mail.TypeTextHTML, body)

	return client.DialAndSendWithContext(ctx, message)
}
