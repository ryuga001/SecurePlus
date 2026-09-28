package services_test

import (
	"errors"
	"strings"
	"testing"

	usersvc "dpdp-backend/internal/admin/services/dashboarduser"
	"dpdp-backend/internal/admin/utils"
)

func TestNormalizeUserInputNormalizesCreate(t *testing.T) {
	input, err := usersvc.NormalizeUserInput(usersvc.UserInput{
		FirstName: "  Asha ",
		LastName:  " Kumar   Rao ",
		Email:     "  Asha@Acme.COM ",
		RoleID:    7,
	}, true)
	if err != nil {
		t.Fatalf("NormalizeUserInput returned %v", err)
	}

	if input.FirstName != "Asha" || input.LastName != "Kumar Rao" || input.Email != "asha@acme.com" || input.RoleID != 7 {
		t.Fatalf("normalized = %+v", input)
	}
}

func TestNormalizeUserInputRequiresRole(t *testing.T) {
	for _, roleID := range []int{0, -2} {
		_, err := usersvc.NormalizeUserInput(usersvc.UserInput{FirstName: "Asha", LastName: "Rao", Email: "a@acme.com", RoleID: roleID}, true)
		if !errors.Is(err, utils.ErrUnknownRole) {
			t.Errorf("role %d: error = %v, want ErrUnknownRole", roleID, err)
		}
	}
}

func TestNormalizeUserInputValidatesNames(t *testing.T) {
	_, err := usersvc.NormalizeUserInput(usersvc.UserInput{FirstName: " ", LastName: "Rao", Email: "a@acme.com", RoleID: 1}, true)
	if !errors.Is(err, utils.ErrNameNeeded) {
		t.Fatalf("error = %v, want ErrNameNeeded", err)
	}

	_, err = usersvc.NormalizeUserInput(usersvc.UserInput{FirstName: "Asha", LastName: strings.Repeat("r", 51), Email: "a@acme.com", RoleID: 1}, true)
	if !errors.Is(err, utils.ErrNameTooLong) {
		t.Fatalf("error = %v, want ErrNameTooLong", err)
	}
}

func TestNormalizeUserInputValidatesEmailOnlyOnCreate(t *testing.T) {
	for _, email := range []string{"", "not-an-email", "a@b", "a b@acme.com", strings.Repeat("a", 95) + "@acme.com"} {
		_, err := usersvc.NormalizeUserInput(usersvc.UserInput{FirstName: "Asha", LastName: "Rao", Email: email, RoleID: 1}, true)
		if !errors.Is(err, utils.ErrInvalidEmail) {
			t.Errorf("email %q: error = %v, want ErrInvalidEmail", email, err)
		}
	}

	input, err := usersvc.NormalizeUserInput(usersvc.UserInput{FirstName: "Asha", LastName: "Rao", RoleID: 1}, false)
	if err != nil || input.Email != "" {
		t.Fatalf("update input = %+v, err = %v", input, err)
	}
}
