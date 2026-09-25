package strategy

import (
	"context"
	"slices"
	"strings"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/datadiscovery/provider"
	"dpdp-backend/internal/db"
)

const maxTargets = 200

type targetStrategy struct {
	sourceType        string
	configurationType string
	normalize         func(raw string) string
	connect           connectFunc
}

func (s targetStrategy) SourceType() string        { return s.sourceType }
func (s targetStrategy) ConfigurationType() string { return s.configurationType }
func (s targetStrategy) Scannable() bool           { return s.connect != nil }

func (s targetStrategy) Connect(
	ctx context.Context,
	client *provider.Client,
	config db.StringMap,
	credential []byte,
) (Source, error) {
	if s.connect == nil {
		return nil, ErrSourceUnsupported
	}

	return s.connect(ctx, client, config, credential)
}

func (s targetStrategy) NormalizeTargets(targets []string) ([]string, error) {
	seen := make(map[string]bool, len(targets))
	values := make([]string, 0, len(targets))

	for _, raw := range targets {
		value := s.normalize(raw)
		if value == "" {
			return nil, utils.ErrInvalidDiscoveryTarget
		}
		if seen[value] {
			continue
		}

		seen[value] = true
		values = append(values, value)
	}

	if len(values) == 0 {
		return nil, utils.ErrDiscoveryTargetsNeeded
	}
	if len(values) > maxTargets {
		return nil, utils.ErrTooManyItems
	}

	slices.Sort(values)

	return values, nil
}

func sharePointStrategy() PolicyStrategy {
	return targetStrategy{
		sourceType:        db.SourceTypeSharePoint,
		configurationType: db.ConfigurationTypeEntra,
		normalize:         normalizePath,
		connect:           connectSharePoint,
	}
}

func oneDriveStrategy() PolicyStrategy {
	return targetStrategy{
		sourceType:        db.SourceTypeOneDrive,
		configurationType: db.ConfigurationTypeEntra,
		normalize:         normalizePath,
		connect:           connectOneDrive,
	}
}

func azureBlobStrategy() PolicyStrategy {
	return targetStrategy{
		sourceType:        db.SourceTypeAzureBlob,
		configurationType: db.ConfigurationTypeAzureStorage,
		normalize:         normalizeContainerPath,
		connect:           connectAzureBlob,
	}
}

func googleDriveStrategy() PolicyStrategy {
	return targetStrategy{
		sourceType:        db.SourceTypeGoogleDrive,
		configurationType: db.ConfigurationTypeGoogleSA,
		normalize:         normalizePath,
		connect:           connectGoogleDrive,
	}
}

func awsS3Strategy() PolicyStrategy {
	return targetStrategy{
		sourceType:        db.SourceTypeAWSS3,
		configurationType: db.ConfigurationTypeAWSIAM,
		normalize:         normalizeBucketPath,
		connect:           connectAWSS3,
	}
}

func normalizePath(raw string) string {
	value := strings.TrimSpace(raw)

	if value == "" || len(value) > 1024 {
		return ""
	}
	if strings.ContainsAny(value, "\x00\r\n\t") {
		return ""
	}

	return value
}

func normalizeBucketPath(raw string) string {
	value := normalizePath(raw)
	if value == "" {
		return ""
	}

	bucket, prefix, hasPrefix := strings.Cut(strings.TrimPrefix(value, "/"), "/")
	bucket = strings.ToLower(bucket)

	if len(bucket) < 3 || len(bucket) > 63 {
		return ""
	}

	for _, char := range bucket {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' && char != '.' {
			return ""
		}
	}

	if !hasPrefix || prefix == "" {
		return bucket
	}

	return bucket + "/" + prefix
}

func normalizeContainerPath(raw string) string {
	value := normalizePath(raw)
	if value == "" {
		return ""
	}

	container, prefix, hasPrefix := strings.Cut(strings.TrimPrefix(value, "/"), "/")
	container = strings.ToLower(container)

	if len(container) < 3 || len(container) > 63 {
		return ""
	}

	for _, char := range container {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
			return ""
		}
	}

	if !hasPrefix || prefix == "" {
		return container
	}

	return container + "/" + prefix
}
