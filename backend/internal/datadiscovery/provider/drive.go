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
	driveFolderMIME   = "application/vnd.google-apps.folder"
	driveNativePrefix = "application/vnd.google-apps."
	drivePageSize     = "1000"
	driveFileFields   = "nextPageToken,files(id,name,mimeType,size,modifiedTime)"
)

var ErrInvalidDriveTarget = errors.New("target does not identify a Google shared drive")

type driveExport struct {
	mime      string
	extension string
}

var driveExports = map[string]driveExport{
	"application/vnd.google-apps.document": {
		mime:      "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		extension: ".docx",
	},
	"application/vnd.google-apps.spreadsheet": {
		mime:      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		extension: ".xlsx",
	},
	"application/vnd.google-apps.presentation": {
		mime:      "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		extension: ".pptx",
	},
}

var driveQueryEscaper = strings.NewReplacer(`\`, `\\`, `'`, `\'`)

type DriveSource struct {
	auth *authorized
}

type driveScope struct {
	driveID string
	root    string
}

type driveFile struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	MIMEType     string    `json:"mimeType"`
	Size         int64     `json:"size,string"`
	ModifiedTime time.Time `json:"modifiedTime"`
}

type driveFileList struct {
	Files         []driveFile `json:"files"`
	NextPageToken string      `json:"nextPageToken"`
}

func NewDriveSource(
	ctx context.Context,
	client *Client,
	clientEmail, subject, tokenURI, privateKey string,
) (*DriveSource, error) {
	auth := newAuthorized(client, ProviderGoogle, map[string]string{"Accept": "application/json"},
		func(ctx context.Context) (Token, error) {
			return client.GoogleAccessToken(ctx, clientEmail, subject, tokenURI, privateKey)
		})

	if _, err := auth.bearer(ctx); err != nil {
		return nil, err
	}

	return &DriveSource{auth: auth}, nil
}

func (s *DriveSource) List(ctx context.Context, target string, emit func(File) error) error {
	segments := pathSegments(target)
	if len(segments) == 0 {
		return ErrInvalidDriveTarget
	}

	scope, err := s.resolveDrive(ctx, segments[0])
	if err != nil {
		return err
	}

	root, err := s.resolveFolder(ctx, scope, segments[1:])
	if err != nil {
		return err
	}

	return walk(ctx, folderRef{id: root}, s.children(scope), emit)
}

func (s *DriveSource) Open(ctx context.Context, file File) (io.ReadCloser, error) {
	if file.Key == "" {
		return nil, ErrInvalidDriveTarget
	}

	endpoint := GoogleDriveBaseURL + "/files/" + url.PathEscape(file.Key)

	if export, ok := driveExports[file.MIMEType]; ok {
		endpoint += "/export?mimeType=" + url.QueryEscape(export.mime)
	} else {
		endpoint += "?alt=media&supportsAllDrives=true"
	}

	response, err := s.auth.get(ctx, StageDownload, endpoint)
	if err != nil {
		return nil, err
	}

	return response.Body, nil
}

func (s *DriveSource) children(scope driveScope) pageFetcher {
	return func(ctx context.Context, folder folderRef, cursor string) (folderPage, error) {
		query := s.query(scope, "'"+driveQueryEscaper.Replace(folder.id)+"' in parents and trashed = false")
		query.Set("fields", driveFileFields)
		query.Set("pageSize", drivePageSize)

		if cursor != "" {
			query.Set("pageToken", cursor)
		}

		var list driveFileList
		if err := s.auth.getJSON(ctx, StageList, GoogleDriveBaseURL+"/files?"+query.Encode(), &list); err != nil {
			return folderPage{}, err
		}

		page := folderPage{next: list.NextPageToken}

		for _, item := range list.Files {
			path := childPath(folder.path, item.Name)

			if item.MIMEType == driveFolderMIME {
				page.folders = append(page.folders, folderRef{id: item.ID, path: path})

				continue
			}

			if strings.HasPrefix(item.MIMEType, driveNativePrefix) {
				export, ok := driveExports[item.MIMEType]
				if !ok {
					continue
				}

				path += export.extension
			}

			page.files = append(page.files, File{
				Key:        item.ID,
				Name:       path,
				MIMEType:   item.MIMEType,
				Size:       item.Size,
				ModifiedAt: item.ModifiedTime,
			})
		}

		return page, nil
	}
}

func (s *DriveSource) resolveDrive(ctx context.Context, name string) (driveScope, error) {
	switch strings.ToLower(name) {
	case "my drive", "mydrive", "root":
		return driveScope{root: "root"}, nil
	}

	query := url.Values{}
	query.Set("q", "name = '"+driveQueryEscaper.Replace(name)+"'")
	query.Set("pageSize", "10")
	query.Set("fields", "drives(id,name)")

	var drives struct {
		Drives []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"drives"`
	}

	if err := s.auth.getJSON(ctx, StageList, GoogleDriveBaseURL+"/drives?"+query.Encode(), &drives); err != nil {
		return driveScope{}, err
	}

	for _, drive := range drives.Drives {
		if drive.Name == name {
			return driveScope{driveID: drive.ID, root: drive.ID}, nil
		}
	}

	var byID struct {
		ID string `json:"id"`
	}

	err := s.auth.getJSON(ctx, StageList, GoogleDriveBaseURL+"/drives/"+url.PathEscape(name)+"?fields=id", &byID)
	if err == nil && byID.ID != "" {
		return driveScope{driveID: byID.ID, root: byID.ID}, nil
	}

	var failure *Error
	if err != nil && !(errors.As(err, &failure) && failure.Reason == ReasonNotFound) {
		return driveScope{}, err
	}

	return driveScope{}, notFound(ProviderGoogle, StageList, "shared_drive_not_found")
}

func (s *DriveSource) resolveFolder(ctx context.Context, scope driveScope, segments []string) (string, error) {
	parent := scope.root

	for _, segment := range segments {
		query := s.query(scope, "'"+driveQueryEscaper.Replace(parent)+"' in parents and name = '"+
			driveQueryEscaper.Replace(segment)+"' and mimeType = '"+driveFolderMIME+"' and trashed = false")
		query.Set("fields", "files(id)")
		query.Set("pageSize", "2")

		var list driveFileList
		if err := s.auth.getJSON(ctx, StageList, GoogleDriveBaseURL+"/files?"+query.Encode(), &list); err != nil {
			return "", err
		}

		if len(list.Files) == 0 {
			return "", notFound(ProviderGoogle, StageList, "folder_not_found")
		}

		parent = list.Files[0].ID
	}

	return parent, nil
}

func (s *DriveSource) query(scope driveScope, q string) url.Values {
	query := url.Values{}
	query.Set("q", q)
	query.Set("supportsAllDrives", "true")
	query.Set("includeItemsFromAllDrives", "true")

	if scope.driveID != "" {
		query.Set("corpora", "drive")
		query.Set("driveId", scope.driveID)
	}

	return query
}
