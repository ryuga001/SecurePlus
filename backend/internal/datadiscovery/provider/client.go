package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	StageToken = "token"
	StageProbe = "probe"

	ReasonAuthFailed       = "AUTH_FAILED"
	ReasonPermissionDenied = "PERMISSION_DENIED"
	ReasonNotFound         = "NOT_FOUND"
	ReasonRateLimited      = "RATE_LIMITED"
	ReasonUnavailable      = "UNAVAILABLE"
	ReasonUnknown          = "UNKNOWN"

	errorBodyMaxSize = 4096
)

type Doer interface {
	Do(request *http.Request) (*http.Response, error)
}

type Client struct {
	http    Doer
	timeout time.Duration
}

func NewClient(timeout time.Duration) *Client {
	return &Client{http: &http.Client{Timeout: timeout}, timeout: timeout}
}

func NewClientWith(doer Doer, timeout time.Duration) *Client {
	return &Client{http: doer, timeout: timeout}
}

type Error struct {
	Provider   string
	Stage      string
	Reason     string
	HTTPStatus int

	code      string
	requestID string
}

func (e *Error) Error() string {
	return e.Provider + " " + e.Stage + " failed: " + e.Reason
}

func (e *Error) Code() string      { return e.code }
func (e *Error) RequestID() string { return e.requestID }

func newError(provider, stage string, status int, transportErr error, safe safeDetail) *Error {
	return &Error{
		Provider:   provider,
		Stage:      stage,
		Reason:     Classify(status, transportErr),
		HTTPStatus: status,
		code:       safe.code,
		requestID:  safe.requestID,
	}
}

func Classify(status int, transportErr error) string {
	if transportErr != nil {
		return ReasonUnavailable
	}

	switch {
	case status == http.StatusUnauthorized, status == http.StatusBadRequest:
		return ReasonAuthFailed
	case status == http.StatusForbidden:
		return ReasonPermissionDenied
	case status == http.StatusNotFound:
		return ReasonNotFound
	case status == http.StatusTooManyRequests:
		return ReasonRateLimited
	case status >= 500:
		return ReasonUnavailable
	}

	return ReasonUnknown
}

type safeDetail struct {
	code      string
	requestID string
}

type providerErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	ErrorString   string `json:"-"`
	ErrorCodes    []int  `json:"error_codes"`
	TraceID       string `json:"trace_id"`
	CorrelationID string `json:"correlation_id"`
}

func parseSafeDetail(raw []byte) safeDetail {
	if len(raw) == 0 {
		return safeDetail{}
	}

	var flat struct {
		Error         string `json:"error"`
		ErrorCodes    []int  `json:"error_codes"`
		TraceID       string `json:"trace_id"`
		CorrelationID string `json:"correlation_id"`
	}

	if err := json.Unmarshal(raw, &flat); err == nil && flat.Error != "" {
		code := flat.Error
		for _, numeric := range flat.ErrorCodes {
			code += ":" + strconv.Itoa(numeric)
		}

		requestID := flat.CorrelationID
		if requestID == "" {
			requestID = flat.TraceID
		}

		return safeDetail{code: code, requestID: requestID}
	}

	var nested providerErrorBody
	if err := json.Unmarshal(raw, &nested); err == nil && nested.Error.Code != "" {
		return safeDetail{code: nested.Error.Code}
	}

	return safeDetail{}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (c *Client) postForm(ctx context.Context, provider, stage, endpoint, body string) (string, error) {
	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return "", err
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response, err := c.http.Do(request)
	if err != nil {
		return "", newError(provider, stage, 0, err, safeDetail{})
	}
	defer response.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(response.Body, errorBodyMaxSize))

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", newError(provider, stage, response.StatusCode, nil, parseSafeDetail(raw))
	}

	var token tokenResponse
	if err := json.Unmarshal(raw, &token); err != nil || token.AccessToken == "" {
		return "", newError(provider, stage, response.StatusCode, nil, safeDetail{code: "malformed_token_response"})
	}

	return token.AccessToken, nil
}

func (c *Client) get(ctx context.Context, provider, stage, endpoint, bearer string, headers map[string]string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	request.Header.Set("Authorization", "Bearer "+bearer)
	request.Header.Set("Accept", "application/json")

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := c.http.Do(request)
	if err != nil {
		return newError(provider, stage, 0, err, safeDetail{})
	}
	defer response.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(response.Body, errorBodyMaxSize))

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return newError(provider, stage, response.StatusCode, nil, parseSafeDetail(raw))
	}

	return nil
}
