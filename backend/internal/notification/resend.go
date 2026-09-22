package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"dpdp-backend/internal/config"
)

const (
	ResendEndpoint = "https://api.resend.com/emails"

	resendTimeout          = 30 * time.Second
	resendErrorBodyMaxSize = 512
)

type ResendSender struct {
	client   *http.Client
	endpoint string
	apiKey   string
	from     string
}

func NewResendSender(cfg config.SMTP) *ResendSender {
	return &ResendSender{
		client:   &http.Client{Timeout: resendTimeout},
		endpoint: ResendEndpoint,
		apiKey:   cfg.Password,
		from:     formatAddress(cfg.FromName, cfg.FromEmail),
	}
}

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (s *ResendSender) Send(ctx context.Context, msg Message) error {
	payload, err := json.Marshal(resendRequest{
		From:    s.from,
		To:      msg.To,
		Subject: msg.Subject,
		HTML:    msg.HTML,
	})
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	request.Header.Set("Authorization", "Bearer "+s.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("resend rejected the message: %s: %s",
			response.Status, readCapped(response.Body, resendErrorBodyMaxSize))
	}

	io.Copy(io.Discard, response.Body)

	return nil
}

func readCapped(body io.Reader, limit int64) string {
	raw, err := io.ReadAll(io.LimitReader(body, limit))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(raw))
}
