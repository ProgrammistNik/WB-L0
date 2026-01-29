package cache

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"l0/internal/cache/lru_cache"
	"l0/internal/models"
	"maps"
	"os"
	"slices"
	"sync"
	"testing"
)

// A mockRepository is a not thread-safe mock implementation of Cache for testing
type mockRepository struct {
	orders map[string]models.Order
	err    error // to create artificial errors
}

func (m *mockRepository) SaveOrder(ctx context.Context, order *models.Order) error {
	if m.err != nil {
		return m.err
	}

	if m.orders == nil {
		m.orders = make(map[string]models.Order)
	}

	m.orders[order.OrderUID] = *order
	return nil
}

func (m *mockRepository) GetOrder(ctx context.Context, orderUid string) (*models.Order, error) {
	if m.err != nil {
		return nil, m.err
	}

	order, ok := m.orders[orderUid]
	if !ok {
		return nil, nil
	}

	return &order, nil
}

func (m *mockRepository) GetNOrders(ctx context.Context, n int) ([]models.Order, error) {
	if m.err != nil {
		return nil, m.err
	}

	if n < 0 {
		return nil, errors.New("expected positive n")
	}

	var orders []models.Order

	i := 0
	for v := range maps.Values(m.orders) {
		orders = append(orders, v)
		i += 1
		if i == n {
			break
		}
	}
	return orders, nil
}

func (m *mockRepository) GetAllOrders(ctx context.Context) ([]models.Order, error) {
	if m.err != nil {
		return nil, m.err
	}

	return slices.Collect(maps.Values(m.orders)), nil

}

// A mockCache is a not thread-safe mock implementation of Cache without eviction for testing
type mockCache[K comparable, V any] struct {
	cache    map[K]V
	capacity int
}

func newMockCache[K comparable, V any](capacity int) *mockCache[K, V] {
	return &mockCache[K, V]{make(map[K]V), capacity}
}

func (m *mockCache[K, V]) Set(key K, value V) {
	m.cache[key] = value
}

func (m *mockCache[K, V]) Get(key K) (V, bool) {
	v, ok := m.cache[key]

	return v, ok
}

func (m *mockCache[K, V]) Delete(key K) error {
	delete(m.cache, key)
	return nil
}

func (m *mockCache[K, V]) Contains(key K) bool {
	_, ok := m.cache[key]
	return ok
}

func (m *mockCache[K, V]) Flush() {
	clear(m.cache)
}

func (m *mockCache[K, V]) Size() int {
	return len(m.cache)
}

func (m *mockCache[K, V]) Capacity() int {
	return m.capacity
}

func (m *mockCache[K, V]) Empty() bool {
	return len(m.cache) == 0
}

func TestNewManager(t *testing.T) {
	cache := newMockCache[string, *models.Order](10)
	repo := mockRepository{}
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	m := NewManager(cache, &repo, &logger)

	if m.cache != cache {
		t.Errorf("error: expected proper cache")
	}
	if m.repo != &repo {
		t.Errorf("error: expected proper repository")
	}
	if m.logger != &logger {
		t.Errorf("error: expected proper logger")
	}
}

func TestNewManager_NilLogger(t *testing.T) {
	cache := newMockCache[string, *models.Order](10)
	repo := mockRepository{}
	m := NewManager(cache, &repo, nil)

	if m.cache != cache {
		t.Errorf("error: expected proper cache")
	}
	if m.repo != &repo {
		t.Errorf("error: expected proper repository")
	}
	if m.logger == nil {
		t.Errorf("error: expected default logger to be used")
	}
}

func TestManager_WarmCache(t *testing.T) {
	cache := newMockCache[string, *models.Order](10)
	repo := mockRepository{
		orders: map[string]*models.Order{
			"order1": {OrderUID: "order1", Entry: "entry1"},
			"order2": {OrderUID: "order2", Entry: "entry2"},
			"order3": {OrderUID: "order3", Entry: "entry3"},
		},
	}
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	m := NewManager(cache, &repo, &logger)
	err := m.WarmCache(context.Background())
	if err != nil {
		t.Fatalf("error: failed to warm cache, %v", err)
	}

	if m.SizeCache() != 3 {
		t.Errorf("error: expected cache size 3, got %d", m.SizeCache())
	}

	order, ok := m.cache.Get("order1")
	if !ok || order.OrderUID != "order1" {
		t.Errorf("error: failed to load order")
	}

}

func TestManager_SetCache(t *testing.T) {
	cache := newMockCache[string, *models.Order](10)
	repo := mockRepository{}
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	m := NewManager(cache, &repo, &logger)

	m.Set(context.Background(), &models.Order{OrderUID: "order1", Entry: "entry1"})
	m.Set(context.Background(), &models.Order{OrderUID: "order2", Entry: "entry2"})
	m.Set(context.Background(), &models.Order{OrderUID: "order3", Entry: "entry3"})

	if m.SizeCache() != 3 {
		t.Errorf("error: expected cache size 3, got %d", m.SizeCache())
	}

	order, ok := m.cache.Get("order2")
	if !ok || order.OrderUID != "order2" {
		t.Errorf("error: failed to load order")
	}
}

func TestManager_GetCache(t *testing.T) {
	cache := newMockCache[string, *models.Order](10)
	repo := mockRepository{
		orders: map[string]*models.Order{
			"order1": {OrderUID: "order1", Entry: "entry1"},
			"order2": {OrderUID: "order2", Entry: "entry2"},
			"order3": {OrderUID: "order3", Entry: "entry3"},
		},
	}
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	m := NewManager(cache, &repo, &logger)

	m.Set(context.Background(), &models.Order{OrderUID: "order1", Entry: "entry1"})
	result, err := m.Get(context.Background(), "order1")
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if result.Entry != "entry1" {
		t.Errorf("error: expexted entry1, got %v", result.Entry)
	}

	result, err = m.Get(context.Background(), "order2")
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if result.Entry != "entry2" {
		t.Errorf("error: expexted entry2, got %v", result.Entry)
	}
	c, ok := m.cache.Get("order2")
	if !ok || c.Entry != "entry2" {
		t.Errorf("error: expected order caching after database retrieval")
	}

	result, err = m.Get(context.Background(), "order66")
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if result != nil {
		t.Errorf("error: expected nil order doesn't exist")
	}
}

func TestManager_DeleteCache(t *testing.T) {
	cache := newMockCache[string, *models.Order](10)
	repo := mockRepository{}
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	m := NewManager(cache, &repo, &logger)

	m.Set(context.Background(), &models.Order{OrderUID: "order1", Entry: "entry1"})
	m.DeleteCache("order1")

	_, ok := m.cache.Get("order1")
	if ok {
		t.Errorf("error: expected order to be deleted from cache")
	}
}

func TestManager_ContainsCache(t *testing.T) {
	cache := newMockCache[string, *models.Order](10)
	repo := mockRepository{}
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	m := NewManager(cache, &repo, &logger)

	m.Set(context.Background(), &models.Order{OrderUID: "order1", Entry: "entry1"})
	ok := m.ContainsCache("order1")
	if !ok {
		t.Errorf("error: expected order to be in cache")
	}
	m.DeleteCache("order1")
	ok = m.ContainsCache("order1")
	if ok {
		t.Errorf("error: expected order to be deleted from cache")
	}
}

func TestManager_FlushCache(t *testing.T) {
	cache := newMockCache[string, *models.Order](10)
	repo := mockRepository{}
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	m := NewManager(cache, &repo, &logger)

	m.Set(context.Background(), &models.Order{OrderUID: "order1", Entry: "entry1"})
	m.Set(context.Background(), &models.Order{OrderUID: "order2", Entry: "entry2"})
	m.Set(context.Background(), &models.Order{OrderUID: "order3", Entry: "entry3"})

	m.FlushCache()
	if m.cache.Size() != 0 {
		t.Errorf("expected cache to be empty after Flush, got %d", m.SizeCache())
	}
}

func TestManager_SizeCache(t *testing.T) {
	cache := newMockCache[string, *models.Order](10)
	repo := mockRepository{}
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	m := NewManager(cache, &repo, &logger)

	m.Set(context.Background(), &models.Order{OrderUID: "order1", Entry: "entry1"})
	m.Set(context.Background(), &models.Order{OrderUID: "order2", Entry: "entry2"})
	m.Set(context.Background(), &models.Order{OrderUID: "order3", Entry: "entry3"})

	if m.SizeCache() != 3 {
		t.Errorf("error: expected size 3, got: %d", m.SizeCache())
	}

	m.DeleteCache("order2")
	if m.SizeCache() != 2 {
		t.Errorf("error: expected size 2, got: %d", m.SizeCache())
	}

	m.FlushCache()
	if m.SizeCache() != 0 {
		t.Errorf("error: expected cache to be empty after Flush, got %d", m.SizeCache())
	}
}

func TestManager_EmptyCache(t *testing.T) {
	cache := newMockCache[string, *models.Order](10)
	repo := mockRepository{}
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	m := NewManager(cache, &repo, &logger)

	m.Set(context.Background(), &models.Order{OrderUID: "order1", Entry: "entry1"})
	m.Set(context.Background(), &models.Order{OrderUID: "order2", Entry: "entry2"})
	m.Set(context.Background(), &models.Order{OrderUID: "order3", Entry: "entry3"})

	if m.EmptyCache() {
		t.Errorf("error: expected cache not to be empty")
	}
	m.DeleteCache("order3")
	if m.EmptyCache() {
		t.Errorf("error: expected cache not to be empty")
	}
	m.FlushCache()
	if !m.EmptyCache() {
		t.Errorf("error: expected cache to be empty")
	}
}

func TestManager_DBError(t *testing.T) {
	cache := newMockCache[string, *models.Order](10)
	repo := mockRepository{err: errors.New("db mock connection error")}
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	m := NewManager(cache, &repo, &logger)

	m.Set(context.Background(), &models.Order{OrderUID: "order1", Entry: "entry1"})
	m.Set(context.Background(), &models.Order{OrderUID: "order2", Entry: "entry2"})
	m.DeleteCache("order1")

	result, err := m.Get(context.Background(), "order3")
	if err == nil {
		t.Errorf("error: expected error")
	}
	if result != nil {
		t.Errorf("error: expected nil as database errored")
	}
}

func TestManager_Concurrency(t *testing.T) {
	cache, err := lru_cache.NewLRUCache[string, *models.Order](1000000)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	repo := mockRepository{}
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	m := NewManager(cache, &repo, &logger)

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for i := 0; i < 100000; i++ {
			m.Set(
				context.Background(),
				&models.Order{OrderUID: fmt.Sprintf("order%d", i), Entry: fmt.Sprintf("entry%d", i)},
			)
		}
		wg.Done()
	}()

	go func() {
		for i := 0; i < 100000; i++ {
			m.Get(context.Background(), fmt.Sprintf("order%d", i))
		}
		wg.Done()
	}()

	wg.Wait()
	if m.SizeCache() != 100000 {
		t.Errorf("%d", m.SizeCache())
		t.Fail()
	}
}