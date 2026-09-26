package provider_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"dpdp-backend/internal/datadiscovery/provider"
)

func values(page provider.TargetPage) string {
	items := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, item.Value)
	}

	return strings.Join(items, ",")
}

func TestS3BucketsPageWithPrefixAndCursor(t *testing.T) {
	fake := newFakeS3()
	fake.buckets["fin|"] = `<ListAllMyBucketsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Buckets>` +
		`<Bucket><Name>finance-raw</Name><BucketRegion>ap-south-1</BucketRegion></Bucket>` +
		`<Bucket><Name>finance-archive</Name><BucketRegion>eu-west-1</BucketRegion></Bucket>` +
		`</Buckets><ContinuationToken>next-1</ContinuationToken></ListAllMyBucketsResult>`
	fake.buckets["fin|next-1"] = `<ListAllMyBucketsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Buckets>` +
		`<Bucket><Name>finance-zeta</Name></Bucket></Buckets></ListAllMyBucketsResult>`

	source := connectS3(t, fake)

	first, err := source.Buckets(context.Background(), provider.TargetQuery{Search: "FIN", Limit: 2})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}

	if values(first) != "finance-raw,finance-archive" || first.NextCursor != "next-1" || first.Items[1].Description != "eu-west-1" {
		t.Fatalf("first page = %+v", first)
	}

	second, err := source.Buckets(context.Background(), provider.TargetQuery{Search: "fin", Cursor: first.NextCursor})
	if err != nil || values(second) != "finance-zeta" || second.NextCursor != "" {
		t.Fatalf("second page = %+v err = %v", second, err)
	}

	_, err = source.Buckets(context.Background(), provider.TargetQuery{Search: "denied"})

	var failure *provider.Error
	if !errors.As(err, &failure) || failure.Reason != provider.ReasonPermissionDenied {
		t.Fatalf("denied err = %v", err)
	}
}

func TestBlobContainersPage(t *testing.T) {
	fake := newFakeAzure()
	fake.pages["/?marker=&prefix=fin"] = `<EnumerationResults><Containers><Container><Name>finance</Name></Container>` +
		`<Container><Name>finops</Name></Container></Containers><NextMarker>m2</NextMarker></EnumerationResults>`
	fake.pages["/?marker=m2&prefix=fin"] = `<EnumerationResults><Containers><Container><Name>finz</Name></Container></Containers><NextMarker/></EnumerationResults>`

	source := connect(t, fake)

	first, err := source.Containers(context.Background(), provider.TargetQuery{Search: "Fin"})
	if err != nil || values(first) != "finance,finops" || first.NextCursor != "m2" {
		t.Fatalf("first = %+v err = %v", first, err)
	}

	second, err := source.Containers(context.Background(), provider.TargetQuery{Search: "fin", Cursor: "m2"})
	if err != nil || values(second) != "finz" || second.NextCursor != "" {
		t.Fatalf("second = %+v err = %v", second, err)
	}
}

func TestDriveTargetsListSharedDrivesThenWorkspaceUsers(t *testing.T) {
	fake := newFakeDrive()
	fake.drives = []object{
		{"id": "0A1", "name": "Finance"},
		{"id": "0A2", "name": "Legal"},
		{"id": "0A3", "name": "Legal"},
	}
	fake.users = []object{
		{"primaryEmail": "Jane@Corp.com", "name": object{"fullName": "Jane Doe"}},
		{"primaryEmail": "gone@corp.com", "suspended": true},
		{"primaryEmail": "raj@corp.com"},
	}

	source := connectDriveAs(t, fake, privateKeyPEM(t), "admin@corp.com")

	drives, err := source.Targets(context.Background(), provider.TargetQuery{}, true)
	if err != nil || values(drives) != "Finance,0A2,0A3" || drives.Items[0].Kind != provider.TargetKindSharedDrive {
		t.Fatalf("drives = %+v err = %v", drives, err)
	}

	if drives.NextCursor != "users:" {
		t.Fatalf("cursor after shared drives = %q, want the users phase", drives.NextCursor)
	}

	users, err := source.Targets(context.Background(), provider.TargetQuery{Cursor: drives.NextCursor}, true)
	if err != nil || values(users) != "jane@corp.com,raj@corp.com" {
		t.Fatalf("users = %+v err = %v", users, err)
	}

	if users.Items[0].Label != "Jane Doe" || users.Items[1].Label != "raj@corp.com" || users.Items[0].Kind != provider.TargetKindUserDrive {
		t.Fatalf("user labels = %+v", users.Items)
	}

	if got := fake.bearers["/admin/directory/v1/users"]; got != "token-admin@corp.com-directory" {
		t.Fatalf("directory listing ran as %q, want the delegated admin with the directory scope", got)
	}

	if users.NextCursor != "" {
		t.Fatalf("users cursor = %q", users.NextCursor)
	}
}

func TestDriveTargetsSkipUsersWithoutDelegation(t *testing.T) {
	fake := newFakeDrive()
	fake.drives = []object{{"id": "0A1", "name": "Finance"}}

	source := connectDrive(t, fake, privateKeyPEM(t))

	page, err := source.Targets(context.Background(), provider.TargetQuery{}, false)
	if err != nil || values(page) != "Finance" || page.NextCursor != "" {
		t.Fatalf("page = %+v err = %v", page, err)
	}
}

func TestDriveTargetsUserSearch(t *testing.T) {
	fake := newFakeDrive()
	source := connectDriveAs(t, fake, privateKeyPEM(t), "admin@corp.com")

	for search, want := range map[string]string{
		"jane":     "directory|jane|my_customer",
		"jan@corp": "directory|email:jan@corp*|my_customer",
		"o'neil*":  "directory|oneil|my_customer",
	} {
		fake.queries = nil

		if _, err := source.Targets(context.Background(), provider.TargetQuery{Search: search, Cursor: "users:"}, true); err != nil {
			t.Fatalf("search %q: %v", search, err)
		}

		if len(fake.queries) != 1 || fake.queries[0] != want {
			t.Fatalf("search %q sent %v, want %q", search, fake.queries, want)
		}
	}
}

func TestGraphSitesSkipPersonalSitesAndExpand(t *testing.T) {
	fake := newFakeGraph()
	fake.set("/v1.0/sites", object{
		"value": []object{
			{"displayName": "Finance", "webUrl": "https://contoso.sharepoint.com/sites/finance"},
			{"displayName": "Root", "webUrl": "https://contoso.sharepoint.com/"},
			{"displayName": "Jane", "webUrl": "https://contoso-my.sharepoint.com/personal/jane"},
		},
		"@odata.nextLink": "https://graph.microsoft.com/v1.0/sites?search=%2A&$skiptoken=s2",
	})

	source, _ := provider.NewGraphSource(context.Background(), fake.client(), "tenant", "client", "secret", provider.GraphSharePoint)

	page, err := source.Sites(context.Background(), provider.TargetQuery{})
	if err != nil {
		t.Fatalf("sites: %v", err)
	}

	if values(page) != "https://contoso.sharepoint.com/sites/finance,https://contoso.sharepoint.com" {
		t.Fatalf("sites = %s", values(page))
	}

	if !page.Items[0].Expandable || page.Items[0].Description != "/sites/finance" || page.Items[1].Description != "/" {
		t.Fatalf("items = %+v", page.Items)
	}

	if !strings.Contains(fake.queries[len(fake.queries)-1], "search=%2A") || page.NextCursor == "" {
		t.Fatalf("query = %s cursor = %q", fake.queries[len(fake.queries)-1], page.NextCursor)
	}

	if _, err := source.Sites(context.Background(), provider.TargetQuery{Cursor: "https://evil.example.com/next"}); err == nil {
		t.Fatal("untrusted cursor followed")
	}
}

func TestGraphLibrariesForSite(t *testing.T) {
	fake := newFakeGraph()
	seedSite(fake)

	source, _ := provider.NewGraphSource(context.Background(), fake.client(), "tenant", "client", "secret", provider.GraphSharePoint)

	page, err := source.Libraries(context.Background(), provider.TargetQuery{Parent: "https://contoso.sharepoint.com/sites/finance"})
	if err != nil || values(page) != "Documents,Legal" {
		t.Fatalf("libraries = %+v err = %v", page, err)
	}

	filtered, _ := source.Libraries(context.Background(), provider.TargetQuery{Parent: "/sites/finance", Search: "leg"})
	if values(filtered) != "Legal" {
		t.Fatalf("filtered = %s", values(filtered))
	}
}

func TestGraphUsersSearchEscapesQuotes(t *testing.T) {
	fake := newFakeGraph()
	fake.set("/v1.0/users", object{"value": []object{
		{"displayName": "Dan O'Neil", "userPrincipalName": "dan@contoso.com", "mail": "dan@contoso.com"},
		{"displayName": "No UPN"},
	}})

	source, _ := provider.NewGraphSource(context.Background(), fake.client(), "tenant", "client", "secret", provider.GraphOneDrive)

	page, err := source.Users(context.Background(), provider.TargetQuery{Search: "Dan O'"})
	if err != nil || values(page) != "dan@contoso.com" || page.Items[0].Label != "Dan O'Neil" {
		t.Fatalf("users = %+v err = %v", page, err)
	}

	if query := fake.queries[len(fake.queries)-1]; !strings.Contains(query, "startswith%28displayName%2C%27Dan+O%27%27%27%29") {
		t.Fatalf("filter not escaped: %s", query)
	}
}
