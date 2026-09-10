package utils

import "errors"

const (
	MsgDeliveryAuditNotFound = "delivery audit not found"
	MsgEmailIncidentNotFound = "email incident not found"

	CodeNotFound = "not_found"
)

var (
	ErrDeliveryAuditNotFound = errors.New(MsgDeliveryAuditNotFound)
	ErrEmailIncidentNotFound = errors.New(MsgEmailIncidentNotFound)
)
