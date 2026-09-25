package provider_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"dpdp-backend/internal/datadiscovery/provider"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(request *http.Request) (*http.Response, error) { return f(request) }

type fakeAzure struct {
	mu         sync.Mutex
	expiresIn  int
	tokens     int
	requests   []string
	statuses   map[string][]int
	pages      map[string]string
	blobs      map[string]string
	authHeader []string
}

func newFakeAzure() *fakeAzure {
	return &fakeAzure{
		expiresIn: 3600,
		statuses:  map[string][]int{},
		pages:     map[string]string{},
		blobs:     map[string]string{},
	}
}

func (f *fakeAzure) client() *provider.Client {
	return provider.NewClientWith(doerFunc(f.do), time.Second)
}

func (f *fakeAzure) do(request *http.Request) (*http.Response, error) {
	if err := request.Context().Err(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if request.URL.Host == "login.microsoftonline.com" {
		f.tokens++

		return respond(http.StatusOK, fmt.Sprintf(`{"access_token":"token-%d","expires_in":%d}`, f.tokens, f.expiresIn), nil), nil
	}

	key := request.URL.EscapedPath()
	if request.URL.Query().Get("comp") == "list" {
		key += "?marker=" + request.URL.Query().Get("marker") + "&prefix=" + request.URL.Query().Get("prefix")
	}

	f.requests = append(f.requests, key)
	f.authHeader = append(f.authHeader, request.Header.Get("Authorization"))

	if queued := f.statuses[key]; len(queued) > 0 {
		status := queued[0]
		f.statuses[key] = queued[1:]

		return respond(status, "", http.Header{"X-Ms-Error-Code": {"Busy"}}), nil
	}

	if page, ok := f.pages[key]; ok {
		return respond(http.StatusOK, page, nil), nil
	}

	if blob, ok := f.blobs[key]; ok {
		return respond(http.StatusOK, blob, nil), nil
	}

	return respond(http.StatusNotFound, "", http.Header{"X-Ms-Error-Code": {"ContainerNotFound"}}), nil
}

func respond(status int, body string, header http.Header) *http.Response {
	if header == nil {
		header = http.Header{}
	}

	return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}
}

func listing(next string, blobs ...string) string {
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="utf-8"?><EnumerationResults><Blobs>`)
	builder.WriteString(strings.Join(blobs, ""))
	builder.WriteString(`</Blobs><NextMarker>` + next + `</NextMarker></EnumerationResults>`)

	return builder.String()
}

func blob(name string, size int, contentType, resourceType string) string {
	return fmt.Sprintf(`<Blob><Name>%s</Name><Properties><Last-Modified>Tue, 01 Sep 2026 10:00:00 GMT</Last-Modified>`+
		`<Content-Length>%d</Content-Length><Content-Type>%s</Content-Type><ResourceType>%s</ResourceType></Properties></Blob>`,
		name, size, contentType, resourceType)
}

func connect(t *testing.T, fake *fakeAzure) *provider.BlobSource {
	t.Helper()

	source, err := provider.NewBlobSource(context.Background(), fake.client(), "acct", "tenant", "client", "secret")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	return source
}

func TestBlobSourceListsAllPagesAndSkipsDirectories(t *testing.T) {
	fake := newFakeAzure()
	fake.pages["/finance?marker=&prefix=reports/"] = listing("page-2",
		blob("reports/a.txt", 10, "text/plain", ""),
		blob("reports/sub", 0, "", "directory"),
		blob("reports/folder/", 0, "", ""),
	)
	fake.pages["/finance?marker=page-2&prefix=reports/"] = listing("",
		blob("reports/b &amp; c.csv", 20, "text/csv", "file"),
	)

	source := connect(t, fake)

	var files []provider.File
	err := source.List(context.Background(), "finance/reports/", func(file provider.File) error {
		files = append(files, file)
		return nil
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("files = %+v", files)
	}

	if files[0].Key != "finance/reports/a.txt" || files[0].Name != "reports/a.txt" || files[0].Size != 10 || files[0].MIMEType != "text/plain" {
		t.Fatalf("first = %+v", files[0])
	}

	if files[1].Key != "finance/reports/b & c.csv" || files[1].ModifiedAt.IsZero() {
		t.Fatalf("second = %+v", files[1])
	}
}

func TestBlobSourceStopsWhenEmitFails(t *testing.T) {
	fake := newFakeAzure()
	fake.pages["/finance?marker=&prefix="] = listing("page-2", blob("a.txt", 1, "", ""), blob("b.txt", 1, "", ""))

	source := connect(t, fake)
	stop := errors.New("queue closed")

	calls := 0
	err := source.List(context.Background(), "finance", func(provider.File) error {
		calls++
		return stop
	})

	if !errors.Is(err, stop) || calls != 1 {
		t.Fatalf("err = %v calls = %d", err, calls)
	}
}

func TestBlobSourceOpenStreamsEscapedName(t *testing.T) {
	fake := newFakeAzure()
	fake.blobs["/finance/reports/Q1%20plan%231.txt"] = "PAN ABCDE1234F"

	source := connect(t, fake)

	body, err := source.Open(context.Background(), provider.File{Key: "finance/reports/Q1 plan#1.txt"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer body.Close()

	content, _ := io.ReadAll(body)
	if string(content) != "PAN ABCDE1234F" {
		t.Fatalf("content = %q", content)
	}
}

func TestBlobSourceRetriesRateLimits(t *testing.T) {
	fake := newFakeAzure()
	fake.statuses["/finance/a.txt"] = []int{http.StatusTooManyRequests}
	fake.blobs["/finance/a.txt"] = "ok"

	source := connect(t, fake)

	body, err := source.Open(context.Background(), provider.File{Key: "finance/a.txt"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	body.Close()

	if len(fake.requests) != 2 {
		t.Fatalf("requests = %v", fake.requests)
	}
}

func TestBlobSourceRefreshesTokenAfterUnauthorized(t *testing.T) {
	fake := newFakeAzure()
	fake.statuses["/finance/a.txt"] = []int{http.StatusUnauthorized}
	fake.blobs["/finance/a.txt"] = "ok"

	source := connect(t, fake)

	body, err := source.Open(context.Background(), provider.File{Key: "finance/a.txt"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	body.Close()

	if fake.tokens != 2 || fake.authHeader[1] != "Bearer token-2" {
		t.Fatalf("tokens = %d headers = %v", fake.tokens, fake.authHeader)
	}
}

func TestBlobSourceRefreshesTokenBeforeExpiry(t *testing.T) {
	fake := newFakeAzure()
	fake.expiresIn = 60
	fake.blobs["/finance/a.txt"] = "ok"

	source := connect(t, fake)

	for range 2 {
		body, err := source.Open(context.Background(), provider.File{Key: "finance/a.txt"})
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		body.Close()
	}

	if fake.tokens != 3 {
		t.Fatalf("tokens = %d, short-lived tokens must be refreshed before use", fake.tokens)
	}

	long := newFakeAzure()
	long.blobs["/finance/a.txt"] = "ok"

	cached := connect(t, long)
	for range 2 {
		body, _ := cached.Open(context.Background(), provider.File{Key: "finance/a.txt"})
		body.Close()
	}

	if long.tokens != 1 {
		t.Fatalf("tokens = %d, long-lived tokens must be reused", long.tokens)
	}
}

func TestBlobSourceClassifiesMissingContainer(t *testing.T) {
	source := connect(t, newFakeAzure())

	err := source.List(context.Background(), "missing", func(provider.File) error { return nil })

	var failure *provider.Error
	if !errors.As(err, &failure) || failure.Reason != provider.ReasonNotFound || failure.Code() != "ContainerNotFound" {
		t.Fatalf("err = %v", err)
	}
}

func TestBlobSourceHonoursCancellation(t *testing.T) {
	fake := newFakeAzure()
	fake.pages["/finance?marker=&prefix="] = listing("", blob("a.txt", 1, "", ""))

	source := connect(t, fake)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := source.List(ctx, "finance", func(provider.File) error { return nil }); err == nil {
		t.Fatal("list succeeded after cancellation")
	}
}

func TestBlobSourceRejectsTargetWithoutContainer(t *testing.T) {
	source := connect(t, newFakeAzure())

	if err := source.List(context.Background(), "", func(provider.File) error { return nil }); !errors.Is(err, provider.ErrInvalidBlobTarget) {
		t.Fatalf("err = %v", err)
	}
}
