// Package cache implements a manager connector of cache and database
package cache

import (
	"context"
	"github.com/rs/zerolog"
	"l0/internal/interfaces"
	"l0/internal/models"
	"os"
	"sync"
)

// A Manager is a thread-safe connector of cache and database to work with stored data
type Manager struct {
	cache  interfaces.Cache[string, *models.Order]
	repo   interfaces.Repository
	logger *zerolog.Logger
	mu     sync.Mutex
}

// NewManager creates a new manager with specified cache, repo and logger
func NewManager(
	cache interfaces.Cache[string, *models.Order], repo interfaces.Repository, logger *zerolog.Logger,
) *Manager {
	if logger == nil {
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		return &Manager{cache: cache, repo: repo, logger: &logger}
	}
	return &Manager{cache: cache, repo: repo, logger: logger}
}

// WarmCache add at most cache.capacity elements in cache
func (c *Manager) WarmCache(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	orders, err := c.repo.GetNOrders(context.Background(), c.cache.Capacity())
	if err != nil {
		c.logger.Error().Stack().Err(err).Msg("")
	}
	for _, order := range orders {
		c.cache.Set(order.OrderUID, &order)
	}

	return nil
}

// Set add an order to the cache and database
func (c *Manager) Set(ctx context.Context, order *models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.repo.SaveOrder(ctx, order)
	if err != nil {
		c.logger.Error().Stack().Err(err).Msg("")
	}
	c.cache.Set(order.OrderUID, order)
}

// Get returns order from cache, if it's not there - from database
func (c *Manager) Get(ctx context.Context, orderUID string) (*models.Order, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, ok := c.cache.Get(orderUID)
	if ok {
		return node, nil
	}

	node, err := c.repo.GetOrder(ctx, orderUID)
	if err != nil {
		c.logger.Error().Stack().Err(err).Msg("")
		return nil, err
	}
	if node != nil {
		c.cache.Set(orderUID, node)
	}
	return node, nil
}

// DeleteCache removes element from the cache
func (c *Manager) DeleteCache(orderUID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.cache.Delete(orderUID)
	if err != nil {
		c.logger.Error().Stack().Err(err).Msg("")
	}
	return
}

// ContainsCache checks if element is present in cache
func (c *Manager) ContainsCache(orderUID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.cache.Contains(orderUID)
}

// FlushCache cleans all cache
func (c *Manager) FlushCache() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.Flush()
	return
}

// SizeCache returns number of elements in cache
func (c *Manager) SizeCache() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.cache.Size()
}

// EmptyCache return whether the cache is empty
func (c *Manager) EmptyCache() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.cache.Empty()
}