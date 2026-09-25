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

func TestS3TargetsKeepPrefixCase(t *testing.T) {
	selected, _ := strategy.DefaultPolicyRegistry().For(db.SourceTypeAWSS3)

	targets, err := selected.NormalizeTargets([]string{"My-Bucket/Raw/2025/", "/my-bucket"})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}

	if want := []string{"my-bucket", "my-bucket/Raw/2025/"}; !reflect.DeepEqual(targets, want) {
		t.Fatalf("targets = %v, want %v", targets, want)
	}
}

func TestAllSourcesAreScannable(t *testing.T) {
	registry := strategy.DefaultPolicyRegistry()

	for _, source := range []string{
		db.SourceTypeSharePoint, db.SourceTypeOneDrive, db.SourceTypeAzureBlob,
		db.SourceTypeGoogleDrive, db.SourceTypeAWSS3,
	} {
		selected, ok := registry.For(source)
		if !ok || !selected.Scannable() {
			t.Fatalf("%s should be scannable", source)
		}
	}
}

func TestConnectValidatesCredentialAndConfig(t *testing.T) {
	registry := strategy.DefaultPolicyRegistry()

	cases := []struct {
		source     string
		config     db.StringMap
		credential string
	}{
		{db.SourceTypeSharePoint, db.StringMap{"tenantId": "t", "clientId": "c"}, `{"clientSecret":"s"}`},
		{db.SourceTypeOneDrive, db.StringMap{"tenantId": "t", "clientId": "c"}, `{"clientSecret":"s"}`},
		{db.SourceTypeGoogleDrive, db.StringMap{"clientEmail": "sa@example.iam.gserviceaccount.com"}, `{"privateKey":"-----BEGIN PRIVATE KEY-----"}`},
		{db.SourceTypeAWSS3, db.StringMap{"accessKeyId": "AKIAABCDEFGHIJKLMNOP", "region": "ap-south-1"}, `{"secretAccessKey":"s"}`},
	}

	for _, tc := range cases {
		selected, _ := registry.For(tc.source)

		if _, err := selected.Connect(context.Background(), nil, tc.config, []byte("not json")); !errors.Is(err, utils.ErrCredentialUnavailable) {
			t.Fatalf("%s bad credential error = %v", tc.source, err)
		}

		if tc.source == db.SourceTypeGoogleDrive {
			if _, err := selected.Connect(context.Background(), nil, tc.config, []byte(tc.credential)); !errors.Is(err, utils.ErrCredentialUnavailable) {
				t.Fatalf("google malformed key error = %v", err)
			}

			continue
		}

		if _, err := selected.Connect(context.Background(), nil, db.StringMap{}, []byte(tc.credential)); !errors.Is(err, utils.ErrConfigFieldNeeded) {
			t.Fatalf("%s missing config error = %v", tc.source, err)
		}
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
