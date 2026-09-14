package branding

type UpdateBrandingRequest struct {
	Theme    *string `json:"theme" binding:"omitempty,max=10"`
	Language *string `json:"language" binding:"omitempty,max=20"`
	Timezone *string `json:"timezone" binding:"omitempty,max=64"`
}
