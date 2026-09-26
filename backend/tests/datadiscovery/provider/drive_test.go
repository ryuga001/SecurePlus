package provider_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"dpdp-backend/internal/datadiscovery/provider"
)

const folderMIME = "application/vnd.google-apps.folder"

type fakeDrive struct {
	mu        sync.Mutex
	files     map[string][]object
	drives    []object
	users     []object
	byID      map[string]string
	downloads map[string]string
	statuses  map[string][]int
	queries   []string
	bearers   map[string]string
}

func newFakeDrive() *fakeDrive {
	return &fakeDrive{
		files:     map[string][]object{},
		byID:      map[string]string{},
		downloads: map[string]string{},
		statuses:  map[string][]int{},
		bearers:   map[string]string{},
	}
}

func impersonatedToken(request *http.Request) string {
	body, _ := io.ReadAll(request.Body)
	form, _ := url.ParseQuery(string(body))
	parts := strings.Split(form.Get("assertion"), ".")

	if len(parts) != 3 {
		return "token-invalid"
	}

	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])

	var claims struct {
		Sub   string `json:"sub"`
		Scope string `json:"scope"`
	}

	_ = json.Unmarshal(payload, &claims)

	subject := claims.Sub
	if subject == "" {
		subject = "service-account"
	}

	scope := "drive"
	if strings.Contains(claims.Scope, "admin.directory") {
		scope = "directory"
	}

	return "token-" + subject + "-" + scope
}

func (f *fakeDrive) client() *provider.Client {
	return provider.NewClientWith(doerFunc(f.do), time.Second)
}

func (f *fakeDrive) do(request *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if request.URL.Host == "oauth2.googleapis.com" {
		return respond(http.StatusOK, `{"access_token":"`+impersonatedToken(request)+`","expires_in":3600}`, nil), nil
	}

	path := request.URL.Path
	query := request.URL.Query()
	f.bearers[path] = strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")

	if request.URL.Host == "admin.googleapis.com" {
		f.queries = append(f.queries, "directory|"+query.Get("query")+"|"+query.Get("customer"))
		encoded, _ := json.Marshal(object{"users": f.users})

		return respond(http.StatusOK, string(encoded), nil), nil
	}

	if queued := f.statuses[path]; len(queued) > 0 {
		f.statuses[path] = queued[1:]

		return respond(queued[0], `{"error":{"code":403,"errors":[{"reason":"rateLimitExceeded"}]}}`, nil), nil
	}

	switch {
	case path == "/drive/v3/drives":
		encoded, _ := json.Marshal(object{"drives": f.drives})

		return respond(http.StatusOK, string(encoded), nil), nil
	case strings.HasPrefix(path, "/drive/v3/drives/"):
		if id, ok := f.byID[strings.TrimPrefix(path, "/drive/v3/drives/")]; ok {
			return respond(http.StatusOK, `{"id":"`+id+`"}`, nil), nil
		}

		return respond(http.StatusNotFound, `{"error":{"code":404,"errors":[{"reason":"notFound"}]}}`, nil), nil
	case path == "/drive/v3/files":
		q := query.Get("q")
		f.queries = append(f.queries, q+"|corpora="+query.Get("corpora")+"|driveId="+query.Get("driveId"))
		encoded, _ := json.Marshal(object{"files": f.files[q]})

		return respond(http.StatusOK, string(encoded), nil), nil
	}

	key := path
	if export := query.Get("mimeType"); export != "" {
		key += "?export=" + export
	}

	if body, ok := f.downloads[key]; ok {
		return respond(http.StatusOK, body, nil), nil
	}

	return respond(http.StatusNotFound, `{"error":{"code":404}}`, nil), nil
}

func privateKeyPEM(t *testing.T) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
}

func connectDrive(t *testing.T, fake *fakeDrive, key string) *provider.DriveSource {
	t.Helper()

	return connectDriveAs(t, fake, key, "")
}

func connectDriveAs(t *testing.T, fake *fakeDrive, key, subject string) *provider.DriveSource {
	t.Helper()

	source, err := provider.NewDriveSource(context.Background(), fake.client(), "sa@project.iam.gserviceaccount.com", subject, "", key)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	return source
}

func children(parent string) string {
	return "'" + parent + "' in parents and trashed = false"
}

func folderLookup(parent, name string) string {
	return "'" + parent + "' in parents and name = '" + name + "' and mimeType = '" + folderMIME + "' and trashed = false"
}

func TestDriveListsSharedDriveFolderRecursively(t *testing.T) {
	fake := newFakeDrive()
	fake.drives = []object{{"id": "sd-1", "name": "Finance"}}
	fake.files[folderLookup("sd-1", "Reports")] = []object{{"id": "reports"}}
	fake.files[children("reports")] = []object{
		{"id": "pdf-1", "name": "q1.pdf", "mimeType": "application/pdf", "size": "1234", "modifiedTime": "2026-09-01T10:00:00Z"},
		{"id": "doc-1", "name": "Budget v1.2", "mimeType": "application/vnd.google-apps.document"},
		{"id": "sub", "name": "Payroll", "mimeType": folderMIME},
		{"id": "short", "name": "Link", "mimeType": "application/vnd.google-apps.shortcut"},
		{"id": "form", "name": "Survey", "mimeType": "application/vnd.google-apps.form"},
	}
	fake.files[children("sub")] = []object{{"id": "txt-1", "name": "notes.txt", "mimeType": "text/plain", "size": "10"}}

	source := connectDrive(t, fake, privateKeyPEM(t))
	files := listAll(t, source, "Finance//Reports")

	if got := names(files); got != "q1.pdf,Budget v1.2.docx,Payroll/notes.txt" {
		t.Fatalf("files = %s", got)
	}

	if files[0].Key != "pdf-1" || files[0].Size != 1234 || files[0].ModifiedAt.IsZero() {
		t.Fatalf("pdf = %+v", files[0])
	}

	for _, query := range fake.queries {
		if !strings.Contains(query, "|corpora=drive|driveId=sd-1") {
			t.Fatalf("shared drive scope missing from %q", query)
		}
	}
}

func TestDriveResolvesMyDriveAndDriveIDs(t *testing.T) {
	fake := newFakeDrive()
	fake.byID["0AbCdEf"] = "0AbCdEf"
	fake.files[children("root")] = []object{{"id": "a", "name": "mine.csv", "mimeType": "text/csv", "size": "3"}}
	fake.files[children("0AbCdEf")] = []object{{"id": "b", "name": "shared.csv", "mimeType": "text/csv", "size": "3"}}

	source := connectDrive(t, fake, privateKeyPEM(t))

	if got := names(listAll(t, source, "My Drive")); got != "mine.csv" {
		t.Fatalf("my drive files = %s", got)
	}

	if got := names(listAll(t, source, "0AbCdEf")); got != "shared.csv" {
		t.Fatalf("drive id files = %s", got)
	}
}

func TestDriveMissingSharedDriveIsNotFound(t *testing.T) {
	source := connectDrive(t, newFakeDrive(), privateKeyPEM(t))

	err := source.List(context.Background(), "Nowhere", func(provider.File) error { return nil })

	var failure *provider.Error
	if !errors.As(err, &failure) || failure.Reason != provider.ReasonNotFound {
		t.Fatalf("err = %v", err)
	}
}

func TestDriveOpenDownloadsAndExports(t *testing.T) {
	fake := newFakeDrive()
	fake.downloads["/drive/v3/files/pdf-1"] = "binary pdf"
	fake.downloads["/drive/v3/files/doc-1/export?export=application/vnd.openxmlformats-officedocument.wordprocessingml.document"] = "docx bytes"

	source := connectDrive(t, fake, privateKeyPEM(t))

	for file, want := range map[provider.File]string{
		{Key: "pdf-1", MIMEType: "application/pdf"}:                      "binary pdf",
		{Key: "doc-1", MIMEType: "application/vnd.google-apps.document"}: "docx bytes",
	} {
		body, err := source.Open(context.Background(), file)
		if err != nil {
			t.Fatalf("open %s: %v", file.Key, err)
		}

		content, _ := io.ReadAll(body)
		body.Close()

		if string(content) != want {
			t.Fatalf("%s content = %q", file.Key, content)
		}
	}
}

func TestDriveRetriesRateLimit(t *testing.T) {
	fake := newFakeDrive()
	fake.statuses["/drive/v3/files"] = []int{http.StatusForbidden}
	fake.files[children("root")] = []object{{"id": "a", "name": "a.txt", "mimeType": "text/plain", "size": "1"}}

	source := connectDrive(t, fake, privateKeyPEM(t))

	if got := names(listAll(t, source, "root")); got != "a.txt" {
		t.Fatalf("files = %s", got)
	}
}

func TestDriveScansUserDriveByImpersonation(t *testing.T) {
	fake := newFakeDrive()
	fake.files[folderLookup("root", "Reports")] = []object{{"id": "jane-reports"}}
	fake.files[children("jane-reports")] = []object{{"id": "f-1", "name": "payslip.pdf", "mimeType": "application/pdf", "size": "9"}}
	fake.downloads["/drive/v3/files/f-1"] = "pdf bytes"

	source := connectDriveAs(t, fake, privateKeyPEM(t), "admin@corp.com")
	files := listAll(t, source, "Jane@Corp.com//Reports")

	if len(files) != 1 || files[0].Key != "jane@corp.com/f-1" || files[0].Name != "payslip.pdf" {
		t.Fatalf("files = %+v", files)
	}

	if got := fake.bearers["/drive/v3/files"]; got != "token-jane@corp.com-drive" {
		t.Fatalf("listing ran as %q, want the impersonated user", got)
	}

	body, err := source.Open(context.Background(), files[0])
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	body.Close()

	if got := fake.bearers["/drive/v3/files/f-1"]; got != "token-jane@corp.com-drive" {
		t.Fatalf("download ran as %q, want the file owner", got)
	}
}
