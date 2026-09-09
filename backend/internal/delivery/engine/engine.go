package engine

import (
	"context"

	"dpdp-backend/internal/delivery"
	dto "dpdp-backend/internal/delivery/engine/dto/evaluation"
	"dpdp-backend/internal/delivery/engine/evaluation"
)

type Engine struct {
	configs  *ConfigCache
	enforcer dto.Enforcer
}

func NewEngine(configs *ConfigCache, enforcer dto.Enforcer) *Engine {
	return &Engine{configs: configs, enforcer: enforcer}
}

func (e *Engine) Process(ctx context.Context, msg delivery.EmailMessage) (delivery.ProcessResult, error) {
	cfg, err := e.configs.Resolve(ctx, msg.CustomerID, msg.ConfigID)
	if err != nil {
		return delivery.ProcessResult{}, err
	}

	if e.enforcer == nil {
		return delivery.ProcessResult{Message: msg, Config: cfg}, nil
	}

	outcome, err := e.enforcer.Enforce(ctx, dto.MessageContext{
		CorrelationID: msg.CorrelationID,
		MessageID:     msg.MessageID,
		CustomerID:    msg.CustomerID,
		ConfigID:      msg.ConfigID,
		From:          msg.From,
		SenderDomain:  msg.SenderDomain,
		Recipients:    msg.Recipients,
		Raw:           msg.Raw,
	})
	if err != nil {
		return delivery.ProcessResult{Withheld: evaluation.Withhold(outcome.Withheld)}, err
	}

	msg.Recipients = outcome.Delivered

	return delivery.ProcessResult{
		Message:  msg,
		Config:   cfg,
		Withheld: evaluation.Withhold(outcome.Withheld),
	}, nil
}
