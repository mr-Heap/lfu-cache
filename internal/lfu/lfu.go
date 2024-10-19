package lfu

import (
	"errors"
	"iter"
	"lfucache/internal/linkedlist"
)

var ErrKeyNotFound = errors.New("key not found")

// DefaultCapacity represents the default capacity of the LFU Cache
const DefaultCapacity = 5

// Cache
// O(capacity) memory
type Cache[K comparable, V any] interface {
	// Get returns the value of the key if the key exists in the cache,
	// otherwise, returns ErrKeyNotFound.
	//
	// O(1)
	Get(key K) (V, error)

	// Put updates the value of the key if present, or inserts the key if not already present.
	//
	// When the cache reaches its capacity, it should invalidate and remove the least frequently used key
	// before inserting a new item. For this problem, when there is a tie
	// (i.e., two or more keys with the same frequencies), the least recently used key would be invalidated.
	//
	// O(1)
	Put(key K, value V)

	// All returns the iterator in descending order of frequencies.
	// If two or more keys have the same frequencies, the most recently used key will be listed first.
	//
	// O(capacity)
	All() iter.Seq2[K, V]

	// Size returns the cache size.
	//
	// O(1)
	Size() int

	// Capacity returns the cache capacity.
	//
	// O(1)
	Capacity() int

	// GetKeyFrequency returns the element's frequencies if the key exists in the cache,
	// otherwise, returns ErrKeyNotFound.
	//
	// O(1)
	GetKeyFrequency(key K) (int, error)
}

type BaseNode[K comparable, V any] *linkedlist.Node[int, *linkedlist.List[K, V]]

type cacheNode[K comparable, V any] struct {
	node     *linkedlist.Node[K, V]
	baseNode *linkedlist.Node[int, *linkedlist.List[K, V]]
}

// cacheImpl represents LFU cache implementation
type cacheImpl[K comparable, V any] struct {
	capacity    int
	frequencies linkedlist.List[int, *linkedlist.List[K, V]]
	mp          map[K]*cacheNode[K, V]
}

// New initializes the cache with the specified capacity.
// If no capacity is provided, it defaults to DefaultCapacity.
//
// Arguments:
//   - capacity: Optional integer specifying the initial capacity of the cache.
//     Must be a positive number if provided.
//
// Returns:
//   - A pointer to a new cacheImpl instance.
func New[K comparable, V any](capacity ...int) *cacheImpl[K, V] {
	resultCapacity := DefaultCapacity
	if len(capacity) > 0 {
		if capacity[0] < 0 {
			panic("Capacity must be positive.")
		}
		resultCapacity = capacity[0]
	}

	return &cacheImpl[K, V]{
		capacity:    resultCapacity,
		frequencies: *linkedlist.NewList[int, *linkedlist.List[K, V]](),
		mp:          make(map[K]*cacheNode[K, V]),
	}
}

// Get returns the value of the key if the key exists in the cache,
// otherwise, returns ErrKeyNotFound.
//
// O(1)
func (l *cacheImpl[K, V]) Get(key K) (V, error) {
	node, exists := l.mp[key]
	if !exists {
		var zeroVal V
		return zeroVal, ErrKeyNotFound
	}

	return l.hangUpNode(node).Value, nil
}

func (l *cacheImpl[K, V]) hangUpNode(node *cacheNode[K, V]) *linkedlist.Node[K, V] {
	value := node.node
	currentFreq := node.baseNode
	nextFreq := currentFreq.Next()
	value.Untie()
	if currentFreq == l.frequencies.Last() || nextFreq.Key != currentFreq.Key+1 {
		newList := linkedlist.NewList[K, V]()
		newList.AddFrontOrAfter(value)
		l.frequencies.AddFrontOrAfter(linkedlist.NewNode(currentFreq.Key+1, newList), currentFreq)
	} else {
		nextFreq.Value.AddFrontOrAfter(value)
	}
	node.baseNode = currentFreq.Next()

	if currentFreq.Value.IsEmpty() {
		currentFreq.Untie()
	}

	return value
}

// GetKeyFrequency returns the element's frequencies if the key exists in the cache,
// otherwise, returns ErrKeyNotFound.
//
// O(1)
func (l *cacheImpl[K, V]) GetKeyFrequency(key K) (int, error) {
	val, ex := l.mp[key]
	if !ex {
		return 0, ErrKeyNotFound
	}
	return val.baseNode.Key, nil
}

// Put updates the value of the key if present, or inserts the key if not already present.
//
// When the cache reaches its capacity, it should invalidate and remove the least frequently used key
// before inserting a new item. For this problem, when there is a tie
// (i.e., two or more keys with the same frequencies), the least recently used key would be invalidated.
//
// O(1)
func (l *cacheImpl[K, V]) Put(key K, value V) {
	if cached, exists := l.mp[key]; exists {
		cached.node.Value = value
		_ = l.hangUpNode(cached)
		return
	}

	if l.Size() >= l.capacity {
		l.delLast()
	}

	node := linkedlist.NewNode(key, value)
	if l.frequencies.First().Key == 1 {
		l.frequencies.First().Value.AddFrontOrAfter(node)
	} else {
		newList := linkedlist.NewList[K, V]()
		newList.AddFrontOrAfter(node)
		l.frequencies.AddFrontOrAfter(linkedlist.NewNode(1, newList))
	}
	l.mp[key] = &cacheNode[K, V]{node: node, baseNode: l.frequencies.First()}
}

// delLast removes the least frequently used item from the cache.
// It updates the internal data structures accordingly to maintain the LFU policy.
func (l *cacheImpl[K, V]) delLast() {
	node := l.frequencies.First().Value.Last()
	node.Untie()
	delete(l.mp, node.Key)
	if l.frequencies.First().Value.IsEmpty() {
		l.frequencies.First().Untie()
	}
}

// Size returns the cache size using the map size
//
// O(1)
func (l *cacheImpl[K, V]) Size() int {
	return len(l.mp)
}

// Capacity returns the cache capacity.
//
// O(1)
func (l *cacheImpl[K, V]) Capacity() int {
	return l.capacity
}

// All returns the iterator in descending order of frequencies.
// If two or more keys have the same frequencies, the most recently used key will be listed first.
//
// O(capacity)
func (l *cacheImpl[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for freqNode := l.frequencies.Last(); freqNode != l.frequencies.First().Prev(); freqNode = freqNode.Prev() {
			for valNode := freqNode.Value.First(); valNode != freqNode.Value.Last().Next(); valNode = valNode.Next() {
				if !yield(valNode.Key, valNode.Value) {
					return
				}
			}
		}
	}
}
