package interfaces

import (
	"context"
	"l0/internal/models"
)

type OrderService interface {
	ProcessOrder(ctx context.Context, order *models.Order) error
	GetOrder(ctx context.Context, OrderUID string) (*models.Order, error)
	WarmCache(ctx context.Context) error
}