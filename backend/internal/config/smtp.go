package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	TransportSMTP   = "smtp"
	TransportResend = "resend"
)

type SMTP struct {
	Transport string
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
}

func loadSMTP() SMTP {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))

	return SMTP{
		Transport: strings.ToLower(strings.TrimSpace(envString("EMAIL_TRANSPORT", TransportSMTP))),
		Host:      os.Getenv("SMTP_HOST"),
		Port:      port,
		Username:  os.Getenv("SMTP_USERNAME"),
		Password:  os.Getenv("SMTP_PASSWORD"),
		FromEmail: os.Getenv("SMTP_FROM_EMAIL"),
		FromName:  os.Getenv("SMTP_FROM_NAME"),
	}
}

func (s SMTP) Validate() error {
	switch s.Transport {
	case TransportSMTP:
		return nil
	case TransportResend:
		if s.Password == "" {
			return errors.New("SMTP_PASSWORD is required and holds the Resend API key when EMAIL_TRANSPORT=resend")
		}
		if s.FromEmail == "" {
			return errors.New("SMTP_FROM_EMAIL is required when EMAIL_TRANSPORT=resend")
		}

		return nil
	}

	return fmt.Errorf("unsupported email transport: %q", s.Transport)
}
