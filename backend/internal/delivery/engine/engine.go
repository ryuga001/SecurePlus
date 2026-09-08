package engine

import (
	"context"

	"dpdp-backend/internal/delivery"
)

type Engine struct {
	configs *ConfigCache
}

func NewEngine(configs *ConfigCache) *Engine {
	return &Engine{configs: configs}
}

func (e *Engine) Process(ctx context.Context, msg delivery.EmailMessage) (delivery.EmailMessage, delivery.TenantConfig, error) {
	cfg, err := e.configs.Resolve(ctx, msg.CustomerID, msg.ConfigID)
	if err != nil {
		return delivery.EmailMessage{}, delivery.TenantConfig{}, err
	}

	return msg, cfg, nil
}
