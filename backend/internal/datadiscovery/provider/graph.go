package provider

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"time"
)

const (
	GraphSharePoint = "SHARE_POINT"
	GraphOneDrive   = "ONE_DRIVE"

	graphPageSize   = "1000"
	graphItemFields = "id,name,size,file,folder,package,remoteItem,lastModifiedDateTime"
)

var ErrInvalidGraphTarget = errors.New("target does not identify a SharePoint site or OneDrive user")

type GraphSource struct {
	auth *authorized
	mode string
}

type graphItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModifiedDateTime"`
	WebURL       string    `json:"webUrl"`
	File         *struct {
		MimeType string `json:"mimeType"`
	} `json:"file"`
	Folder     *struct{} `json:"folder"`
	Package    *struct{} `json:"package"`
	RemoteItem *struct{} `json:"remoteItem"`
}

type graphPage struct {
	Value []graphItem `json:"value"`
	Next  string      `json:"@odata.nextLink"`
}

type sharePointTarget struct {
	host    string
	site    string
	library string
	folder  string
}

func NewGraphSource(
	ctx context.Context,
	client *Client,
	tenantID, clientID, secret, mode string,
) (*GraphSource, error) {
	auth := newAuthorized(client, ProviderEntra, map[string]string{"Accept": "application/json"},
		func(ctx context.Context) (Token, error) {
			return client.EntraAccessToken(ctx, tenantID, clientID, secret)
		})

	if _, err := auth.bearer(ctx); err != nil {
		return nil, err
	}

	return &GraphSource{auth: auth, mode: mode}, nil
}

func (s *GraphSource) List(ctx context.Context, target string, emit func(File) error) error {
	driveID, rootID, err := s.resolve(ctx, target)
	if err != nil {
		return err
	}

	return walk(ctx, folderRef{id: rootID}, s.children(driveID), emit)
}

func (s *GraphSource) Open(ctx context.Context, file File) (io.ReadCloser, error) {
	driveID, itemID, ok := strings.Cut(file.Key, "/")
	if !ok || driveID == "" || itemID == "" {
		return nil, ErrInvalidGraphTarget
	}

	response, err := s.auth.get(ctx, StageDownload,
		GraphBaseURL+"/drives/"+url.PathEscape(driveID)+"/items/"+url.PathEscape(itemID)+"/content")
	if err != nil {
		return nil, err
	}

	return response.Body, nil
}

func (s *GraphSource) children(driveID string) pageFetcher {
	return func(ctx context.Context, folder folderRef, cursor string) (folderPage, error) {
		endpoint := cursor
		if endpoint == "" {
			query := url.Values{}
			query.Set("$top", graphPageSize)
			query.Set("$select", graphItemFields)

			endpoint = GraphBaseURL + "/drives/" + url.PathEscape(driveID) +
				"/items/" + url.PathEscape(folder.id) + "/children?" + query.Encode()
		}

		if !strings.HasPrefix(endpoint, GraphBaseURL+"/") {
			return folderPage{}, &Error{Provider: ProviderEntra, Stage: StageList, Reason: ReasonUnknown, code: "untrusted_next_link"}
		}

		var page graphPage
		if err := s.auth.getJSON(ctx, StageList, endpoint, &page); err != nil {
			return folderPage{}, err
		}

		result := folderPage{next: page.Next}

		for _, item := range page.Value {
			path := childPath(folder.path, item.Name)

			switch {
			case item.RemoteItem != nil || item.Package != nil:
				continue
			case item.Folder != nil:
				result.folders = append(result.folders, folderRef{id: item.ID, path: path})
			case item.File != nil:
				result.files = append(result.files, File{
					Key:        driveID + "/" + item.ID,
					Name:       path,
					MIMEType:   item.File.MimeType,
					Size:       item.Size,
					ModifiedAt: item.LastModified,
				})
			}
		}

		return result, nil
	}
}

func (s *GraphSource) resolve(ctx context.Context, target string) (string, string, error) {
	if s.mode == GraphOneDrive {
		return s.resolveOneDrive(ctx, target)
	}

	return s.resolveSharePoint(ctx, target)
}

func (s *GraphSource) resolveOneDrive(ctx context.Context, target string) (string, string, error) {
	segments := pathSegments(target)
	if len(segments) == 0 {
		return "", "", ErrInvalidGraphTarget
	}

	var drive graphItem
	if err := s.auth.getJSON(ctx, StageList,
		GraphBaseURL+"/users/"+url.PathEscape(segments[0])+"/drive?$select=id", &drive); err != nil {
		return "", "", err
	}

	rootID, err := s.resolveFolder(ctx, drive.ID, strings.Join(segments[1:], "/"))

	return drive.ID, rootID, err
}

func (s *GraphSource) resolveSharePoint(ctx context.Context, raw string) (string, string, error) {
	target, err := parseSharePointTarget(raw)
	if err != nil {
		return "", "", err
	}

	if target.host == "" {
		var root struct {
			SiteCollection struct {
				Hostname string `json:"hostname"`
			} `json:"siteCollection"`
		}

		if err := s.auth.getJSON(ctx, StageList, GraphBaseURL+"/sites/root?$select=siteCollection", &root); err != nil {
			return "", "", err
		}

		target.host = root.SiteCollection.Hostname
	}

	endpoint := GraphBaseURL + "/sites/" + url.PathEscape(target.host)
	if target.site != "/" {
		endpoint += ":" + escapePath(target.site)
	}

	var site graphItem
	if err := s.auth.getJSON(ctx, StageList, endpoint+"?$select=id", &site); err != nil {
		return "", "", err
	}

	driveID, folder, err := s.siteDrive(ctx, site.ID, target.library, target.folder)
	if err != nil {
		return "", "", err
	}

	rootID, err := s.resolveFolder(ctx, driveID, folder)

	return driveID, rootID, err
}

func (s *GraphSource) siteDrive(ctx context.Context, siteID, library, folder string) (string, string, error) {
	base := GraphBaseURL + "/sites/" + url.PathEscape(siteID)

	if library != "" {
		var drives struct {
			Value []graphItem `json:"value"`
		}

		if err := s.auth.getJSON(ctx, StageList, base+"/drives?$select=id,name,webUrl", &drives); err != nil {
			return "", "", err
		}

		for _, drive := range drives.Value {
			if strings.EqualFold(drive.Name, library) || strings.EqualFold(lastURLSegment(drive.WebURL), library) {
				return drive.ID, folder, nil
			}
		}

		folder = strings.Trim(library+"/"+folder, "/")
	}

	var drive graphItem
	if err := s.auth.getJSON(ctx, StageList, base+"/drive?$select=id", &drive); err != nil {
		return "", "", err
	}

	return drive.ID, folder, nil
}

func (s *GraphSource) resolveFolder(ctx context.Context, driveID, folder string) (string, error) {
	endpoint := GraphBaseURL + "/drives/" + url.PathEscape(driveID) + "/root"
	if folder = strings.Trim(folder, "/"); folder != "" {
		endpoint += ":" + escapePath(folder)
	}

	var item graphItem
	if err := s.auth.getJSON(ctx, StageList, endpoint+"?$select=id,folder", &item); err != nil {
		return "", err
	}

	if folder != "" && item.Folder == nil {
		return "", notFound(ProviderEntra, StageList, "not_a_folder")
	}

	return item.ID, nil
}

func parseSharePointTarget(raw string) (sharePointTarget, error) {
	value := strings.TrimSpace(raw)
	target := sharePointTarget{}

	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Host == "" {
			return sharePointTarget{}, ErrInvalidGraphTarget
		}

		target.host = parsed.Host
		value = parsed.Path
	}

	parts := strings.Split(value, "/")
	index := 0

	for index < len(parts) && parts[index] == "" {
		index++
	}

	target.site = "/"
	if index+1 < len(parts) && (strings.EqualFold(parts[index], "sites") || strings.EqualFold(parts[index], "teams")) {
		if parts[index+1] == "" {
			return sharePointTarget{}, ErrInvalidGraphTarget
		}

		target.site = "/" + parts[index] + "/" + parts[index+1]
		index += 2
	}

	if index < len(parts) && parts[index] != "" {
		target.library = parts[index]
		index++
	}

	target.folder = strings.Join(pathSegments(strings.Join(parts[min(index, len(parts)):], "/")), "/")

	return target, nil
}

func pathSegments(value string) []string {
	segments := make([]string, 0, 4)

	for _, segment := range strings.Split(value, "/") {
		if segment = strings.TrimSpace(segment); segment != "" {
			segments = append(segments, segment)
		}
	}

	return segments
}

func escapePath(value string) string {
	segments := pathSegments(value)
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}

	return "/" + strings.Join(segments, "/")
}

func lastURLSegment(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	segments := pathSegments(parsed.Path)
	if len(segments) == 0 {
		return ""
	}

	return segments[len(segments)-1]
}
