package datadiscovery

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"unicode/utf8"

	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/datadiscovery/provider"
	"dpdp-backend/internal/datadiscovery/strategy"
	"dpdp-backend/internal/db"
)

const (
	maxTargetSearchRunes = 100
	maxTargetParentBytes = 1024
)

type TargetInput struct {
	SourceType string
	Search     string
	Parent     string
	Cursor     string
	Limit      int
}

type TargetListing struct {
	Items      []provider.TargetOption
	NextCursor string
}

func (s *ConfigurationService) Targets(
	ctx context.Context,
	customerID, id int,
	in TargetInput,
) (TargetListing, error) {
	input, err := normalizeTargetInput(in)
	if err != nil {
		return TargetListing{}, err
	}

	row, err := s.repo.Find(ctx, customerID, id)
	if err != nil {
		return TargetListing{}, configurationError(err, "")
	}

	if row.Status != db.DiscoveryStatusActive {
		return TargetListing{}, utils.ErrUnknownConfiguration
	}

	selected, ok := s.registry.For(row.ConfigurationType)
	if !ok {
		return TargetListing{}, utils.ErrInvalidConfigurationType
	}

	cursor, err := s.openCursor(customerID, id, input)
	if err != nil {
		return TargetListing{}, err
	}

	credential, err := s.OpenCredential(ctx, customerID, id)
	if err != nil {
		return TargetListing{}, err
	}

	browseCtx, cancel := context.WithTimeout(ctx, s.testTimeout)
	defer cancel()

	page, err := selected.BrowseTargets(browseCtx, input.SourceType, row.Config, credential, provider.TargetQuery{
		Search: input.Search,
		Parent: input.Parent,
		Cursor: cursor,
		Limit:  input.Limit,
	})
	if err != nil {
		return TargetListing{}, s.browseError(ctx, customerID, selected, err)
	}

	next, err := s.sealCursor(customerID, id, input, page.NextCursor)
	if err != nil {
		return TargetListing{}, err
	}

	return TargetListing{Items: uniqueTargets(page.Items), NextCursor: next}, nil
}

func normalizeTargetInput(in TargetInput) (TargetInput, error) {
	in.SourceType = strings.TrimSpace(in.SourceType)
	if !validSourceType(in.SourceType) {
		return TargetInput{}, utils.ErrInvalidSourceType
	}

	in.Search = utils.NormalizeName(in.Search)
	if utf8.RuneCountInString(in.Search) > maxTargetSearchRunes {
		return TargetInput{}, utils.ErrTooManyItems
	}

	in.Parent = strings.TrimSpace(in.Parent)
	if in.Parent != "" && (in.SourceType != db.SourceTypeSharePoint || len(in.Parent) > maxTargetParentBytes) {
		return TargetInput{}, utils.ErrInvalidDiscoveryTarget
	}

	switch {
	case in.Limit <= 0:
		in.Limit = provider.DefaultTargetLimit
	case in.Limit > provider.MaxTargetLimit:
		in.Limit = provider.MaxTargetLimit
	}

	in.Cursor = strings.TrimSpace(in.Cursor)

	return in, nil
}

func (s *ConfigurationService) sealCursor(customerID, id int, in TargetInput, raw string) (string, error) {
	if raw == "" {
		return "", nil
	}

	sealed, err := s.box.Seal([]byte(raw), cursorAAD(customerID, id, in))
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (s *ConfigurationService) openCursor(customerID, id int, in TargetInput) (string, error) {
	if in.Cursor == "" {
		return "", nil
	}

	sealed, err := base64.RawURLEncoding.DecodeString(in.Cursor)
	if err != nil {
		return "", utils.ErrInvalidCursor
	}

	raw, err := s.box.Open(sealed, cursorAAD(customerID, id, in))
	if err != nil {
		return "", utils.ErrInvalidCursor
	}

	return string(raw), nil
}

func cursorAAD(customerID, id int, in TargetInput) []byte {
	return []byte("dpdp:dd:targets:" + strconv.Itoa(customerID) + ":" + strconv.Itoa(id) +
		":" + in.SourceType + ":" + in.Parent + ":" + in.Search)
}

func uniqueTargets(items []provider.TargetOption) []provider.TargetOption {
	seen := make(map[string]bool, len(items))
	unique := make([]provider.TargetOption, 0, len(items))

	for _, item := range items {
		item.Value = strings.TrimSpace(item.Value)
		if item.Value == "" || seen[item.Value] {
			continue
		}

		if item.Label = strings.TrimSpace(item.Label); item.Label == "" {
			item.Label = item.Value
		}

		seen[item.Value] = true
		unique = append(unique, item)
	}

	return unique
}

func (s *ConfigurationService) browseError(
	ctx context.Context,
	customerID int,
	selected strategy.ConfigurationStrategy,
	err error,
) error {
	var providerErr *provider.Error
	if !errors.As(err, &providerErr) {
		return err
	}

	slog.WarnContext(ctx, "data discovery target listing failed",
		"customer_id", customerID,
		"configuration_type", selected.Type(),
		"provider", providerErr.Provider,
		"stage", providerErr.Stage,
		"reason", providerErr.Reason,
		"http_status", providerErr.HTTPStatus,
		"provider_code", providerErr.Code(),
		"request_id", providerErr.RequestID(),
	)

	if providerErr.Reason == provider.ReasonUnavailable || providerErr.Reason == provider.ReasonRateLimited {
		return utils.ErrProviderUnavailable
	}

	return utils.TargetBrowseFailed(providerErr.Reason)
}
