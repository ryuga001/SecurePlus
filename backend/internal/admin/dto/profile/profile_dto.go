package profile

import service "dpdp-backend/internal/admin/services/profile"

type UpdateProfileRequest struct {
	OrgName    *string `json:"org_name" binding:"omitempty,max=200"`
	FirstName  *string `json:"first_name" binding:"omitempty,max=200"`
	LastName   *string `json:"last_name" binding:"omitempty,max=200"`
	AdminEmail *string `json:"admin_email"`
}

type ProfileResponse struct {
	OrgName    string `json:"org_name"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	AdminEmail string `json:"admin_email"`
}

func ToProfileResponse(profile service.Profile) ProfileResponse {
	return ProfileResponse{
		OrgName:    profile.OrgName,
		FirstName:  profile.FirstName,
		LastName:   profile.LastName,
		AdminEmail: profile.AdminEmail,
	}
}
