package provider_test

import (
	"context"
	"encoding/json"
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

type fakeGraph struct {
	mu        sync.Mutex
	responses map[string]string
	statuses  map[string][]int
	requests  []string
}

func newFakeGraph() *fakeGraph {
	return &fakeGraph{responses: map[string]string{}, statuses: map[string][]int{}}
}

func (f *fakeGraph) client() *provider.Client {
	return provider.NewClientWith(doerFunc(f.do), time.Second)
}

func (f *fakeGraph) do(request *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if request.URL.Host == "login.microsoftonline.com" {
		return respond(http.StatusOK, `{"access_token":"graph-token","expires_in":3600}`, nil), nil
	}

	key := request.URL.Path
	if token := request.URL.Query().Get("$skiptoken"); token != "" {
		key += "#" + token
	}

	f.requests = append(f.requests, key)

	if queued := f.statuses[key]; len(queued) > 0 {
		f.statuses[key] = queued[1:]

		return respond(queued[0], `{"error":{"code":"throttled"}}`, http.Header{"Retry-After": {"1"}}), nil
	}

	if body, ok := f.responses[key]; ok {
		return respond(http.StatusOK, body, nil), nil
	}

	return respond(http.StatusNotFound, `{"error":{"code":"itemNotFound"}}`, nil), nil
}

func (f *fakeGraph) set(path string, value any) {
	encoded, _ := json.Marshal(value)
	f.responses[path] = string(encoded)
}

type object = map[string]any

func graphFile(id, name string, size int) object {
	return object{"id": id, "name": name, "size": size, "file": object{"mimeType": "text/plain"}, "lastModifiedDateTime": "2026-09-01T10:00:00Z"}
}

func graphFolder(id, name string) object {
	return object{"id": id, "name": name, "folder": object{"childCount": 1}}
}

func seedDocuments(f *fakeGraph) {
	f.set("/v1.0/drives/drive-docs/items/folder-q1/children", object{
		"value": []object{
			graphFile("file-a", "a.docx", 10),
			graphFolder("sub-1", "Sub"),
			{"id": "remote-1", "name": "shared.docx", "remoteItem": object{}},
			{"id": "pkg-1", "name": "Notebook", "package": object{"type": "oneNote"}},
		},
		"@odata.nextLink": "https://graph.microsoft.com/v1.0/drives/drive-docs/items/folder-q1/children?$skiptoken=page-2",
	})
	f.set("/v1.0/drives/drive-docs/items/folder-q1/children#page-2", object{"value": []object{graphFile("file-z", "z.txt", 5)}})
	f.set("/v1.0/drives/drive-docs/items/sub-1/children", object{"value": []object{graphFile("file-b", "b.pdf", 7)}})
	f.set("/v1.0/drives/drive-docs/root:/2025/Q1", object{"id": "folder-q1", "folder": object{}})
}

func seedSite(f *fakeGraph) {
	f.set("/v1.0/sites/root", object{"siteCollection": object{"hostname": "contoso.sharepoint.com"}})
	f.set("/v1.0/sites/contoso.sharepoint.com:/sites/finance", object{"id": "site-1"})
	f.set("/v1.0/sites/site-1/drives", object{"value": []object{
		{"id": "drive-docs", "name": "Documents", "webUrl": "https://contoso.sharepoint.com/sites/finance/Shared%20Documents"},
		{"id": "drive-legal", "name": "Legal", "webUrl": "https://contoso.sharepoint.com/sites/finance/Legal"},
	}})
	f.set("/v1.0/sites/site-1/drive", object{"id": "drive-docs"})
}

func listAll(t *testing.T, source interface {
	List(context.Context, string, func(provider.File) error) error
}, target string) []provider.File {
	t.Helper()

	var files []provider.File
	if err := source.List(context.Background(), target, func(file provider.File) error {
		files = append(files, file)
		return nil
	}); err != nil {
		t.Fatalf("list %q: %v", target, err)
	}

	return files
}

func names(files []provider.File) string {
	values := make([]string, 0, len(files))
	for _, file := range files {
		values = append(values, file.Name)
	}

	return strings.Join(values, ",")
}

func TestGraphSharePointTargetsResolveLibraryAndFolder(t *testing.T) {
	for _, target := range []string{
		"/sites/finance/Documents//2025/Q1",
		"/sites/finance//2025/Q1",
		"/sites/finance/2025/Q1",
		"https://contoso.sharepoint.com/sites/finance/Shared%20Documents//2025/Q1",
	} {
		fake := newFakeGraph()
		seedSite(fake)
		seedDocuments(fake)

		source, err := provider.NewGraphSource(context.Background(), fake.client(), "tenant", "client", "secret", provider.GraphSharePoint)
		if err != nil {
			t.Fatalf("connect: %v", err)
		}

		files := listAll(t, source, target)
		if got := names(files); got != "a.docx,Sub/b.pdf,z.txt" {
			t.Fatalf("target %q files = %s, requests = %v", target, got, fake.requests)
		}

		if files[0].Key != "drive-docs/file-a" || files[0].Size != 10 || files[0].ModifiedAt.IsZero() {
			t.Fatalf("target %q first file = %+v", target, files[0])
		}
	}
}

func TestGraphOneDriveTarget(t *testing.T) {
	fake := newFakeGraph()
	fake.set("/v1.0/users/user@contoso.com/drive", object{"id": "drive-user"})
	fake.set("/v1.0/drives/drive-user/root:/Documents/Reports", object{"id": "reports", "folder": object{}})
	fake.set("/v1.0/drives/drive-user/items/reports/children", object{"value": []object{graphFile("f1", "salary.xlsx", 3)}})
	fake.set("/v1.0/drives/drive-user/root", object{"id": "root-id", "folder": object{}})
	fake.set("/v1.0/drives/drive-user/items/root-id/children", object{"value": []object{graphFile("f2", "top.txt", 1)}})

	source, err := provider.NewGraphSource(context.Background(), fake.client(), "tenant", "client", "secret", provider.GraphOneDrive)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	if got := names(listAll(t, source, "user@contoso.com//Documents/Reports")); got != "salary.xlsx" {
		t.Fatalf("folder files = %s", got)
	}

	if got := names(listAll(t, source, "user@contoso.com")); got != "top.txt" {
		t.Fatalf("root files = %s", got)
	}
}

func TestGraphRejectsFileAsFolderTarget(t *testing.T) {
	fake := newFakeGraph()
	fake.set("/v1.0/users/user@contoso.com/drive", object{"id": "drive-user"})
	fake.set("/v1.0/drives/drive-user/root:/notes.txt", object{"id": "file-1", "file": object{}})

	source, _ := provider.NewGraphSource(context.Background(), fake.client(), "tenant", "client", "secret", provider.GraphOneDrive)

	err := source.List(context.Background(), "user@contoso.com/notes.txt", func(provider.File) error { return nil })

	var failure *provider.Error
	if !errors.As(err, &failure) || failure.Reason != provider.ReasonNotFound {
		t.Fatalf("err = %v", err)
	}
}

func TestGraphRejectsUntrustedNextLink(t *testing.T) {
	fake := newFakeGraph()
	fake.set("/v1.0/users/u/drive", object{"id": "d"})
	fake.set("/v1.0/drives/d/root", object{"id": "r", "folder": object{}})
	fake.set("/v1.0/drives/d/items/r/children", object{"value": []object{}, "@odata.nextLink": "https://evil.example.com/steal"})

	source, _ := provider.NewGraphSource(context.Background(), fake.client(), "tenant", "client", "secret", provider.GraphOneDrive)

	if err := source.List(context.Background(), "u", func(provider.File) error { return nil }); err == nil {
		t.Fatal("untrusted next link was followed")
	}
}

func TestGraphRetriesThrottledListing(t *testing.T) {
	fake := newFakeGraph()
	fake.set("/v1.0/users/u/drive", object{"id": "d"})
	fake.set("/v1.0/drives/d/root", object{"id": "r", "folder": object{}})
	fake.set("/v1.0/drives/d/items/r/children", object{"value": []object{graphFile("f", "a.txt", 1)}})
	fake.statuses["/v1.0/drives/d/items/r/children"] = []int{http.StatusTooManyRequests}

	source, _ := provider.NewGraphSource(context.Background(), fake.client(), "tenant", "client", "secret", provider.GraphOneDrive)

	if got := names(listAll(t, source, "u")); got != "a.txt" {
		t.Fatalf("files = %s", got)
	}
}

func TestGraphOpenStreamsContent(t *testing.T) {
	fake := newFakeGraph()
	fake.responses["/v1.0/drives/drive-docs/items/file-a/content"] = "PAN ABCDE1234F"

	source, _ := provider.NewGraphSource(context.Background(), fake.client(), "tenant", "client", "secret", provider.GraphSharePoint)

	body, err := source.Open(context.Background(), provider.File{Key: "drive-docs/file-a"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer body.Close()

	content, _ := io.ReadAll(body)
	if string(content) != "PAN ABCDE1234F" {
		t.Fatalf("content = %q", content)
	}
}

func TestGraphWalkStopsAtMaximumDepth(t *testing.T) {
	fake := newFakeGraph()
	fake.set("/v1.0/users/u/drive", object{"id": "d"})
	fake.set("/v1.0/drives/d/root", object{"id": "level-0", "folder": object{}})

	for level := range 70 {
		id := "level-" + strconv.Itoa(level)
		fake.set("/v1.0/drives/d/items/"+id+"/children", object{"value": []object{
			graphFile("file-"+strconv.Itoa(level), "f.txt", 1),
			graphFolder("level-"+strconv.Itoa(level+1), "n"),
		}})
	}

	source, _ := provider.NewGraphSource(context.Background(), fake.client(), "tenant", "client", "secret", provider.GraphOneDrive)

	if files := listAll(t, source, "u"); len(files) != provider.MaxFolderDepth {
		t.Fatalf("files = %d, want %d", len(files), provider.MaxFolderDepth)
	}
}
