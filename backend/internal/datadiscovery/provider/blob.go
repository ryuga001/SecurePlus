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
	"time"
)

const (
	blobListPageSize    = 1000
	blobListBodyMaxSize = 8 << 20
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
	auth    *authorized
	account string
}

func NewBlobSource(
	ctx context.Context,
	client *Client,
	account, tenantID, clientID, secret string,
) (*BlobSource, error) {
	auth := newAuthorized(client, ProviderAzureStorage, map[string]string{
		"x-ms-version": AzureStorageAPIVersion,
	}, func(ctx context.Context) (Token, error) {
		return client.AzureStorageAccessToken(ctx, tenantID, clientID, secret)
	})

	if _, err := auth.bearer(ctx); err != nil {
		return nil, err
	}

	return &BlobSource{auth: auth, account: account}, nil
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

	response, err := s.auth.get(ctx, StageDownload, endpoint)
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

	response, err := s.auth.get(ctx, StageList, endpoint)
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
