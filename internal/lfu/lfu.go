package lfu

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

// nodeList represents a list of nodes sharing the same frequency.
type nodeList[K comparable, V any] struct {
	frequency int
	prev      *nodeList[K, V]
	next      *nodeList[K, V]
	first     *nodeValue[K, V]
	last      *nodeValue[K, V]
}

// newNodeList creates a new node list with the specified parameters.
func newNodeList[K comparable, V any](frequency int, first, last *nodeValue[K, V], prev, next *nodeList[K, V]) *nodeList[K, V] {
	return &nodeList[K, V]{
		frequency: frequency,
		first:     first,
		last:      last,
		prev:      prev,
		next:      next,
	}
}

// nodeValue represents a value in a list for the same frequency
type nodeValue[K comparable, V any] struct {
	key       K
	value     V
	next      *nodeValue[K, V]
	prev      *nodeValue[K, V]
	frequency *nodeList[K, V]
}

// list represents a doubly linked list of nodeList.
type list[K comparable, V any] struct {
	sentinel *nodeList[K, V] // sentinel.next = head, sentinel.prev = tail
}

// newList initializes a new list with a sentinel node.
func newList[K comparable, V any]() *list[K, V] {
	sentinel := newNodeList[K, V](0, nil, nil, nil, nil)
	return &list[K, V]{sentinel: sentinel}
}

// addListFrontOrAfter adds a new node list with the specified frequency
// in front of or after the specified node list.
func (l *list[K, V]) addListFrontOrAfter(frequency int, first *nodeValue[K, V], before ...*nodeList[K, V]) {
	bfr := l.sentinel
	if len(before) > 0 {
		bfr = before[0]
	}

	node := newNodeList(frequency, first, first, bfr, bfr.next)

	bfr.next = node
	if node.next != nil {
		node.next.prev = node
	} else {
		l.sentinel.prev = node
	}

	first.frequency = node
	first.prev = nil
}

// addFrontByFreq adds a new node value to the front of the frequency list.
func (l *nodeList[K, V]) addFrontByFreq(newFirst *nodeValue[K, V]) {
	newFirst.prev = nil
	newFirst.next = l.first
	l.first.prev = newFirst
	l.first = newFirst
	newFirst.frequency = l
}

// cacheImpl represents LFU cache implementation
type cacheImpl[K comparable, V any] struct {
	capacity    int
	frequencies list[K, V]
	mp          map[K]*nodeValue[K, V]
}

// New initializes the cache with the given capacity.
// If no capacity is provided, the cache will use DefaultCapacity.
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
		mp:          make(map[K]*nodeValue[K, V], resultCapacity),
		frequencies: *newList[K, V](),
	}
}

// untie disconnects the node from its linked list.
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

// Get returns the value of the key if the key exists in the cache,
// otherwise, returns ErrKeyNotFound.
//
// O(1)
func (l *cacheImpl[K, V]) Get(key K) (V, error) {
	value, exists := l.mp[key]
	if !exists {
		var zeroVal V
		return zeroVal, ErrKeyNotFound
	}

	value.untie()
	currentNode := value.frequency
	if next := currentNode.next; next == nil || currentNode.frequency+1 != next.frequency {
		l.frequencies.addListFrontOrAfter(currentNode.frequency+1, value, currentNode)
	} else {
		currentNode.next.addFrontByFreq(value)
	}
	if currentNode.first == nil {
		prevNode := currentNode.prev
		prevNode.next = currentNode.next
		if currentNode.next != nil {
			currentNode.next.prev = prevNode
		}
	}
	return value.value, nil
}

// Put updates the value of the key if present, or inserts the key if not already present.
//
// When the cache reaches its capacity, it should invalidate and remove the least frequently used key
// before inserting a new item. For this problem, when there is a tie
// (i.e., two or more keys with the same frequencies), the least recently used key would be invalidated.
//
// O(1)
func (l *cacheImpl[K, V]) Put(key K, value V) {
	if node, exists := l.mp[key]; exists {
		node.value = value
		_, err := l.Get(key)
		if err != nil {
			panic(err)
		}
		return
	}

	if l.Size() >= l.capacity {
		l.delLast()
	}

	node := &nodeValue[K, V]{key: key, value: value, frequency: l.frequencies.sentinel.next}
	if firstFreqNode := l.frequencies.sentinel.next; firstFreqNode == nil || firstFreqNode.frequency != 1 {
		l.frequencies.addListFrontOrAfter(1, node)
	} else {
		firstFreqNode.addFrontByFreq(node)
	}
	l.mp[key] = node
}

// All returns the iterator in descending order of frequencies.
// If two or more keys have the same frequencies, the most recently used key will be listed first.
//
// O(capacity)
func (l *cacheImpl[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for freqNode := l.frequencies.sentinel.prev; freqNode != nil && freqNode != l.frequencies.sentinel; freqNode = freqNode.prev {
			for valNode := freqNode.first; valNode != nil; valNode = valNode.next {
				if !yield(valNode.key, valNode.value) {
					return
				}
			}
		}
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

// GetKeyFrequency returns the element's frequencies if the key exists in the cache,
// otherwise, returns ErrKeyNotFound.
//
// O(1)
func (l *cacheImpl[K, V]) GetKeyFrequency(key K) (int, error) {
	val, ex := l.mp[key]
	if !ex {
		return 0, ErrKeyNotFound
	}
	return val.frequency.frequency, nil
}

// delLast removes the least frequently used item from the cache.
func (l *cacheImpl[K, V]) delLast() {
	lastFreqNode := l.frequencies.sentinel.next
	if lastFreqNode == nil {
		return
	}
	delete(l.mp, lastFreqNode.last.key)
	if lastFreqNode.first == nil || lastFreqNode.first.next == nil {
		if lastFreqNode.next != nil {
			lastFreqNode.next.prev = l.frequencies.sentinel
		}
		l.frequencies.sentinel.next = l.frequencies.sentinel.next.next
		return
	}
	lastFreqNode.last.prev.next = nil
	l.frequencies.sentinel.next.last = lastFreqNode.last.prev
}
