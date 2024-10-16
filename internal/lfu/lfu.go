package lfu

//TODO: optimize что хранить в структурах
import (
	"errors"
	"iter"
)

var ErrKeyNotFound = errors.New("key not found")

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

// nodeList represents a list of nodes with a certain frequency
// Each frequency has its own list of node values.
type nodeList[K comparable, V any] struct {
	frequency int
	prev      *nodeList[K, V]
	next      *nodeList[K, V]
	first     *nodeValue[K, V]
	last      *nodeValue[K, V]
}

// nodeValue represents a key-value pair in the cache,
// along with its frequency and links to other nodes with the same frequency.
type nodeValue[K comparable, V any] struct {
	key       K
	value     V
	next      *nodeValue[K, V]
	prev      *nodeValue[K, V]
	frequency *nodeList[K, V] // Points to the frequency list this node belongs to
}

// list manages all the frequency lists.
// It uses a sentinel node for easier management of head and tail.
type list[K comparable, V any] struct {
	sentinel *nodeList[K, V] // sentinel.next = head, sentinel.prev = tail
	size     int
}

// addListFrontOrAfter adds a new frequency list either at the front
// or after a specific node in the frequency list.
func (l *list[K, V]) addListFrontOrAfter(frequency int, first *nodeValue[K, V], before ...*nodeList[K, V]) {
	bfr := l.sentinel
	if len(before) > 0 {
		bfr = before[0]
	}
	node := &nodeList[K, V]{frequency: frequency, first: first, prev: nil, next: bfr.next, last: first}
	bfr.next = node
	if bfr.next.first != nil {
		bfr.next.first.frequency = bfr.next
	}
	bfr.next.first.prev = nil
	node.prev = bfr
	if node.next == nil || node.next == l.sentinel {
		l.sentinel.prev = node
	}
}

// addFrontByFreq inserts a new node at the front of a frequency list.
func (l *nodeList[K, V]) addFrontByFreq(newFirst *nodeValue[K, V]) {
	newFirst.prev = nil
	newFirst.next = l.first
	l.first.prev = newFirst
	l.first = newFirst
	newFirst.frequency = l
}

// updateLastAfterMove updates the last node reference when a node is removed or moved.
func (l *nodeList[K, V]) updateLastAfterMove(del *nodeValue[K, V]) {
	if del == l.last {
		l.last = del.prev
	}
}

// cacheImpl represents LFU cache implementation
// It manages the capacity, size, and frequency-based list of cache entries.
type cacheImpl[K comparable, V any] struct {
	capacity    int                    // Maximum number of items in the cache
	size        int                    // Current number of items
	frequencies list[K, V]             // Frequency list (stores nodes based on their usage frequency)
	mp          map[K]*nodeValue[K, V] // Key to node mapping
}

// New initializes the cache with the given capacity.
// If no capacity is provided, the cache will use DefaultCapacity.
func New[K comparable, V any](capacity ...int) *cacheImpl[K, V] {
	c := &cacheImpl[K, V]{
		capacity: DefaultCapacity,
		mp:       make(map[K]*nodeValue[K, V]),
	}

	// Initialize sentinel node for frequency list
	c.frequencies.sentinel = &nodeList[K, V]{frequency: 0, prev: nil, next: nil, first: nil, last: nil}

	if len(capacity) > 0 {
		c.capacity = capacity[0]
	}
	return c
}

// untie removes a node from the doubly-linked list of nodes for a given frequency.
func (value *nodeValue[K, V]) untie() {
	if value.prev != nil {
		value.prev.next = value.next
	}
	if value.next != nil {
		value.next.prev = value.prev
	}
	if value.frequency.first == value {
		value.frequency.first = value.next
	}
	if value.frequency.last == value {
		value.frequency.last = value.prev
	}
	value.prev = nil
	value.next = nil
}

// Get retrieves the value for the given key.
// If the key exists, its frequency is increased and the node is moved to the appropriate frequency list.
func (l *cacheImpl[K, V]) Get(key K) (V, error) {
	value, ex := l.mp[key]
	if !ex {
		var zeroVal V
		return zeroVal, ErrKeyNotFound
	}

	if value.frequency.next == nil || value.frequency.frequency+1 != value.frequency.next.frequency {
		value.untie()
		l.frequencies.addListFrontOrAfter(value.frequency.frequency+1, value, value.frequency)
	} else {
		value.untie()
		value.frequency.next.addFrontByFreq(value)
	}
	return value.value, nil
}

// Put inserts or updates a key-value pair in the cache.
// If the cache is full, it evicts the least frequently used item.
func (l *cacheImpl[K, V]) Put(key K, value V) {
	if _, exists := l.mp[key]; exists {
		l.mp[key].value = value
		l.Get(key)
		return
	}

	if l.Size() >= l.capacity {
		l.delLast()
	}

	node := &nodeValue[K, V]{key: key, value: value, next: nil, prev: nil, frequency: l.frequencies.sentinel.next}
	if l.frequencies.sentinel.next == nil || l.frequencies.sentinel.next.frequency != 1 {
		l.frequencies.addListFrontOrAfter(1, node)
	} else {
		l.frequencies.sentinel.next.addFrontByFreq(node)
	}
	l.mp[key] = node
}

// All returns an iterator over the cache's items in descending order of frequency.
func (l *cacheImpl[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for freqNode := l.frequencies.sentinel.prev; freqNode != l.frequencies.sentinel; freqNode = freqNode.prev {
			for valNode := freqNode.first; valNode != nil; valNode = valNode.next {
				if !yield(valNode.key, valNode.value) {
					return
				}
			}
		}
	}
}

// Size returns the current size of the cache.
func (l *cacheImpl[K, V]) Size() int {
	return l.size
}

// Capacity returns the cache's capacity.
func (l *cacheImpl[K, V]) Capacity() int {
	return l.capacity
}

// GetKeyFrequency retrieves the frequency of a key in the cache.
func (l *cacheImpl[K, V]) GetKeyFrequency(key K) (int, error) {
	val, ex := l.mp[key]
	if !ex {
		return 0, ErrKeyNotFound
	}
	return val.frequency.frequency, nil
}

// delLast removes the least frequently used item from the cache.
func (l *cacheImpl[K, V]) delLast() {
	if l.frequencies.sentinel.next == nil || l.frequencies.sentinel.next.first == nil {
		return
	}
	if l.frequencies.sentinel.next.first.next == nil {
		l.frequencies.sentinel.next.next.prev = l.frequencies.sentinel
		l.frequencies.sentinel.next = l.frequencies.sentinel.next.next
		return
	}
	l.frequencies.sentinel.next.last.prev.next = nil
	l.frequencies.sentinel.next.last = l.frequencies.sentinel.next.last.prev
}
