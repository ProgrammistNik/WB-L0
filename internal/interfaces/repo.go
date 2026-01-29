package interfaces

import (
	"context"
	"l0/internal/models"
)

type Repository interface {
	SaveOrder(ctx context.Context, order *models.Order) error
	GetOrder(ctx context.Context, orderUid string) (*models.Order, error)
	GetNOrders(ctx context.Context, n int) ([]models.Order, error)
	GetAllOrders(ctx context.Context) ([]models.Order, error)
}