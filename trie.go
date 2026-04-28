// Package dnstrie
package dnstrie

import (
	"strings"
	"sync"
)

// public interface for trie structure
type Trie[T any] interface {
	Del(uri string)
	Get(uri string) (result T, found bool)
	Match(uri string) (result []T)
	Set(uri string, value T)
}

// trie node struct
type node[T any] struct {
	Branches map[string]*node[T] `json:"b,omitempty"`
	IsLeaf   bool                `json:"l,omitempty"`
	Key      string              `json:"k,omitempty"`
	Value    T                   `json:"v,omitempty"`

	mutex  sync.RWMutex
	parent *node[T]
}

// gte a new trie structure
func New[T any]() Trie[T] {
	return &node[T]{}
}

// wildcard match (breadth first on *)
func (current *node[T]) Match(uri string) (result []T) {
	var (
		found bool
		next  *node[T] = nil
		posX  int      = 0
		posY  int      = len(uri)
	)

	for {
		posX = strings.LastIndexByte(uri[:posY], '.')
		key := uri[posX+1 : posY]
		if key == "*" {
			current.mutex.RLock()
			children := make([]*node[T], 0, len(current.Branches))
			for _, child := range current.Branches {
				children = append(children, child)
			}
			current.mutex.RUnlock()
			if posX >= 0 {
				for _, next := range children {
					result = append(result, next.Match(uri[:posX])...)
				}
			} else {
				for _, next := range children {
					if next.IsLeaf {
						result = append(result, next.Value)
					}
				}
			}
			return
		} else {
			current.mutex.RLock()
			if next, found = current.Branches[key]; !found {
				current.mutex.RUnlock()
				return
			}
			current.mutex.RUnlock()
			current = next
		}

		if posY = posX; posY < 0 {
			break
		}
	}

	current.mutex.RLock()
	if current.IsLeaf {
		result = append(result, current.Value)
	}
	current.mutex.RUnlock()
	return
}

// delete leaf-value and prune dead branches
func (current *node[T]) Del(uri string) {
	var (
		found bool     = false
		next  *node[T] = nil
		posX  int      = 0
		posY  int      = len(uri)
	)

	for {
		posX = strings.LastIndexByte(uri[:posY], '.')
		current.mutex.RLock()
		if next, found = current.Branches[uri[posX+1:posY]]; !found {
			current.mutex.RUnlock()
			return
		}
		current.mutex.RUnlock()
		current = next

		if posY = posX; posY < 0 {
			break
		}
	}

	current.mutex.Lock()
	current.IsLeaf = false
	current.mutex.Unlock()

	var empty bool
	for current.parent != nil {
		current.mutex.RLock()
		if empty = len(current.Branches) == 0 && !current.IsLeaf; !empty {
			current.mutex.RUnlock()
			break
		}
		current.mutex.RUnlock()

		current.parent.mutex.Lock()
		current.mutex.RLock()
		if empty = len(current.Branches) == 0 && !current.IsLeaf; !empty {
			current.mutex.RUnlock()
			current.parent.mutex.Unlock()
			break
		}

		current.mutex.RUnlock()
		if empty {
			delete(current.parent.Branches, current.Key)
			current.parent.mutex.Unlock()
			current = current.parent
		} else {
			current.parent.mutex.Unlock()
			break
		}
	}
}

// get leaf-value
func (current *node[T]) Get(uri string) (result T, found bool) {
	var (
		next *node[T] = nil
		posX int      = 0
		posY int      = len(uri)
	)

	for {
		posX = strings.LastIndexByte(uri[:posY], '.')
		current.mutex.RLock()
		if next, found = current.Branches[uri[posX+1:posY]]; !found {
			current.mutex.RUnlock()
			return
		}
		current.mutex.RUnlock()
		current = next

		if posY = posX; posY < 0 {
			break
		}
	}

	current.mutex.RLock()
	if found = current.IsLeaf; found {
		result = current.Value
	}
	current.mutex.RUnlock()
	return
}

// update leaf value and create branch if needed
func (current *node[T]) Set(uri string, value T) {
	var (
		found bool     = false
		next  *node[T] = nil
		posX  int      = 0
		posY  int      = len(uri)
	)

	for {
		posX = strings.LastIndexByte(uri[:posY], '.')
		key := uri[posX+1 : posY]

		current.mutex.Lock()
		if current.Branches == nil {
			next = &node[T]{Key: key, parent: current}
			current.Branches = map[string]*node[T]{key: next}
		} else if next, found = current.Branches[key]; !found {
			next = &node[T]{Key: key, parent: current}
			current.Branches[key] = next
		}
		current.mutex.Unlock()

		current = next
		if posY = posX; posY < 0 {
			break
		}
	}

	current.mutex.Lock()
	current.Value = value
	current.IsLeaf = true
	current.mutex.Unlock()
}
