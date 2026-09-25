package provider

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	blobListPageSize    = 1000
	blobListBodyMaxSize = 8 << 20
	blobRetryAttempts   = 3
	blobRetryBaseDelay  = time.Second
	blobRetryMaxDelay   = time.Minute
	tokenRefreshMargin  = 5 * time.Minute
)

var ErrInvalidBlobTarget = errors.New("blob target must name a container")

type File struct {
	Key        string
	Name       string
	MIMEType   string
	Size       int64
	ModifiedAt time.Time
}

type BlobSource struct {
	client   *Client
	account  string
	tenantID string
	clientID string
	secret   string

	mu    sync.Mutex
	token Token
}

func NewBlobSource(
	ctx context.Context,
	client *Client,
	account, tenantID, clientID, secret string,
) (*BlobSource, error) {
	source := &BlobSource{
		client:   client,
		account:  account,
		tenantID: tenantID,
		clientID: clientID,
		secret:   secret,
	}

	if _, err := source.bearer(ctx); err != nil {
		return nil, err
	}

	return source, nil
}

type blobListing struct {
	Blobs struct {
		Items []blobItem `xml:"Blob"`
	} `xml:"Blobs"`
	NextMarker string `xml:"NextMarker"`
}

type blobItem struct {
	Name       string `xml:"Name"`
	Properties struct {
		ContentLength int64  `xml:"Content-Length"`
		ContentType   string `xml:"Content-Type"`
		LastModified  string `xml:"Last-Modified"`
		ResourceType  string `xml:"ResourceType"`
	} `xml:"Properties"`
}

func (s *BlobSource) List(ctx context.Context, target string, emit func(File) error) error {
	container, prefix := splitBlobPath(target)
	if container == "" {
		return ErrInvalidBlobTarget
	}

	marker := ""

	for {
		page, err := s.listPage(ctx, container, prefix, marker)
		if err != nil {
			return err
		}

		for _, item := range page.Blobs.Items {
			if isBlobDirectory(item) {
				continue
			}

			file := File{
				Key:      container + "/" + item.Name,
				Name:     item.Name,
				MIMEType: item.Properties.ContentType,
				Size:     item.Properties.ContentLength,
			}

			if modified, err := http.ParseTime(item.Properties.LastModified); err == nil {
				file.ModifiedAt = modified
			}

			if err := emit(file); err != nil {
				return err
			}
		}

		if page.NextMarker == "" {
			return nil
		}

		marker = page.NextMarker
	}
}

func (s *BlobSource) Open(ctx context.Context, file File) (io.ReadCloser, error) {
	container, name := splitBlobPath(file.Key)
	if container == "" || name == "" {
		return nil, ErrInvalidBlobTarget
	}

	endpoint := s.baseURL() + "/" + url.PathEscape(container) + "/" + escapeBlobName(name)

	response, err := s.get(ctx, StageDownload, endpoint)
	if err != nil {
		return nil, err
	}

	return response.Body, nil
}

func (s *BlobSource) listPage(ctx context.Context, container, prefix, marker string) (blobListing, error) {
	query := url.Values{}
	query.Set("restype", "container")
	query.Set("comp", "list")
	query.Set("maxresults", fmt.Sprint(blobListPageSize))

	if prefix != "" {
		query.Set("prefix", prefix)
	}

	if marker != "" {
		query.Set("marker", marker)
	}

	endpoint := s.baseURL() + "/" + url.PathEscape(container) + "?" + query.Encode()

	response, err := s.get(ctx, StageList, endpoint)
	if err != nil {
		return blobListing{}, err
	}
	defer response.Body.Close()

	var page blobListing
	if err := xml.NewDecoder(io.LimitReader(response.Body, blobListBodyMaxSize)).Decode(&page); err != nil {
		return blobListing{}, &Error{
			Provider: ProviderAzureStorage,
			Stage:    StageList,
			Reason:   ReasonUnknown,
			code:     "malformed_listing",
		}
	}

	return page, nil
}

func (s *BlobSource) get(ctx context.Context, stage, endpoint string) (*http.Response, error) {
	var lastErr error

	for attempt := range blobRetryAttempts {
		token, err := s.bearer(ctx)
		if err != nil {
			return nil, err
		}

		response, err := s.client.stream(ctx, ProviderAzureStorage, stage, endpoint, token, map[string]string{
			"x-ms-version": AzureStorageAPIVersion,
		})
		if err == nil {
			return response, nil
		}

		lastErr = err

		var failure *Error
		if !errors.As(err, &failure) || attempt == blobRetryAttempts-1 {
			return nil, err
		}

		if failure.HTTPStatus == http.StatusUnauthorized && attempt == 0 {
			s.expire()

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

func (s *BlobSource) bearer(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token.Value != "" && time.Until(s.token.ExpiresAt) > tokenRefreshMargin {
		return s.token.Value, nil
	}

	token, err := s.client.AzureStorageAccessToken(ctx, s.tenantID, s.clientID, s.secret)
	if err != nil {
		return "", err
	}

	s.token = token

	return token.Value, nil
}

func (s *BlobSource) expire() {
	s.mu.Lock()
	s.token = Token{}
	s.mu.Unlock()
}

func (s *BlobSource) baseURL() string {
	return fmt.Sprintf(AzureBlobURLFormat, s.account)
}

func splitBlobPath(value string) (string, string) {
	container, rest, _ := strings.Cut(strings.TrimPrefix(value, "/"), "/")

	return container, rest
}

func escapeBlobName(name string) string {
	segments := strings.Split(name, "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}

	return strings.Join(segments, "/")
}

func isBlobDirectory(item blobItem) bool {
	if strings.EqualFold(item.Properties.ResourceType, "directory") {
		return true
	}

	return item.Properties.ContentLength == 0 && strings.HasSuffix(item.Name, "/")
}

func retryDelay(hint time.Duration, attempt int) time.Duration {
	delay := blobRetryBaseDelay << attempt
	if hint > 0 {
		delay = hint
	}

	return min(delay, blobRetryMaxDelay)
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
