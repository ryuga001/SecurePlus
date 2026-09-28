package services_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	rolesvc "dpdp-backend/internal/admin/services/role"
	"dpdp-backend/internal/admin/utils"
)

func TestNormalizeRoleInputTrimsAndDedupes(t *testing.T) {
	input, err := rolesvc.NormalizeRoleInput(rolesvc.RoleInput{
		Name:        "  Support   Team ",
		Description: "  Handles tickets  ",
		Privileges:  []string{" admin.rule.view", "admin.policy.view", "admin.rule.view", "  "},
	})
	if err != nil {
		t.Fatalf("NormalizeRoleInput returned %v", err)
	}

	if input.Name != "Support Team" || input.Description != "Handles tickets" {
		t.Fatalf("normalized = %q / %q", input.Name, input.Description)
	}
	if !slices.Equal(input.Privileges, []string{"admin.policy.view", "admin.rule.view"}) {
		t.Fatalf("privileges = %v", input.Privileges)
	}
}

func TestNormalizeRoleInputValidatesNameLength(t *testing.T) {
	for _, name := range []string{"", " ", "A", strings.Repeat("x", 51)} {
		_, err := rolesvc.NormalizeRoleInput(rolesvc.RoleInput{Name: name, Privileges: []string{"admin.rule.view"}})
		if !errors.Is(err, utils.ErrRoleNameNeeded) {
			t.Errorf("name %q: error = %v, want ErrRoleNameNeeded", name, err)
		}
	}

	if _, err := rolesvc.NormalizeRoleInput(rolesvc.RoleInput{Name: strings.Repeat("役", 50), Privileges: []string{"admin.rule.view"}}); err != nil {
		t.Fatalf("50-rune name rejected: %v", err)
	}
}

func TestNormalizeRoleInputLimitsDescription(t *testing.T) {
	_, err := rolesvc.NormalizeRoleInput(rolesvc.RoleInput{
		Name:        "Support",
		Description: strings.Repeat("d", 256),
		Privileges:  []string{"admin.rule.view"},
	})
	if !errors.Is(err, utils.ErrRoleDescriptionTooLong) {
		t.Fatalf("error = %v, want ErrRoleDescriptionTooLong", err)
	}
}

func TestNormalizeRoleInputRequiresPrivileges(t *testing.T) {
	for _, privileges := range [][]string{nil, {}, {"  ", ""}} {
		_, err := rolesvc.NormalizeRoleInput(rolesvc.RoleInput{Name: "Support", Privileges: privileges})
		if !errors.Is(err, utils.ErrPrivilegesNeeded) {
			t.Errorf("privileges %v: error = %v, want ErrPrivilegesNeeded", privileges, err)
		}
	}
}

func TestNormalizeRoleInputCapsPrivilegeCount(t *testing.T) {
	privileges := make([]string, 0, 101)
	for index := range 101 {
		privileges = append(privileges, "admin.test."+strings.Repeat("p", index+1))
	}

	_, err := rolesvc.NormalizeRoleInput(rolesvc.RoleInput{Name: "Support", Privileges: privileges})
	if !errors.Is(err, utils.ErrTooManyItems) {
		t.Fatalf("error = %v, want ErrTooManyItems", err)
	}
}
