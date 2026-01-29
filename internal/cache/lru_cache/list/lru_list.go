// Package list implements a list data structure
package list

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// A LRUListError is a custom error type for list
type LRUListError struct {
	message string
}

func (e *LRUListError) Error() string {
	return fmt.Sprintf("lru list error: %v", e.message)
}

var ErrRoot = &LRUListError{"can't perform operations with root"}
var ErrNotInList = &LRUListError{"node should be an element of this list"}
var ErrIsListElem = &LRUListError{"node shouldn't be an element of any list"}

// A LRUListNode is a thread-safe node for an LRUList
type LRUListNode[K comparable, V any] struct {
	next, prev atomic.Pointer[LRUListNode[K, V]]
	list       atomic.Pointer[LRUList[K, V]]
	Key        K
	Value      V
}

// Next returns a next node in the current doubly linked list or nil if there is none
func (node *LRUListNode[K, V]) Next() *LRUListNode[K, V] {
	if n := node.next.Load(); node.list.Load() != nil && n != &node.list.Load().root {
		return n
	}
	return nil
}

// Prev returns a previous node in the current doubly linked list or nil if there is none
func (node *LRUListNode[K, V]) Prev() *LRUListNode[K, V] {
	if p := node.prev.Load(); node.list.Load() != nil && p != &node.list.Load().root {
		return p
	}
	return nil
}

// NewLRUListNode returns a new node for LRUList
func NewLRUListNode[K comparable, V any](key K, value V, list *LRUList[K, V], next, prev *LRUListNode[K, V]) *LRUListNode[K, V] {
	node := &LRUListNode[K, V]{}
	node.next.Store(next)
	node.prev.Store(prev)
	node.list.Store(list)
	node.Key = key
	node.Value = value

	return node
}

// A LRUList is a thread-safe implementation of List interface for LRUCache
type LRUList[K comparable, V any] struct {
	root LRUListNode[K, V] // sentinel
	len  int
	mu   sync.RWMutex
}

// NewLRUList creates an empty LRUList. It should be created only using this command
func NewLRUList[K comparable, V any]() *LRUList[K, V] {
	l := LRUList[K, V]{}
	l.root.next.Store(&l.root)
	l.root.prev.Store(&l.root)
	l.root.list.Store(&l)
	return &l
}

// Front returns pointer to the element that is currently in the front of the list
func (l *LRUList[K, V]) Front() *LRUListNode[K, V] {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.root.Prev()
}

// Back returns pointer to the element that is currently in the back of the list
func (l *LRUList[K, V]) Back() *LRUListNode[K, V] {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.root.Next()
}

// MoveToFront moves the element pointed by at to the front of the list
func (l *LRUList[K, V]) MoveToFront(at *LRUListNode[K, V]) error {
	node, err := l.Remove(at)
	if err != nil {
		return err
	}
	_, err = l.pushFront(node)
	if err != nil {
		return err
	}
	return nil
}

// MoveToBack moves the element pointed by at to the back of the list
func (l *LRUList[K, V]) MoveToBack(at *LRUListNode[K, V]) error {
	node, err := l.Remove(at)
	if err != nil {
		return err
	}
	_, err = l.pushBack(node)
	if err != nil {
		return err
	}
	return nil

}

// Size returns the amount of elements that it currently holds
func (l *LRUList[K, V]) Size() int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.len
}

// Empty return if the list has no elements
func (l *LRUList[K, V]) Empty() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.len == 0
}

// insert adds an existing LRUListNode and inserts it after the element pointed by at. Added node shouldn't belong to any list
func (l *LRUList[K, V]) insert(node *LRUListNode[K, V], at *LRUListNode[K, V]) (*LRUListNode[K, V], error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node.list.Load() != nil || node.next.Load() != nil || node.prev.Load() != nil {
		return nil, ErrIsListElem
	}

	if at.list.Load() != l {
		return nil, ErrNotInList
	}
	node.list.Store(l)

	next := at.next.Load()
	at.next.Store(node)
	node.prev.Store(at)
	node.next.Store(next)
	next.prev.Store(node)

	l.len += 1

	return node, nil
}

// Insert creates a LRUListNode with key K and value V and inserts it after the element pointed by at
func (l *LRUList[K, V]) Insert(key K, value V, at *LRUListNode[K, V]) (*LRUListNode[K, V], error) {
	node := NewLRUListNode(key, value, nil, nil, nil)
	return l.insert(node, at)
}

// pushFront is an internal method to add existing node in the front. Added node shouldn't belong to any list
func (l *LRUList[K, V]) pushFront(node *LRUListNode[K, V]) (*LRUListNode[K, V], error) {
	return l.insert(node, l.root.prev.Load())
}

// PushFront creates an element with key K and value V and adds it to the front
func (l *LRUList[K, V]) PushFront(key K, value V) *LRUListNode[K, V] {
	node := NewLRUListNode(key, value, nil, nil, nil)
	n, _ := l.pushFront(node)

	return n
}

// pushBack is an internal method to add existing node in the back. Added node shouldn't belong to any list
func (l *LRUList[K, V]) pushBack(node *LRUListNode[K, V]) (*LRUListNode[K, V], error) {
	return l.insert(node, &l.root)
}

// PushBack creates an element with key K and value V and adds it to the back
func (l *LRUList[K, V]) PushBack(key K, value V) *LRUListNode[K, V] {
	node := NewLRUListNode(key, value, nil, nil, nil)
	n, _ := l.pushBack(node)
	return n
}

// Remove deleted element that is pointed by at from the list and returns it
func (l *LRUList[K, V]) Remove(at *LRUListNode[K, V]) (*LRUListNode[K, V], error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if at.list.Load() != l {
		return nil, ErrNotInList
	}
	if at == &l.root {
		return nil, ErrRoot
	}

	at.prev.Load().next.Store(at.next.Load())
	at.next.Load().prev.Store(at.prev.Load())

	at.next.Store(nil)
	at.prev.Store(nil)
	at.list.Store(nil)

	l.len -= 1

	return at, nil
}

// PopFront removes the element from the front of the list and returns it
func (l *LRUList[K, V]) PopFront() (*LRUListNode[K, V], error) {
	return l.Remove(l.Front())
}

// PopBack removes the element from the back of the list and returns it
func (l *LRUList[K, V]) PopBack() (*LRUListNode[K, V], error) {
	return l.Remove(l.Back())
}