package provider

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	retryAttempts      = 3
	retryBaseDelay     = time.Second
	retryMaxDelay      = time.Minute
	tokenRefreshMargin = 5 * time.Minute
	jsonBodyMaxSize    = 8 << 20
)

type tokenSource func(ctx context.Context) (Token, error)

type authorized struct {
	client   *Client
	provider string
	headers  map[string]string
	fetch    tokenSource

	mu    sync.Mutex
	token Token
}

func newAuthorized(client *Client, provider string, headers map[string]string, fetch tokenSource) *authorized {
	return &authorized{client: client, provider: provider, headers: headers, fetch: fetch}
}

func (a *authorized) bearer(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.token.Value != "" && time.Until(a.token.ExpiresAt) > tokenRefreshMargin {
		return a.token.Value, nil
	}

	token, err := a.fetch(ctx)
	if err != nil {
		return "", err
	}

	a.token = token

	return token.Value, nil
}

func (a *authorized) expire() {
	a.mu.Lock()
	a.token = Token{}
	a.mu.Unlock()
}

func (a *authorized) get(ctx context.Context, stage, endpoint string) (*http.Response, error) {
	var lastErr error

	for attempt := range retryAttempts {
		token, err := a.bearer(ctx)
		if err != nil {
			return nil, err
		}

		response, err := a.client.stream(ctx, a.provider, stage, endpoint, token, a.headers)
		if err == nil {
			return response, nil
		}

		lastErr = err

		var failure *Error
		if !errors.As(err, &failure) || attempt == retryAttempts-1 {
			return nil, err
		}

		if failure.HTTPStatus == http.StatusUnauthorized && attempt == 0 {
			a.expire()

			continue
		}

		if !failure.Transient() {
			return nil, err
		}

		if err := wait(ctx, retryDelay(failure.RetryAfter(), attempt)); err != nil {
			return nil, err
		}
	}

	return nil, lastErr
}

func (a *authorized) getJSON(ctx context.Context, stage, endpoint string, into any) error {
	response, err := a.get(ctx, stage, endpoint)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if err := json.NewDecoder(io.LimitReader(response.Body, jsonBodyMaxSize)).Decode(into); err != nil {
		return &Error{Provider: a.provider, Stage: stage, Reason: ReasonUnknown, code: "malformed_response"}
	}

	return nil
}

func notFound(provider, stage, code string) *Error {
	return &Error{
		Provider:   provider,
		Stage:      stage,
		Reason:     ReasonNotFound,
		HTTPStatus: http.StatusNotFound,
		code:       code,
	}
}

func retryDelay(hint time.Duration, attempt int) time.Duration {
	delay := retryBaseDelay << attempt
	if hint > 0 {
		delay = hint
	}

	return min(delay, retryMaxDelay)
}

func wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
