package screening

import (
	"context"

	"dpdp-backend/internal/delivery"
	"dpdp-backend/internal/delivery/repositories/provider"
)

type Engine struct {
	configs   *provider.ConfigCache
	screening *ScreeningService
}

func NewEngine(configs *provider.ConfigCache, screening *ScreeningService) *Engine {
	return &Engine{configs: configs, screening: screening}
}

func (e *Engine) Process(ctx context.Context, msg delivery.EmailMessage) (delivery.ProcessResult, error) {
	cfg, err := e.configs.Resolve(ctx, msg.CustomerID, msg.ConfigID)
	if err != nil {
		return delivery.ProcessResult{}, err
	}

	if e.screening == nil {
		return delivery.ProcessResult{Message: msg, Config: cfg}, nil
	}

	outcome, err := e.screening.Enforce(ctx, msg)
	if err != nil {
		return delivery.ProcessResult{Withheld: Withhold(outcome.Withheld)}, err
	}

	msg.Recipients = outcome.Delivered

	return delivery.ProcessResult{
		Message:  msg,
		Config:   cfg,
		Withheld: Withhold(outcome.Withheld),
	}, nil
}
