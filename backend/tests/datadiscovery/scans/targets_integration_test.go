//go:build integration

package scans_test

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"testing"
	"time"

	discoveryrepo "dpdp-backend/internal/admin/repositories/datadiscovery"
	discoverysvc "dpdp-backend/internal/admin/services/datadiscovery"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/datadiscovery/provider"
	"dpdp-backend/internal/datadiscovery/strategy"
	"dpdp-backend/internal/db"
)

func configurations(f fixture) *discoverysvc.ConfigurationService {
	return discoverysvc.NewConfigurationService(
		f.database,
		discoveryrepo.NewConfigurationRepository(f.database),
		strategy.DefaultConfigurationRegistry(provider.NewClient(time.Second)),
		f.box,
		time.Second,
	)
}

func TestIntegrationTargetBrowsingPreprocessing(t *testing.T) {
	f := setup(t)
	service := configurations(f)
	ctx := context.Background()

	blob := f.configuration(t, db.ConfigurationTypeAzureStorage, db.StringMap{
		"storageAccount": "acct",
		"tenantId":       "tenant",
		"clientId":       "client",
		"authMode":       db.AzureAuthModeServicePrincipal,
	})

	cases := []struct {
		name  string
		input discoverysvc.TargetInput
		want  error
	}{
		{"unknown source", discoverysvc.TargetInput{SourceType: "FTP"}, utils.ErrInvalidSourceType},
		{"parent outside SharePoint", discoverysvc.TargetInput{SourceType: db.SourceTypeAzureBlob, Parent: "/sites/x"}, utils.ErrInvalidDiscoveryTarget},
		{"garbage cursor", discoverysvc.TargetInput{SourceType: db.SourceTypeAzureBlob, Cursor: "%%%"}, utils.ErrInvalidCursor},
		{"source of another configuration type", discoverysvc.TargetInput{SourceType: db.SourceTypeAWSS3}, utils.ErrIncompatibleSource},
	}

	for _, tc := range cases {
		if _, err := service.Targets(ctx, f.customer.ID, blob.ID, tc.input); !errors.Is(err, tc.want) {
			t.Fatalf("%s: err = %v, want %v", tc.name, err, tc.want)
		}
	}

	foreign, err := f.box.Seal([]byte("marker-2"), []byte("dpdp:dd:targets:"+strconv.Itoa(f.customer.ID)+":"+strconv.Itoa(blob.ID)+":AZURE_BLOB::other-search"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	_, err = service.Targets(ctx, f.customer.ID, blob.ID, discoverysvc.TargetInput{
		SourceType: db.SourceTypeAzureBlob,
		Search:     "finance",
		Cursor:     base64.RawURLEncoding.EncodeToString(foreign),
	})
	if !errors.Is(err, utils.ErrInvalidCursor) {
		t.Fatalf("cursor from a different search must be rejected, err = %v", err)
	}

	if err := f.database.Exec("UPDATE data_discovery_configurations SET status = ? WHERE id = ?", db.DiscoveryStatusInactive, blob.ID).Error; err != nil {
		t.Fatalf("deactivate: %v", err)
	}

	if _, err := service.Targets(ctx, f.customer.ID, blob.ID, discoverysvc.TargetInput{SourceType: db.SourceTypeAzureBlob}); !errors.Is(err, utils.ErrUnknownConfiguration) {
		t.Fatalf("inactive configuration err = %v", err)
	}

	if _, err := service.Targets(ctx, f.customer.ID+1000, blob.ID, discoverysvc.TargetInput{SourceType: db.SourceTypeAzureBlob}); !errors.Is(err, utils.ErrDiscoveryConfigNotFound) {
		t.Fatalf("other tenant err = %v", err)
	}
}
