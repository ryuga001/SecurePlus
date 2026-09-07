package services_test

import (
	"errors"
	"testing"

	usersvc "dpdp-backend/internal/admin/services/emailuser"
	groupsvc "dpdp-backend/internal/admin/services/group"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

func TestNormalizeEmailUserInputNormalisesEmail(t *testing.T) {
	input, err := usersvc.NormalizeEmailUserInput(usersvc.EmailUserInput{
		Email:     "  Alice.Smith@Example.COM ",
		FirstName: " Alice  ",
		LastName:  "  Smith ",
	})
	if err != nil {
		t.Fatalf("usersvc.NormalizeEmailUserInput returned %v", err)
	}

	if input.Email != "alice.smith@example.com" {
		t.Fatalf("email = %q", input.Email)
	}
	if input.FirstName != "Alice" || input.LastName != "Smith" {
		t.Fatalf("names = %q %q", input.FirstName, input.LastName)
	}
}

func TestNormalizeEmailUserInputRejectsBadEmail(t *testing.T) {
	for _, email := range []string{"", "   ", "alice", "alice example.com"} {
		_, err := usersvc.NormalizeEmailUserInput(usersvc.EmailUserInput{Email: email, FirstName: "A", LastName: "B"})
		if !errors.Is(err, utils.ErrInvalidEmail) {
			t.Errorf("email %q accepted, error = %v", email, err)
		}
	}
}

func TestNormalizeEmailUserInputRequiresNames(t *testing.T) {
	_, err := usersvc.NormalizeEmailUserInput(usersvc.EmailUserInput{Email: "alice@example.com", FirstName: " ", LastName: "Smith"})
	if !errors.Is(err, utils.ErrNameNeeded) {
		t.Fatalf("error = %v, want ErrNameNeeded", err)
	}
}

func TestNormalizeGroupInputDefaultsToUserType(t *testing.T) {
	input, err := groupsvc.NormalizeGroupInput(groupsvc.GroupInput{Name: "  Finance   Team "})
	if err != nil {
		t.Fatalf("groupsvc.NormalizeGroupInput returned %v", err)
	}

	if input.Type != db.GroupTypeUser {
		t.Fatalf("type = %q", input.Type)
	}
	if input.Name != "Finance Team" {
		t.Fatalf("name = %q", input.Name)
	}
}

func TestNormalizeGroupInputAcceptsLowercaseUser(t *testing.T) {
	input, err := groupsvc.NormalizeGroupInput(groupsvc.GroupInput{Name: "Finance", Type: " user "})
	if err != nil {
		t.Fatalf("groupsvc.NormalizeGroupInput returned %v", err)
	}

	if input.Type != db.GroupTypeUser {
		t.Fatalf("type = %q", input.Type)
	}
}

func TestNormalizeGroupInputRejectsOtherTypes(t *testing.T) {
	_, err := groupsvc.NormalizeGroupInput(groupsvc.GroupInput{Name: "Finance", Type: "DOMAIN"})
	if !errors.Is(err, utils.ErrInvalidGroupType) {
		t.Fatalf("error = %v, want ErrInvalidGroupType", err)
	}
}

func TestNormalizeGroupInputRequiresName(t *testing.T) {
	_, err := groupsvc.NormalizeGroupInput(groupsvc.GroupInput{Name: "   "})
	if !errors.Is(err, utils.ErrGroupNameNeeded) {
		t.Fatalf("error = %v, want ErrGroupNameNeeded", err)
	}
}
