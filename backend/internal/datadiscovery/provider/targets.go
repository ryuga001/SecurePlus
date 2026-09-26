package provider

import (
	"context"
	"encoding/xml"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	DefaultTargetLimit = 50
	MaxTargetLimit     = 100

	TargetKindSharedDrive = "shared_drive"
	TargetKindUserDrive   = "user_drive"

	drivePhaseDrives = "drives"
	drivePhaseUsers  = "users"
)

type TargetQuery struct {
	Search string
	Cursor string
	Parent string
	Limit  int
}

type TargetOption struct {
	Value       string
	Label       string
	Description string
	Kind        string
	Expandable  bool
}

type TargetPage struct {
	Items      []TargetOption
	NextCursor string
}

func (q TargetQuery) limit() int {
	if q.Limit <= 0 {
		return DefaultTargetLimit
	}

	return min(q.Limit, MaxTargetLimit)
}

func (s *S3Source) Buckets(ctx context.Context, query TargetQuery) (TargetPage, error) {
	input := &s3.ListBucketsInput{MaxBuckets: aws.Int32(int32(query.limit()))}

	if search := strings.ToLower(query.Search); search != "" {
		input.Prefix = aws.String(search)
	}

	if query.Cursor != "" {
		input.ContinuationToken = aws.String(query.Cursor)
	}

	output, err := s.api(s.region, "").ListBuckets(ctx, input)
	if err != nil {
		return TargetPage{}, s3Error(StageList, err)
	}

	page := TargetPage{NextCursor: aws.ToString(output.ContinuationToken)}

	for _, bucket := range output.Buckets {
		name := aws.ToString(bucket.Name)
		page.Items = append(page.Items, TargetOption{
			Value:       name,
			Label:       name,
			Description: aws.ToString(bucket.BucketRegion),
		})
	}

	return page, nil
}

type containerListing struct {
	Containers struct {
		Items []struct {
			Name string `xml:"Name"`
		} `xml:"Container"`
	} `xml:"Containers"`
	NextMarker string `xml:"NextMarker"`
}

func (s *BlobSource) Containers(ctx context.Context, query TargetQuery) (TargetPage, error) {
	values := url.Values{}
	values.Set("comp", "list")
	values.Set("maxresults", strconv.Itoa(query.limit()))

	if search := strings.ToLower(query.Search); search != "" {
		values.Set("prefix", search)
	}

	if query.Cursor != "" {
		values.Set("marker", query.Cursor)
	}

	response, err := s.auth.get(ctx, StageList, s.baseURL()+"/?"+values.Encode())
	if err != nil {
		return TargetPage{}, err
	}
	defer response.Body.Close()

	var listing containerListing
	if err := xml.NewDecoder(io.LimitReader(response.Body, blobListBodyMaxSize)).Decode(&listing); err != nil {
		return TargetPage{}, &Error{Provider: ProviderAzureStorage, Stage: StageList, Reason: ReasonUnknown, code: "malformed_listing"}
	}

	page := TargetPage{NextCursor: listing.NextMarker}

	for _, container := range listing.Containers.Items {
		page.Items = append(page.Items, TargetOption{Value: container.Name, Label: container.Name})
	}

	return page, nil
}

func (s *DriveSource) Targets(ctx context.Context, query TargetQuery, delegated bool) (TargetPage, error) {
	phase, token, _ := strings.Cut(query.Cursor, ":")
	if query.Cursor == "" {
		phase = drivePhaseDrives
	}

	switch phase {
	case drivePhaseDrives:
		page, err := s.sharedDrivePage(ctx, query, token)
		if err != nil {
			return TargetPage{}, err
		}

		switch {
		case page.NextCursor != "":
			page.NextCursor = drivePhaseDrives + ":" + page.NextCursor
		case delegated:
			page.NextCursor = drivePhaseUsers + ":"
		}

		return page, nil
	case drivePhaseUsers:
		if !delegated {
			return TargetPage{}, nil
		}

		page, err := s.userDrivePage(ctx, query, token)
		if err != nil {
			return TargetPage{}, err
		}

		if page.NextCursor != "" {
			page.NextCursor = drivePhaseUsers + ":" + page.NextCursor
		}

		return page, nil
	}

	return TargetPage{}, ErrInvalidDriveTarget
}

func (s *DriveSource) sharedDrivePage(ctx context.Context, query TargetQuery, token string) (TargetPage, error) {
	values := url.Values{}
	values.Set("pageSize", strconv.Itoa(query.limit()))
	values.Set("fields", "nextPageToken,drives(id,name)")

	if query.Search != "" {
		values.Set("q", "name contains '"+driveQueryEscaper.Replace(query.Search)+"'")
	}

	if token != "" {
		values.Set("pageToken", token)
	}

	var listing struct {
		NextPageToken string `json:"nextPageToken"`
		Drives        []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"drives"`
	}

	if err := s.auth.getJSON(ctx, StageList, GoogleDriveBaseURL+"/drives?"+values.Encode(), &listing); err != nil {
		return TargetPage{}, err
	}

	page := TargetPage{NextCursor: listing.NextPageToken}

	counts := make(map[string]int, len(listing.Drives))
	for _, drive := range listing.Drives {
		counts[drive.Name]++
	}

	for _, drive := range listing.Drives {
		value := drive.Name
		if counts[drive.Name] > 1 || value == "" || isEmailTarget(value) {
			value = drive.ID
		}

		page.Items = append(page.Items, TargetOption{
			Value:       value,
			Label:       drive.Name,
			Description: drive.ID,
			Kind:        TargetKindSharedDrive,
		})
	}

	return page, nil
}

func (s *DriveSource) userDrivePage(ctx context.Context, query TargetQuery, token string) (TargetPage, error) {
	values := url.Values{}
	values.Set("customer", "my_customer")
	values.Set("maxResults", strconv.Itoa(query.limit()))
	values.Set("orderBy", "email")
	values.Set("projection", "basic")
	values.Set("viewType", "admin_view")
	values.Set("fields", "nextPageToken,users(primaryEmail,name/fullName,suspended)")

	if search := directorySearch(query.Search); search != "" {
		values.Set("query", search)
	}

	if token != "" {
		values.Set("pageToken", token)
	}

	var listing struct {
		NextPageToken string `json:"nextPageToken"`
		Users         []struct {
			PrimaryEmail string `json:"primaryEmail"`
			Suspended    bool   `json:"suspended"`
			Name         struct {
				FullName string `json:"fullName"`
			} `json:"name"`
		} `json:"users"`
	}

	if err := s.directoryAuth().getJSON(ctx, StageList, GoogleDirectoryUsersURL+"?"+values.Encode(), &listing); err != nil {
		return TargetPage{}, err
	}

	page := TargetPage{NextCursor: listing.NextPageToken}

	for _, user := range listing.Users {
		if user.Suspended || user.PrimaryEmail == "" {
			continue
		}

		email := strings.ToLower(user.PrimaryEmail)

		label := user.Name.FullName
		if label == "" {
			label = email
		}

		page.Items = append(page.Items, TargetOption{
			Value:       email,
			Label:       label,
			Description: email,
			Kind:        TargetKindUserDrive,
		})
	}

	return page, nil
}

func directorySearch(search string) string {
	term := strings.NewReplacer("'", "", `"`, "", "*", "", ":", "").Replace(strings.TrimSpace(search))
	if term == "" {
		return ""
	}

	if strings.Contains(term, "@") {
		return "email:" + term + "*"
	}

	return term
}

func (s *GraphSource) Sites(ctx context.Context, query TargetQuery) (TargetPage, error) {
	endpoint := query.Cursor
	if endpoint == "" {
		search := query.Search
		if search == "" {
			search = "*"
		}

		values := url.Values{}
		values.Set("search", search)
		values.Set("$select", "id,displayName,name,webUrl")
		values.Set("$top", strconv.Itoa(query.limit()))

		endpoint = GraphBaseURL + "/sites?" + values.Encode()
	}

	var listing struct {
		Value []struct {
			DisplayName string `json:"displayName"`
			Name        string `json:"name"`
			WebURL      string `json:"webUrl"`
		} `json:"value"`
		Next string `json:"@odata.nextLink"`
	}

	if err := s.pagedJSON(ctx, endpoint, &listing); err != nil {
		return TargetPage{}, err
	}

	page := TargetPage{NextCursor: listing.Next}

	for _, site := range listing.Value {
		parsed, err := url.Parse(site.WebURL)
		if err != nil || parsed.Host == "" || strings.Contains(parsed.Host, "-my.sharepoint.") {
			continue
		}

		label := site.DisplayName
		if label == "" {
			label = site.Name
		}

		path := strings.TrimSuffix(parsed.Path, "/")
		if path == "" {
			path = "/"
		}

		page.Items = append(page.Items, TargetOption{
			Value:       strings.TrimSuffix(site.WebURL, "/"),
			Label:       label,
			Description: path,
			Expandable:  true,
		})
	}

	return page, nil
}

func (s *GraphSource) Libraries(ctx context.Context, query TargetQuery) (TargetPage, error) {
	endpoint := query.Cursor
	if endpoint == "" {
		target, err := parseSharePointTarget(query.Parent)
		if err != nil {
			return TargetPage{}, err
		}

		siteID, err := s.resolveSite(ctx, target)
		if err != nil {
			return TargetPage{}, err
		}

		endpoint = GraphBaseURL + "/sites/" + url.PathEscape(siteID) + "/drives?$select=id,name,webUrl"
	}

	var listing struct {
		Value []graphItem `json:"value"`
		Next  string      `json:"@odata.nextLink"`
	}

	if err := s.pagedJSON(ctx, endpoint, &listing); err != nil {
		return TargetPage{}, err
	}

	page := TargetPage{NextCursor: listing.Next}
	search := strings.ToLower(query.Search)

	for _, drive := range listing.Value {
		if search != "" && !strings.Contains(strings.ToLower(drive.Name), search) {
			continue
		}

		page.Items = append(page.Items, TargetOption{Value: drive.Name, Label: drive.Name, Description: drive.WebURL})
	}

	return page, nil
}

func (s *GraphSource) Users(ctx context.Context, query TargetQuery) (TargetPage, error) {
	endpoint := query.Cursor
	if endpoint == "" {
		values := url.Values{}
		values.Set("$select", "id,displayName,userPrincipalName,mail")
		values.Set("$top", strconv.Itoa(query.limit()))

		if query.Search != "" {
			term := strings.ReplaceAll(query.Search, "'", "''")
			values.Set("$filter", "startswith(displayName,'"+term+"') or startswith(userPrincipalName,'"+term+"') or startswith(mail,'"+term+"')")
		}

		endpoint = GraphBaseURL + "/users?" + values.Encode()
	}

	var listing struct {
		Value []struct {
			DisplayName       string `json:"displayName"`
			UserPrincipalName string `json:"userPrincipalName"`
			Mail              string `json:"mail"`
		} `json:"value"`
		Next string `json:"@odata.nextLink"`
	}

	if err := s.pagedJSON(ctx, endpoint, &listing); err != nil {
		return TargetPage{}, err
	}

	page := TargetPage{NextCursor: listing.Next}

	for _, user := range listing.Value {
		if user.UserPrincipalName == "" {
			continue
		}

		label := user.DisplayName
		if label == "" {
			label = user.UserPrincipalName
		}

		description := user.Mail
		if description == "" {
			description = user.UserPrincipalName
		}

		page.Items = append(page.Items, TargetOption{
			Value:       user.UserPrincipalName,
			Label:       label,
			Description: description,
		})
	}

	return page, nil
}

func (s *GraphSource) pagedJSON(ctx context.Context, endpoint string, into any) error {
	if !strings.HasPrefix(endpoint, GraphBaseURL+"/") {
		return &Error{Provider: ProviderEntra, Stage: StageList, Reason: ReasonUnknown, code: "untrusted_next_link"}
	}

	return s.auth.getJSON(ctx, StageList, endpoint, into)
}
