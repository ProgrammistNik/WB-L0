package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/sony/gobreaker"

	"l0/internal/cache"
	"l0/internal/models"
)

// An OrderService implements the business logic for order processing
type OrderService struct {
	cacheManager   *cache.Manager
	logger         *zerolog.Logger
	circuitBreaker *gobreaker.CircuitBreaker
}

// NewOrderService creates a new order service with the provided cache manager and logger
func NewOrderService(cacheManager *cache.Manager, logger *zerolog.Logger) *OrderService {
	cb := gobreaker.NewCircuitBreaker(
		gobreaker.Settings{
			Name:        "order-service",
			MaxRequests: 3,
			Interval:    60 * time.Second,
			Timeout:     60 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 5
			},
		},
	)

	return &OrderService{
		cacheManager:   cacheManager,
		logger:         logger,
		circuitBreaker: cb,
	}
}

// ProcessOrder handles incoming orders from Kafka, validates them, and saves to database/cache
func (s *OrderService) ProcessOrder(ctx context.Context, order *models.Order) error {
	start := time.Now()

	if order == nil {
		err := errors.New("order cannot be nil")
		s.logger.Error().Err(err).Msg("ProcessOrder: received nil order")
		return err
	}

	processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := s.validateOrder(order); err != nil {
		s.logger.Error().
			Err(err).
			Str("order_uid", order.OrderUID).
			Dur("duration", time.Since(start)).
			Msg("ProcessOrder: order validation failed")
		return fmt.Errorf("order validation failed: %w", err)
	}

	if order.DateCreated.IsZero() {
		order.DateCreated = time.Now()
	}

	_, err := s.circuitBreaker.Execute(
		func() (interface{}, error) {
			s.cacheManager.Set(processCtx, order)
			return nil, nil
		},
	)

	duration := time.Since(start)

	if err != nil {
		s.logger.Error().
			Err(err).
			Str("order_uid", order.OrderUID).
			Dur("duration", duration).
			Msg("ProcessOrder: order processing failed")
		return fmt.Errorf("failed to process order: %w", err)
	}

	return nil
}

// GetOrder retrieves an order by UID, checking cache first, then database
func (s *OrderService) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	start := time.Now()

	if strings.TrimSpace(orderUID) == "" {
		err := errors.New("order UID cannot be empty")
		s.logger.Error().Err(err).Msg("GetOrder: empty order UID provided")
		return nil, err
	}

	retrieveCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	result, err := s.circuitBreaker.Execute(
		func() (interface{}, error) {
			return s.cacheManager.Get(retrieveCtx, orderUID)
		},
	)

	duration := time.Since(start)

	if err != nil {
		s.logger.Error().
			Err(err).
			Str("order_uid", orderUID).
			Dur("duration", duration).
			Msg("GetOrder: failed to retrieve order")
		return nil, fmt.Errorf("failed to retrieve order: %w", err)
	}

	order := result.(*models.Order)
	if order == nil {
		return nil, nil
	}

	return order, nil
}

// WarmCache loads recent orders from database into cache on startup
func (s *OrderService) WarmCache(ctx context.Context) error {
	start := time.Now()

	warmCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	_, err := s.circuitBreaker.Execute(
		func() (interface{}, error) {
			return nil, s.cacheManager.WarmCache(warmCtx)
		},
	)

	duration := time.Since(start)

	if err != nil {
		s.logger.Error().
			Err(err).
			Dur("duration", duration).
			Msg("WarmCache: failed to warm cache")
		return fmt.Errorf("failed to warm cache: %w", err)
	}

	return nil
}

// validateOrder performs comprehensive validation of order data
func (s *OrderService) validateOrder(order *models.Order) error {
	if err := order.Validate(); err != nil {
		s.logger.Error().
			Err(err).
			Str("order_uid", order.OrderUID).
			Msg("Order validation failed")
		return err
	}

	return nil
}