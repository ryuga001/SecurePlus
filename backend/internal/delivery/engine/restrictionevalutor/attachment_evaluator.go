package restrictionevalutor

import (
	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	"dpdp-backend/internal/delivery/utils"
)

type AttachmentEvaluator struct{}

func NewAttachmentEvaluator() *AttachmentEvaluator {
	return &AttachmentEvaluator{}
}

func (e *AttachmentEvaluator) Evaluate(attachments []dto.Attachment, restrictions dto.EffectiveRestrictions) []dto.RestrictionViolation {
	violations := make([]dto.RestrictionViolation, 0)
	set := restrictions.Attachment

	for _, attachment := range attachments {
		if ref, blocked := set.Blocked[attachment.Extension]; blocked && attachment.Extension != "" {
			violations = append(violations, violation(attachment, utils.RestrictionBlock, ref))
			continue
		}

		if set.Allowed == nil {
			continue
		}

		if _, allowed := set.Allowed[attachment.Extension]; allowed && attachment.Extension != "" {
			continue
		}

		violations = append(violations, violation(attachment, utils.RestrictionAllow, anyRef(set.Allowed)))
	}

	return violations
}

func violation(attachment dto.Attachment, mode string, ref dto.PolicyRef) dto.RestrictionViolation {
	return dto.RestrictionViolation{
		Kind:        utils.KindAttachment,
		Mode:        mode,
		Value:       attachment.Extension,
		Filename:    attachment.Filename,
		ContentType: attachment.ContentType,
		PolicyID:    ref.PolicyID,
		PolicyName:  ref.PolicyName,
	}
}
