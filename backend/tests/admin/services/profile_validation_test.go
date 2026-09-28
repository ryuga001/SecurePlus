package services_test

import (
	"errors"
	"strings"
	"testing"

	profilesvc "dpdp-backend/internal/admin/services/profile"
	"dpdp-backend/internal/admin/utils"
)

func text(value string) *string { return &value }

func TestNormalizeProfileInputCollapsesWhitespace(t *testing.T) {
	input, err := profilesvc.NormalizeProfileInput(profilesvc.ProfileInput{
		OrgName:   text("  Acme   India  "),
		FirstName: text(" Rahul "),
		LastName:  text("  Kumar   Singh "),
	})
	if err != nil {
		t.Fatalf("NormalizeProfileInput returned %v", err)
	}

	if *input.OrgName != "Acme India" || *input.FirstName != "Rahul" || *input.LastName != "Kumar Singh" {
		t.Fatalf("normalized = %q %q %q", *input.OrgName, *input.FirstName, *input.LastName)
	}
}

func TestNormalizeProfileInputLeavesOmittedFieldsNil(t *testing.T) {
	input, err := profilesvc.NormalizeProfileInput(profilesvc.ProfileInput{FirstName: text("Asha")})
	if err != nil {
		t.Fatalf("NormalizeProfileInput returned %v", err)
	}

	if input.OrgName != nil || input.LastName != nil {
		t.Fatalf("omitted fields were set: %+v", input)
	}
}

func TestNormalizeProfileInputValidatesOrgNameLength(t *testing.T) {
	for _, value := range []string{"", "   ", "A", strings.Repeat("x", 101)} {
		_, err := profilesvc.NormalizeProfileInput(profilesvc.ProfileInput{OrgName: text(value)})
		if !errors.Is(err, utils.ErrOrgNameNeeded) {
			t.Errorf("org name %q: error = %v, want ErrOrgNameNeeded", value, err)
		}
	}

	for _, value := range []string{"AB", strings.Repeat("x", 100), strings.Repeat("組", 100)} {
		if _, err := profilesvc.NormalizeProfileInput(profilesvc.ProfileInput{OrgName: text(value)}); err != nil {
			t.Errorf("org name of %d runes rejected: %v", len([]rune(value)), err)
		}
	}
}

func TestNormalizeProfileInputRequiresPersonNames(t *testing.T) {
	for _, input := range []profilesvc.ProfileInput{
		{FirstName: text("  ")},
		{LastName: text("")},
	} {
		if _, err := profilesvc.NormalizeProfileInput(input); !errors.Is(err, utils.ErrNameNeeded) {
			t.Errorf("error = %v, want ErrNameNeeded", err)
		}
	}
}

func TestNormalizeProfileInputLimitsPersonNames(t *testing.T) {
	_, err := profilesvc.NormalizeProfileInput(profilesvc.ProfileInput{LastName: text(strings.Repeat("a", 51))})
	if !errors.Is(err, utils.ErrNameTooLong) {
		t.Fatalf("error = %v, want ErrNameTooLong", err)
	}

	if _, err := profilesvc.NormalizeProfileInput(profilesvc.ProfileInput{LastName: text(strings.Repeat("é", 50))}); err != nil {
		t.Fatalf("50-rune name rejected: %v", err)
	}
}
