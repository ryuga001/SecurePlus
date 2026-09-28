package utils_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"dpdp-backend/internal/admin/utils"
)

var errUnrecognised = errors.New("something the mapping has never seen")

func respond(t *testing.T, err error) (int, utils.ErrorResponse) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	utils.Respond(context, err)

	var body utils.ErrorResponse
	if unmarshalErr := json.Unmarshal(recorder.Body.Bytes(), &body); unmarshalErr != nil {
		t.Fatalf("response is not json: %v (%s)", unmarshalErr, recorder.Body.String())
	}

	return recorder.Code, body
}

func TestEveryAdminSentinelIsMapped(t *testing.T) {
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrAdminEmailImmutable", utils.ErrAdminEmailImmutable},
		{"ErrAlertNameNeeded", utils.ErrAlertNameNeeded},
		{"ErrAlertNameTaken", utils.ErrAlertNameTaken},
		{"ErrAlertNotFound", utils.ErrAlertNotFound},
		{"ErrBrandingNotFound", utils.ErrBrandingNotFound},
		{"ErrConfigFieldNeeded", utils.ErrConfigFieldNeeded},
		{"ErrConfigurationNotFound", utils.ErrConfigurationNotFound},
		{"ErrConfigurationTypeImmutable", utils.ErrConfigurationTypeImmutable},
		{"ErrConnectionFailed", utils.ErrConnectionFailed},
		{"ErrCredentialUnavailable", utils.ErrCredentialUnavailable},
		{"ErrDashboardUserNotFound", utils.ErrDashboardUserNotFound},
		{"ErrDiscoveryConfigInUse", utils.ErrDiscoveryConfigInUse},
		{"ErrDiscoveryConfigNameTaken", utils.ErrDiscoveryConfigNameTaken},
		{"ErrDiscoveryConfigNotFound", utils.ErrDiscoveryConfigNotFound},
		{"ErrDiscoveryNameNeeded", utils.ErrDiscoveryNameNeeded},
		{"ErrDiscoveryPolicyInactive", utils.ErrDiscoveryPolicyInactive},
		{"ErrDiscoveryPolicyNameTaken", utils.ErrDiscoveryPolicyNameTaken},
		{"ErrDiscoveryPolicyNotFound", utils.ErrDiscoveryPolicyNotFound},
		{"ErrDiscoveryScanActive", utils.ErrDiscoveryScanActive},
		{"ErrDiscoveryScanNotFound", utils.ErrDiscoveryScanNotFound},
		{"ErrDiscoveryTargetsNeeded", utils.ErrDiscoveryTargetsNeeded},
		{"ErrDomainTaken", utils.ErrDomainTaken},
		{"ErrEmailTaken", utils.ErrEmailTaken},
		{"ErrEmailUserNotFound", utils.ErrEmailUserNotFound},
		{"ErrGroupNameNeeded", utils.ErrGroupNameNeeded},
		{"ErrGroupNameTaken", utils.ErrGroupNameTaken},
		{"ErrGroupNotFound", utils.ErrGroupNotFound},
		{"ErrGroupsNeeded", utils.ErrGroupsNeeded},
		{"ErrIncompatibleSource", utils.ErrIncompatibleSource},
		{"ErrInvalidAction", utils.ErrInvalidAction},
		{"ErrInvalidConfigurationType", utils.ErrInvalidConfigurationType},
		{"ErrInvalidCursor", utils.ErrInvalidCursor},
		{"ErrInvalidDiscoveryTarget", utils.ErrInvalidDiscoveryTarget},
		{"ErrInvalidDomain", utils.ErrInvalidDomain},
		{"ErrInvalidEmail", utils.ErrInvalidEmail},
		{"ErrInvalidGroupType", utils.ErrInvalidGroupType},
		{"ErrInvalidLanguage", utils.ErrInvalidLanguage},
		{"ErrInvalidLogo", utils.ErrInvalidLogo},
		{"ErrInvalidNotificationType", utils.ErrInvalidNotificationType},
		{"ErrInvalidProvider", utils.ErrInvalidProvider},
		{"ErrInvalidRegex", utils.ErrInvalidRegex},
		{"ErrInvalidRestrictionDomain", utils.ErrInvalidRestrictionDomain},
		{"ErrInvalidRestrictionFileType", utils.ErrInvalidRestrictionFileType},
		{"ErrInvalidRestrictionMode", utils.ErrInvalidRestrictionMode},
		{"ErrInvalidRuleType", utils.ErrInvalidRuleType},
		{"ErrInvalidScheduleType", utils.ErrInvalidScheduleType},
		{"ErrInvalidServiceAccountKey", utils.ErrInvalidServiceAccountKey},
		{"ErrInvalidSourceType", utils.ErrInvalidSourceType},
		{"ErrInvalidTarget", utils.ErrInvalidTarget},
		{"ErrInvalidTheme", utils.ErrInvalidTheme},
		{"ErrInvalidTimezone", utils.ErrInvalidTimezone},
		{"ErrLastAdministrator", utils.ErrLastAdministrator},
		{"ErrMappingExists", utils.ErrMappingExists},
		{"ErrMappingNotFound", utils.ErrMappingNotFound},
		{"ErrNameNeeded", utils.ErrNameNeeded},
		{"ErrNameTooLong", utils.ErrNameTooLong},
		{"ErrOrgEditForbidden", utils.ErrOrgEditForbidden},
		{"ErrOrgNameNeeded", utils.ErrOrgNameNeeded},
		{"ErrOrgNameTaken", utils.ErrOrgNameTaken},
		{"ErrPoliciesNeeded", utils.ErrPoliciesNeeded},
		{"ErrPolicyNameNeeded", utils.ErrPolicyNameNeeded},
		{"ErrPolicyNameTaken", utils.ErrPolicyNameTaken},
		{"ErrPolicyNotFound", utils.ErrPolicyNotFound},
		{"ErrPrivilegesNeeded", utils.ErrPrivilegesNeeded},
		{"ErrProfileNotFound", utils.ErrProfileNotFound},
		{"ErrProviderUnavailable", utils.ErrProviderUnavailable},
		{"ErrRestrictionValuesNeeded", utils.ErrRestrictionValuesNeeded},
		{"ErrRoleDescriptionTooLong", utils.ErrRoleDescriptionTooLong},
		{"ErrRoleInUse", utils.ErrRoleInUse},
		{"ErrRoleNameNeeded", utils.ErrRoleNameNeeded},
		{"ErrRoleNameTaken", utils.ErrRoleNameTaken},
		{"ErrRoleNotFound", utils.ErrRoleNotFound},
		{"ErrRuleNameNeeded", utils.ErrRuleNameNeeded},
		{"ErrRuleNameTaken", utils.ErrRuleNameTaken},
		{"ErrRuleNotFound", utils.ErrRuleNotFound},
		{"ErrRulesNeeded", utils.ErrRulesNeeded},
		{"ErrRuleValueNeeded", utils.ErrRuleValueNeeded},
		{"ErrSecretNeeded", utils.ErrSecretNeeded},
		{"ErrSelfModification", utils.ErrSelfModification},
		{"ErrSMSNotSupported", utils.ErrSMSNotSupported},
		{"ErrSourceNotScannable", utils.ErrSourceNotScannable},
		{"ErrSystemAlertImmutable", utils.ErrSystemAlertImmutable},
		{"ErrSystemRoleImmutable", utils.ErrSystemRoleImmutable},
		{"ErrTargetBrowseFailed", utils.ErrTargetBrowseFailed},
		{"ErrTargetsNeeded", utils.ErrTargetsNeeded},
		{"ErrTooManyItems", utils.ErrTooManyItems},
		{"ErrUnauthenticated", utils.ErrUnauthenticated},
		{"ErrUnavailable", utils.ErrUnavailable},
		{"ErrUnknownConfiguration", utils.ErrUnknownConfiguration},
		{"ErrUnknownEmailUser", utils.ErrUnknownEmailUser},
		{"ErrUnknownFileType", utils.ErrUnknownFileType},
		{"ErrUnknownGroup", utils.ErrUnknownGroup},
		{"ErrUnknownPolicy", utils.ErrUnknownPolicy},
		{"ErrUnknownPrivilege", utils.ErrUnknownPrivilege},
		{"ErrUnknownRole", utils.ErrUnknownRole},
		{"ErrUnknownRule", utils.ErrUnknownRule},
		{"ErrUserEmailImmutable", utils.ErrUserEmailImmutable},
	}

	for _, sentinel := range sentinels {
		t.Run(sentinel.name, func(t *testing.T) {
			status, body := respond(t, sentinel.err)

			if status == http.StatusInternalServerError && body.Error == utils.CodeInternal {
				t.Fatalf("%s falls through to the unmapped internal_error bucket; add a case for it in statusFor", sentinel.name)
			}
			if body.Error == "" {
				t.Fatalf("%s produced an empty error code", sentinel.name)
			}
		})
	}
}

func TestSentinelsMapToTheirDocumentedStatusAndCode(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"ErrUnauthenticated", utils.ErrUnauthenticated, http.StatusUnauthorized, utils.CodeUnauthenticated},
		{"ErrUnavailable", utils.ErrUnavailable, http.StatusServiceUnavailable, utils.CodeUnavailable},
		{"ErrPolicyNotFound", utils.ErrPolicyNotFound, http.StatusNotFound, utils.CodeNotFound},
		{"ErrDashboardUserNotFound", utils.ErrDashboardUserNotFound, http.StatusNotFound, utils.CodeNotFound},
		{"ErrDiscoveryScanNotFound", utils.ErrDiscoveryScanNotFound, http.StatusNotFound, utils.CodeDiscoveryScanNotFound},
		{"ErrPolicyNameTaken", utils.ErrPolicyNameTaken, http.StatusConflict, utils.CodePolicyNameTaken},
		{"ErrRoleNameTaken", utils.ErrRoleNameTaken, http.StatusConflict, utils.CodeRoleNameTaken},
		{"ErrSystemRoleImmutable", utils.ErrSystemRoleImmutable, http.StatusConflict, utils.CodeSystemRoleImmutable},
		{"ErrRoleInUse", utils.ErrRoleInUse, http.StatusConflict, utils.CodeRoleInUse},
		{"ErrSelfModification", utils.ErrSelfModification, http.StatusConflict, utils.CodeSelfModification},
		{"ErrLastAdministrator", utils.ErrLastAdministrator, http.StatusConflict, utils.CodeLastAdministrator},
		{"ErrInvalidRegex", utils.ErrInvalidRegex, http.StatusBadRequest, utils.CodeInvalidRegex},
		{"ErrUnknownRole", utils.ErrUnknownRole, http.StatusBadRequest, utils.CodeRoleNotFound},
		{"ErrAdminEmailImmutable", utils.ErrAdminEmailImmutable, http.StatusBadRequest, utils.CodeAdminEmailImmutable},
		{"ErrUserEmailImmutable", utils.ErrUserEmailImmutable, http.StatusBadRequest, utils.CodeAdminEmailImmutable},
		{"ErrOrgEditForbidden", utils.ErrOrgEditForbidden, http.StatusForbidden, utils.CodeForbidden},
		{"ErrProviderUnavailable", utils.ErrProviderUnavailable, http.StatusBadGateway, utils.CodeProviderUnavailable},
		{"ErrTooManyItems", utils.ErrTooManyItems, http.StatusBadRequest, utils.CodeValidation},
		{"ErrPrivilegesNeeded", utils.ErrPrivilegesNeeded, http.StatusBadRequest, utils.CodeValidation},
		{"ErrRoleDescriptionTooLong", utils.ErrRoleDescriptionTooLong, http.StatusBadRequest, utils.CodeValidation},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			status, body := respond(t, testCase.err)

			if status != testCase.status || body.Error != testCase.code {
				t.Fatalf("%s -> (%d, %q), want (%d, %q)", testCase.name, status, body.Error, testCase.status, testCase.code)
			}
		})
	}
}

func TestWrappedSentinelsStillMapCorrectly(t *testing.T) {
	status, body := respond(t, utils.Taken(utils.ErrPolicyNameTaken, "Finance Policy"))
	if status != http.StatusConflict || body.Error != utils.CodePolicyNameTaken {
		t.Fatalf("wrapped Taken() = (%d, %q)", status, body.Error)
	}
	if body.Message == "" {
		t.Fatal("wrapped error message is empty")
	}

	status, body = respond(t, utils.ConnectionFailed("dial tcp: timeout"))
	if status != http.StatusBadRequest || body.Error != utils.CodeConnectionFailed {
		t.Fatalf("wrapped ConnectionFailed() = (%d, %q)", status, body.Error)
	}
}

func TestUnrecognisedErrorFallsBackToInternal(t *testing.T) {
	status, body := respond(t, errUnrecognised)

	if status != http.StatusInternalServerError || body.Error != utils.CodeInternal {
		t.Fatalf("unrecognised error = (%d, %q), want (500, internal_error)", status, body.Error)
	}
}
