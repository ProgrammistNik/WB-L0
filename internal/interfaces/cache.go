package interfaces

type Cache[K comparable, V any] interface {
	Set(key K, value V)
	Get(key K) (V, bool)
	Delete(key K) error
	Contains(key K) bool
	Flush()
	Size() int
	Capacity() int
	Empty() bool
}