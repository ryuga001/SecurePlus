package strategy_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/datadiscovery/strategy"
	"dpdp-backend/internal/db"
)

func TestAzureBlobTargetsKeepPrefixCase(t *testing.T) {
	selected, ok := strategy.DefaultPolicyRegistry().For(db.SourceTypeAzureBlob)
	if !ok {
		t.Fatal("azure blob strategy missing")
	}

	targets, err := selected.NormalizeTargets([]string{"/Finance/Reports/Q1", "finance", "FINANCE"})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}

	if want := []string{"finance", "finance/Reports/Q1"}; !reflect.DeepEqual(targets, want) {
		t.Fatalf("targets = %v, want %v", targets, want)
	}

	if _, err := selected.NormalizeTargets([]string{"ab/prefix"}); !errors.Is(err, utils.ErrInvalidDiscoveryTarget) {
		t.Fatalf("short container error = %v", err)
	}
}

func TestOnlyAzureBlobIsScannable(t *testing.T) {
	registry := strategy.DefaultPolicyRegistry()

	for _, source := range []string{db.SourceTypeSharePoint, db.SourceTypeOneDrive, db.SourceTypeGoogleDrive, db.SourceTypeAWSS3} {
		selected, _ := registry.For(source)

		if selected.Scannable() {
			t.Fatalf("%s should not be scannable yet", source)
		}

		if _, err := selected.Connect(context.Background(), nil, nil, nil); !errors.Is(err, strategy.ErrSourceUnsupported) {
			t.Fatalf("%s connect error = %v", source, err)
		}
	}

	blob, _ := registry.For(db.SourceTypeAzureBlob)
	if !blob.Scannable() {
		t.Fatal("azure blob should be scannable")
	}
}

func TestAzureBlobConnectValidatesCredential(t *testing.T) {
	blob, _ := strategy.DefaultPolicyRegistry().For(db.SourceTypeAzureBlob)

	config := db.StringMap{"storageAccount": "acct", "tenantId": "tenant", "clientId": "client"}

	if _, err := blob.Connect(context.Background(), nil, config, []byte("not json")); !errors.Is(err, utils.ErrCredentialUnavailable) {
		t.Fatalf("credential error = %v", err)
	}

	if _, err := blob.Connect(context.Background(), nil, db.StringMap{}, []byte(`{"clientSecret":"s"}`)); !errors.Is(err, utils.ErrConfigFieldNeeded) {
		t.Fatalf("config error = %v", err)
	}
}
