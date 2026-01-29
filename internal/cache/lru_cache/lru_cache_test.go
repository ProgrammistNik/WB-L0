package lru_cache

import (
	"math/rand/v2"
	"sync"
	"testing"
)

func TestLRUCache_Set(t *testing.T) {
	c, err := NewLRUCache[int, int](2)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	c.Set(2, 2)

	if c.Empty() {
		t.Errorf("error: cache is empty")
	}
}

func TestLRUCache_Get(t *testing.T) {
	c, err := NewLRUCache[int, int](2)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	c.Set(2, 2)

	val, ok := c.Get(2)
	if !ok {
		t.Errorf("error: 2 should be contained")
	}
	if val != 2 {
		t.Errorf("error: value is not correct to the key")
	}
}

func TestLRUCache_Delete(t *testing.T) {
	c, err := NewLRUCache[int, int](5)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	c.Set(1, 1)
	c.Set(2, 2)
	c.Set(3, 3)

	err = c.Delete(2)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	_, ok := c.Get(2)
	if ok {
		t.Fatalf("error: key 2 should have been deleted")
	}
}

func TestLRUCache_Empty(t *testing.T) {
	c, err := NewLRUCache[int, int](5)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !c.Empty() {
		t.Fatalf("error: should be empty")
	}

	c.Set(6, 6)
	if c.Empty() {
		t.Fatalf("error: should not be empty")
	}

	err = c.Delete(6)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if !c.Empty() {
		t.Errorf("error: should be empty")
	}
}

func TestLRUCache_Flush(t *testing.T) {
	c, err := NewLRUCache[int, int](3)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	c.Set(1, 1)
	c.Set(2, 2)
	c.Set(3, 3)

	if c.Empty() {
		t.Fatalf("error:  should not be empty")
	}
	c.Flush()
	if !c.Empty() {
		t.Errorf("error: should be empty")
	}
}

func TestLRUCache_Size(t *testing.T) {
	c, err := NewLRUCache[int, int](5)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	c.Set(1, 1)
	c.Set(2, 2)
	if c.Size() != 2 {
		t.Errorf("error: size should be 2")
	}
	c.Set(3, 3)
	c.Set(4, 4)
	c.Set(5, 5)
	if c.Size() != 5 {
		t.Errorf("error: size should be 5")
	}
	c.Set(6, 6)
	c.Set(7, 7)
	if c.Size() != 5 {
		t.Errorf("error: size should be 5")
	}
}

func TestLRUCache_Contains(t *testing.T) {
	c, err := NewLRUCache[int, int](1)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	c.Set(0, 0)
	if !c.Contains(0) {
		t.Errorf("error: 0 is contained")
	}

	c.Set(1, 1)
	if c.Contains(0) {
		t.Errorf("error: 0 should be evicted")
	}
}
func TestLRUCache_Concurrency(t *testing.T) {
	l, err := NewLRUCache[float64, float64](200000)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for i := 0; i < 100000; i++ {
			l.Set(float64(i), float64(i))
		}
		wg.Done()
	}()

	go func() {
		for i := 0; i < 100000; i++ {
			l.Get(float64(i))
		}
		wg.Done()
	}()

	wg.Wait()
	if l.Size() != 100000 {
		t.Errorf("%d", l.Size())
		t.Fail()
	}
}
func BenchmarkLRUCache_Rand(b *testing.B) {
	c, err := NewLRUCache[int, int](8192)
	if err != nil {
		b.Fatalf("error: %v", err)
	}

	trace := make([]int, b.N*2)
	for i := 0; i < b.N*2; i++ {
		trace[i] = rand.IntN(32768)
	}

	b.ResetTimer()

	var hit, miss int
	for i := 0; i < 2*b.N; i++ {
		if i%2 == 0 {
			c.Set(trace[i], trace[i])
		} else {
			if _, ok := c.Get(trace[i]); ok {
				hit++
			} else {
				miss++
			}
		}
	}
	b.Logf("hit: %d miss: %d", hit, miss)
}

func BenchmarkLRUCache_Freq(b *testing.B) {
	c, err := NewLRUCache[int, int](8192)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	trace := make([]int, b.N*2)
	for i := 0; i < b.N*2; i++ {
		if i%2 == 0 {
			trace[i] = rand.IntN(16384)
		} else {
			trace[i] = rand.IntN(32768)
		}
	}

	b.ResetTimer()

	for i := range trace[:b.N] {
		c.Set(trace[i], trace[i])
	}
	var hit, miss int
	for i := range trace[:b.N] {
		if _, ok := c.Get(trace[i]); ok {
			hit++
		} else {
			miss++
		}
	}
	b.Logf("hit: %d miss: %d", hit, miss)
}