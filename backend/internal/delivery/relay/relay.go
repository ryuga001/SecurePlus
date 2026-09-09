package relay

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"dpdp-backend/internal/config"
	"dpdp-backend/internal/delivery"
	deliveryutils "dpdp-backend/internal/delivery/utils"
)

type Relay struct {
	cfg      config.Relay
	resolver *Resolver
}

func NewRelay(cfg config.Relay) *Relay {
	return &Relay{cfg: cfg, resolver: NewResolver(cfg.DNSTimeout, cfg.IPv6, cfg.MXOverride)}
}

func (r *Relay) Deliver(
	ctx context.Context,
	msg delivery.EmailMessage,
	cfg delivery.TenantConfig,
	sink func(delivery.Attempt),
) ([]delivery.RecipientResult, error) {
	signed, err := Sign(msg.Raw, cfg)
	if err != nil {
		return nil, err
	}

	results := make(map[string]delivery.RecipientResult, len(msg.Recipients))

	for _, recipient := range msg.Recipients {
		results[recipient] = delivery.RecipientResult{
			Email:  recipient,
			Domain: delivery.DomainOf(recipient),
			Status: deliveryutils.StatusProcessing,
		}
	}

	backoff := r.cfg.BackoffInitial

	for attempt := 1; attempt <= r.cfg.MaxAttempts; attempt++ {
		pending := pendingRecipients(results)
		if len(pending) == 0 {
			break
		}

		r.attempt(ctx, attempt, msg, signed, pending, results, sink)

		if len(pendingRecipients(results)) == 0 || attempt == r.cfg.MaxAttempts {
			break
		}

		if err := sleep(ctx, backoff); err != nil {
			markInterrupted(results)
			break
		}

		backoff = NextBackoff(backoff, r.cfg.BackoffMultiplier, r.cfg.BackoffMax)
	}

	return finalize(results), nil
}

func (r *Relay) attempt(
	ctx context.Context,
	number int,
	msg delivery.EmailMessage,
	signed []byte,
	pending []string,
	results map[string]delivery.RecipientResult,
	sink func(delivery.Attempt),
) {
	for domain, recipients := range delivery.GroupByDomain(pending) {
		started := time.Now()
		outcome := r.deliverDomain(ctx, msg.From, domain, recipients, signed)

		if sink != nil {
			sink(delivery.Attempt{
				Number:     number,
				StartedAt:  started,
				FinishedAt: time.Now(),
				MXHost:     outcome.host,
				TLS:        outcome.tls,
				SMTPCode:   outcome.code,
				Error:      outcome.errorText,
			})
		}

		for recipient, result := range outcome.recipients {
			results[recipient] = result
		}
	}
}

type domainOutcome struct {
	host       string
	tls        string
	code       int
	errorText  string
	recipients map[string]delivery.RecipientResult
}

func (r *Relay) deliverDomain(
	ctx context.Context,
	from, domain string,
	recipients []string,
	signed []byte,
) domainOutcome {
	outcome := domainOutcome{tls: deliveryutils.TLSNone, recipients: map[string]delivery.RecipientResult{}}

	destinations, err := r.resolver.Destinations(ctx, domain)
	if err != nil {
		outcome.errorText = err.Error()
		permanent := errors.Is(err, deliveryutils.ErrNoDestination)
		markDomain(outcome.recipients, recipients, domain, 0, err.Error(), permanent)

		return outcome
	}

	var lastErr error

	for _, destination := range destinations {
		outcome.host = destination.Host

		result, err := r.send(ctx, destination, from, recipients, signed)
		if err == nil {
			outcome.tls = result.tls
			outcome.code = result.code

			for recipient, value := range result.recipients {
				outcome.recipients[recipient] = value
			}

			return outcome
		}

		lastErr = err

		if code, permanent := Classify(err); permanent {
			outcome.code = code
			outcome.errorText = err.Error()
			markDomain(outcome.recipients, recipients, domain, code, err.Error(), true)

			return outcome
		}
	}

	code, _ := Classify(lastErr)
	outcome.code = code
	outcome.errorText = lastErr.Error()
	markDomain(outcome.recipients, recipients, domain, code, lastErr.Error(), false)

	return outcome
}

type sendResult struct {
	tls        string
	code       int
	recipients map[string]delivery.RecipientResult
}

func (r *Relay) send(
	ctx context.Context,
	destination Destination,
	from string,
	recipients []string,
	signed []byte,
) (sendResult, error) {
	result := sendResult{tls: deliveryutils.TLSNone, recipients: map[string]delivery.RecipientResult{}}

	client, tlsState, err := r.connect(ctx, destination, false)
	if err != nil {
		return result, err
	}

	defer client.Close()

	result.tls = tlsState

	if err := client.Mail(from); err != nil {
		return result, err
	}

	accepted := make([]string, 0, len(recipients))

	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			code, permanent := Classify(err)
			result.recipients[recipient] = delivery.RecipientResult{
				Email:     recipient,
				Domain:    delivery.DomainOf(recipient),
				Status:    deliveryutils.StatusFailed,
				SMTPCode:  code,
				Error:     err.Error(),
				Permanent: permanent,
			}

			continue
		}

		accepted = append(accepted, recipient)
	}

	if len(accepted) == 0 {
		client.Quit()
		return result, nil
	}

	writer, err := client.Data()
	if err != nil {
		return result, err
	}

	if _, err := writer.Write(signed); err != nil {
		return result, err
	}

	if err := writer.Close(); err != nil {
		return result, err
	}

	result.code = 250

	for _, recipient := range accepted {
		result.recipients[recipient] = delivery.RecipientResult{
			Email:    recipient,
			Domain:   delivery.DomainOf(recipient),
			Status:   deliveryutils.StatusSuccess,
			SMTPCode: 250,
		}
	}

	client.Quit()

	return result, nil
}

func (r *Relay) connect(ctx context.Context, destination Destination, skipVerify bool) (*smtp.Client, string, error) {
	dialer := net.Dialer{Timeout: r.cfg.DialTimeout}

	conn, err := dialer.DialContext(ctx, "tcp", destination.Addr)
	if err != nil {
		return nil, deliveryutils.TLSNone, err
	}

	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(conn, destination.Host)
	if err != nil {
		conn.Close()
		return nil, deliveryutils.TLSNone, err
	}

	if err := client.Hello(r.cfg.HELOHost); err != nil {
		client.Close()
		return nil, deliveryutils.TLSNone, err
	}

	if ok, _ := client.Extension("STARTTLS"); !ok {
		if r.cfg.TLSRequired {
			client.Close()
			return nil, deliveryutils.TLSNone, &permanentError{message: "starttls required but not advertised"}
		}

		return client, deliveryutils.TLSNone, nil
	}

	tlsConfig := &tls.Config{ServerName: destination.Host, InsecureSkipVerify: skipVerify}

	if err := client.StartTLS(tlsConfig); err != nil {
		client.Close()

		if !skipVerify && certificateError(err) {
			return r.connect(ctx, destination, true)
		}

		return nil, deliveryutils.TLSNone, err
	}

	if skipVerify {
		return client, deliveryutils.TLSUnverified, nil
	}

	return client, deliveryutils.TLSVerified, nil
}

type permanentError struct {
	message string
}

func (e *permanentError) Error() string { return e.message }

func certificateError(err error) bool {
	var verification *tls.CertificateVerificationError
	if errors.As(err, &verification) {
		return true
	}

	var hostname x509.HostnameError
	if errors.As(err, &hostname) {
		return true
	}

	var authority x509.UnknownAuthorityError
	if errors.As(err, &authority) {
		return true
	}

	var invalid x509.CertificateInvalidError

	return errors.As(err, &invalid)
}

func Classify(err error) (int, bool) {
	if err == nil {
		return 0, false
	}

	var permanent *permanentError
	if errors.As(err, &permanent) {
		return 0, true
	}

	var protoErr *textproto.Error
	if errors.As(err, &protoErr) {
		return protoErr.Code, protoErr.Code >= 500
	}

	code := parseCode(err.Error())
	if code >= 500 {
		return code, true
	}

	return code, false
}

func parseCode(message string) int {
	fields := strings.Fields(message)
	if len(fields) == 0 {
		return 0
	}

	code, err := strconv.Atoi(fields[0])
	if err != nil || code < 200 || code > 599 {
		return 0
	}

	return code
}

func pendingRecipients(results map[string]delivery.RecipientResult) []string {
	pending := make([]string, 0, len(results))

	for recipient, result := range results {
		if result.Status == deliveryutils.StatusSuccess || result.Permanent {
			continue
		}

		pending = append(pending, recipient)
	}

	return pending
}

func markDomain(
	target map[string]delivery.RecipientResult,
	recipients []string,
	domain string,
	code int,
	message string,
	permanent bool,
) {
	for _, recipient := range recipients {
		target[recipient] = delivery.RecipientResult{
			Email:     recipient,
			Domain:    domain,
			Status:    deliveryutils.StatusFailed,
			SMTPCode:  code,
			Error:     message,
			Permanent: permanent,
		}
	}
}

func markInterrupted(results map[string]delivery.RecipientResult) {
	for recipient, result := range results {
		if result.Status == deliveryutils.StatusSuccess {
			continue
		}

		result.Status = deliveryutils.StatusFailed
		if result.Error == "" {
			result.Error = "interrupted by shutdown"
		}

		results[recipient] = result
	}
}

func finalize(results map[string]delivery.RecipientResult) []delivery.RecipientResult {
	final := make([]delivery.RecipientResult, 0, len(results))

	for _, result := range results {
		if result.Status == deliveryutils.StatusProcessing {
			result.Status = deliveryutils.StatusFailed
			if result.Error == "" {
				result.Error = "delivery did not complete"
			}
		}

		final = append(final, result)
	}

	return final
}

func NextBackoff(current time.Duration, multiplier int, max time.Duration) time.Duration {
	if multiplier < 1 {
		multiplier = 1
	}

	next := current * time.Duration(multiplier)
	if next > max {
		return max
	}

	return next
}

func sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
