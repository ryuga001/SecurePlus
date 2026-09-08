package utils

import "errors"

const (
	MsgDeliveryAuditNotFound = "delivery audit not found"

	CodeNotFound = "not_found"
)

var ErrDeliveryAuditNotFound = errors.New(MsgDeliveryAuditNotFound)
