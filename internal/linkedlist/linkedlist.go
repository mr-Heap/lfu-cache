package linkedlist

// Node represents a node in the doubly linked list.
// It stores a key of type K and a value of type V.
// Each node has pointers to the next and previous nodes in the list.
type Node[K comparable, V any] struct {
	Key   K           // The key associated with the node.
	Value V           // The value stored in the node.
	next  *Node[K, V] // Pointer to the next node in the list.
	prev  *Node[K, V] // Pointer to the previous node in the list.
}

// NewNode creates a new node with the specified key and value.
// It initializes the next and previous pointers with empty nodes.
func NewNode[K comparable, V any](key K, value V) *Node[K, V] {
	next := &Node[K, V]{}
	prev := &Node[K, V]{}
	return &Node[K, V]{Key: key, Value: value, next: next, prev: prev}
}

// List represents a doubly linked list.
// It uses a sentinel node to simplify boundary conditions.
type List[K comparable, V any] struct {
	sentinel *Node[K, V]
}

// NewList creates and initializes a new doubly linked list.
// It initializes the sentinel node, setting its next and prev pointers to itself.
// This makes it easier to manage the head and tail.
func NewList[K comparable, V any]() *List[K, V] {
	sentinel := &Node[K, V]{}
	sentinel.next = sentinel
	sentinel.prev = sentinel
	return &List[K, V]{sentinel: sentinel}
}

// AddFrontOrAfter inserts a new node either at the front of the list
// or after the specified node (if provided).
// The default behavior is to add the node right after the sentinel,
// making it the first node in the list.
func (l *List[K, V]) AddFrontOrAfter(newNode *Node[K, V], before ...*Node[K, V]) {
	bfr := l.sentinel
	if len(before) > 0 {
		bfr = before[0]
	}

	newNode.prev = bfr
	newNode.next = bfr.next
	bfr.next = newNode
	if newNode.next != nil {
		newNode.next.prev = newNode
	} else {
		l.sentinel.prev = newNode
	}
}

// Last returns the last node in the list (the node before the sentinel).
func (l *List[K, V]) Last() *Node[K, V] {
	return l.sentinel.prev
}

// First returns the first node in the list (the node right after the sentinel).
func (l *List[K, V]) First() *Node[K, V] {
	return l.sentinel.next
}

// IsEmpty checks if the list is empty by comparing the sentinel's next pointer with itself.
// If the sentinel's next points to itself, the list is considered empty.
func (l *List[K, V]) IsEmpty() bool {
	return l.sentinel == nil || l.sentinel.next == l.sentinel
}

// Untie removes the node from the list by updating the previous and next nodes' pointers
// to bypass the current node. After calling this function, the node is "unlinked" from the list.
func (n *Node[K, V]) Untie() {
	n.next.prev = n.prev
	n.prev.next = n.next
	n.prev = nil
	n.next = nil
}

// Next returns the next node in the list.
func (n *Node[K, V]) Next() *Node[K, V] {
	return n.next
}

// Prev returns the previous node in the list.
func (n *Node[K, V]) Prev() *Node[K, V] {
	return n.prev
}
