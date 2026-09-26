package provider_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"dpdp-backend/internal/datadiscovery/provider"
)

const stsResponse = `<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
<GetCallerIdentityResult><Arn>arn:aws:iam::123456789012:user/scanner</Arn><UserId>AIDA</UserId><Account>123456789012</Account></GetCallerIdentityResult>
<ResponseMetadata><RequestId>req</RequestId></ResponseMetadata></GetCallerIdentityResponse>`

type fakeS3 struct {
	mu         sync.Mutex
	headStatus int
	headRegion string
	pages      map[string]string
	objects    map[string]string
	buckets    map[string]string
	hosts      []string
}

func newFakeS3() *fakeS3 {
	return &fakeS3{headStatus: http.StatusOK, pages: map[string]string{}, objects: map[string]string{}, buckets: map[string]string{}}
}

func (f *fakeS3) client() *provider.Client {
	return provider.NewClientWith(doerFunc(f.do), time.Second)
}

func (f *fakeS3) do(request *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if strings.HasPrefix(request.URL.Host, "sts.") {
		return xmlResponse(http.StatusOK, stsResponse, nil), nil
	}

	f.hosts = append(f.hosts, request.Method+" "+request.URL.Host)

	switch {
	case request.Method == http.MethodHead:
		header := http.Header{}
		if f.headRegion != "" {
			header.Set("X-Amz-Bucket-Region", f.headRegion)
		}

		return xmlResponse(f.headStatus, "", header), nil
	case request.URL.Query().Has("max-buckets"):
		key := request.URL.Query().Get("prefix") + "|" + request.URL.Query().Get("continuation-token")
		if body, ok := f.buckets[key]; ok {
			return xmlResponse(http.StatusOK, body, nil), nil
		}

		return xmlResponse(http.StatusForbidden, `<Error><Code>AccessDenied</Code><Message>denied</Message></Error>`, nil), nil
	case request.URL.Query().Get("list-type") == "2":
		key := request.URL.Query().Get("prefix") + "|" + request.URL.Query().Get("continuation-token")
		if page, ok := f.pages[key]; ok {
			return xmlResponse(http.StatusOK, page, nil), nil
		}

		return xmlResponse(http.StatusNotFound, `<Error><Code>NoSuchBucket</Code><Message>missing</Message></Error>`, nil), nil
	case request.Method == http.MethodGet:
		if body, ok := f.objects[request.URL.Path]; ok {
			return xmlResponse(http.StatusOK, body, http.Header{"Content-Type": {"text/csv"}}), nil
		}

		return xmlResponse(http.StatusNotFound, `<Error><Code>NoSuchKey</Code><Message>missing</Message></Error>`, nil), nil
	}

	return xmlResponse(http.StatusBadRequest, "", nil), nil
}

func xmlResponse(status int, body string, header http.Header) *http.Response {
	response := respond(status, body, header)
	response.ContentLength = int64(len(body))
	response.Header.Set("Content-Length", strconv.Itoa(len(body)))

	return response
}

func listPage(truncated bool, next string, keys ...string) string {
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Name>reports</Name>`)

	if truncated {
		builder.WriteString(`<IsTruncated>true</IsTruncated><NextContinuationToken>` + next + `</NextContinuationToken>`)
	} else {
		builder.WriteString(`<IsTruncated>false</IsTruncated>`)
	}

	for _, key := range keys {
		size := "12"
		if strings.HasSuffix(key, "/") {
			size = "0"
		}

		builder.WriteString(`<Contents><Key>` + key + `</Key><LastModified>2026-09-01T10:00:00.000Z</LastModified><Size>` + size + `</Size></Contents>`)
	}

	builder.WriteString(`</ListBucketResult>`)

	return builder.String()
}

func connectS3(t *testing.T, fake *fakeS3) *provider.S3Source {
	t.Helper()

	source, err := provider.NewS3Source(context.Background(), fake.client(), "AKIAABCDEFGHIJKLMNOP", "secret", "ap-south-1")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	return source
}

func TestS3ListsAllPagesInBucketRegion(t *testing.T) {
	fake := newFakeS3()
	fake.headRegion = "us-east-1"
	fake.pages["raw/|"] = listPage(true, "token-2", "raw/a.csv", "raw/dir/")
	fake.pages["raw/|token-2"] = listPage(false, "", "raw/dir/b.json")

	source := connectS3(t, fake)
	files := listAll(t, source, "reports/raw/")

	if got := names(files); got != "raw/a.csv,raw/dir/b.json" {
		t.Fatalf("files = %s", got)
	}

	if files[0].Key != "reports/raw/a.csv" || files[0].Size != 12 || files[0].ModifiedAt.IsZero() {
		t.Fatalf("first = %+v", files[0])
	}

	for _, host := range fake.hosts[1:] {
		if !strings.Contains(host, "us-east-1") {
			t.Fatalf("request went to %s instead of the bucket region", host)
		}
	}
}

func TestS3FollowsRegionRedirect(t *testing.T) {
	fake := newFakeS3()
	fake.headStatus = http.StatusMovedPermanently
	fake.headRegion = "eu-west-1"
	fake.pages["|"] = listPage(false, "", "a.txt")

	source := connectS3(t, fake)

	if got := names(listAll(t, source, "reports")); got != "a.txt" {
		t.Fatalf("files = %s", got)
	}

	if last := fake.hosts[len(fake.hosts)-1]; !strings.Contains(last, "eu-west-1") {
		t.Fatalf("listing went to %s", last)
	}
}

func TestS3MissingBucketIsNotFound(t *testing.T) {
	fake := newFakeS3()
	fake.headStatus = http.StatusNotFound

	source := connectS3(t, fake)
	err := source.List(context.Background(), "missing-bucket", func(provider.File) error { return nil })

	var failure *provider.Error
	if !errors.As(err, &failure) || failure.Reason != provider.ReasonNotFound {
		t.Fatalf("err = %v", err)
	}
}

func TestS3OpenStreamsObject(t *testing.T) {
	fake := newFakeS3()
	fake.objects["/raw/a.csv"] = "id,pan\n1,ABCDE1234F\n"

	source := connectS3(t, fake)

	body, err := source.Open(context.Background(), provider.File{Key: "reports/raw/a.csv"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer body.Close()

	content, _ := io.ReadAll(body)
	if string(content) != "id,pan\n1,ABCDE1234F\n" {
		t.Fatalf("content = %q", content)
	}

	_, err = source.Open(context.Background(), provider.File{Key: "reports/raw/missing.csv"})

	var failure *provider.Error
	if !errors.As(err, &failure) || failure.Reason != provider.ReasonNotFound {
		t.Fatalf("missing object err = %v", err)
	}
}
