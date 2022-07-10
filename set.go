package structures

import (
	"encoding/json"
	"sync"
)

type Set[V comparable] struct {
	mu   sync.RWMutex
	data map[V]struct{}
}

// NewSet create and returns a new set, which contains un-repeated items.
// The parameter `safe` is used to specify whether using set in concurrent-safety,
// which is false in default.
func NewSet[V comparable]() *Set[V] {
	return &Set[V]{
		data: make(map[V]struct{}),
		mu:   sync.RWMutex{},
	}
}

// NewSetFrom returns a new set from `items`.
// Parameter `items` can be either a variable of any type, or a slice.
func NewSetFrom[V comparable](items []V) *Set[V] {
	m := make(map[V]struct{})
	for _, v := range items {
		m[v] = struct{}{}
	}
	return &Set[V]{
		data: m,
		mu:   sync.RWMutex{},
	}
}

// Iterator iterates the set readonly with given callback function `f`,
// if `f` returns true then continue iterating; or false to stop.
func (set *Set[V]) Iterator(f func(v V) bool) {
	set.mu.RLock()
	defer set.mu.RUnlock()
	for k, _ := range set.data {
		if !f(k) {
			break
		}
	}
}

// Add adds one or multiple items to the set.
func (set *Set[V]) Add(items ...V) {
	set.mu.Lock()
	defer set.mu.RUnlock()
	if set.data == nil {
		set.data = make(map[V]struct{})
	}
	for _, v := range items {
		set.data[v] = struct{}{}
	}
}

// AddIfNotExist checks whether item exists in the set,
// it adds the item to set and returns true if it does not exists in the set,
// or else it does nothing and returns false.
//
// Note that, if `item` is nil, it does nothing and returns false.
func (set *Set[V]) AddIfNotExist(item V) bool {
	if !set.Contains(item) {
		set.mu.Lock()
		defer set.mu.Unlock()
		if set.data == nil {
			set.data = make(map[V]struct{})
		}
		if _, ok := set.data[item]; !ok {
			set.data[item] = struct{}{}
			return true
		}
	}
	return false
}

// AddIfNotExistFunc checks whether item exists in the set,
// it adds the item to set and returns true if it does not exist in the set and
// function `f` returns true, or else it does nothing and returns false.
//
// Note that, if `item` is nil, it does nothing and returns false. The function `f`
// is executed without writing lock.
func (set *Set[V]) AddIfNotExistFunc(item V, f func() bool) bool {
	if !set.Contains(item) {
		if f() {
			set.mu.Lock()
			defer set.mu.Unlock()
			if set.data == nil {
				set.data = make(map[V]struct{})
			}
			if _, ok := set.data[item]; !ok {
				set.data[item] = struct{}{}
				return true
			}
		}
	}
	return false
}

// AddIfNotExistFuncLock checks whether item exists in the set,
// it adds the item to set and returns true if it does not exists in the set and
// function `f` returns true, or else it does nothing and returns false.
//
// Note that, if `item` is nil, it does nothing and returns false. The function `f`
// is executed within writing lock.
func (set *Set[V]) AddIfNotExistFuncLock(item V, f func() bool) bool {
	if !set.Contains(item) {
		set.mu.Lock()
		defer set.mu.Unlock()
		if set.data == nil {
			set.data = make(map[V]struct{})
		}
		if f() {
			if _, ok := set.data[item]; !ok {
				set.data[item] = struct{}{}
				return true
			}
		}
	}
	return false
}

// Contains checks whether the set contains `item`.
func (set *Set[V]) Contains(item V) bool {
	var ok bool
	set.mu.RLock()
	if set.data != nil {
		_, ok = set.data[item]
	}
	set.mu.RUnlock()
	return ok
}

// Remove deletes `item` from set.
func (set *Set[V]) Remove(item V) {
	set.mu.Lock()
	if set.data != nil {
		delete(set.data, item)
	}
	set.mu.Unlock()
}

// Size returns the size of the set.
func (set *Set[V]) Size() int {
	set.mu.RLock()
	l := len(set.data)
	set.mu.RUnlock()
	return l
}

// Clear deletes all items of the set.
func (set *Set[V]) Clear() {
	set.mu.Lock()
	set.data = make(map[V]struct{})
	set.mu.Unlock()
}

// Slice returns the a of items of the set as slice.
func (set *Set[V]) Slice() []V {
	set.mu.RLock()
	var (
		i   = 0
		ret = make([]V, len(set.data))
	)
	for item := range set.data {
		ret[i] = item
		i++
	}
	set.mu.RUnlock()
	return ret
}

// LockFunc locks writing with callback function `f`.
func (set *Set[V]) LockFunc(f func(m map[V]struct{})) {
	set.mu.Lock()
	defer set.mu.Unlock()
	f(set.data)
}

// RLockFunc locks reading with callback function `f`.
func (set *Set[V]) RLockFunc(f func(m map[V]struct{})) {
	set.mu.RLock()
	defer set.mu.RUnlock()
	f(set.data)
}

// Equal checks whether the two sets equal.
func (set *Set[V]) Equal(other *Set[V]) bool {
	if set == other {
		return true
	}
	set.mu.RLock()
	defer set.mu.RUnlock()
	other.mu.RLock()
	defer other.mu.RUnlock()
	if len(set.data) != len(other.data) {
		return false
	}
	for key := range set.data {
		if _, ok := other.data[key]; !ok {
			return false
		}
	}
	return true
}

// IsSubsetOf checks whether the current set is a sub-set of `other`.
func (set *Set[V]) IsSubsetOf(other *Set[V]) bool {
	if set == other {
		return true
	}
	set.mu.RLock()
	defer set.mu.RUnlock()
	other.mu.RLock()
	defer other.mu.RUnlock()
	for key := range set.data {
		if _, ok := other.data[key]; !ok {
			return false
		}
	}
	return true
}

// Union returns a new set which is the union of `set` and `others`.
// Which means, all the items in `newSet` are in `set` or in `others`.
func (set *Set[V]) Union(others ...*Set[V]) (newSet *Set[V]) {
	newSet = NewSet[V]()
	set.mu.RLock()
	defer set.mu.RUnlock()
	for _, other := range others {
		if set != other {
			other.mu.RLock()
		}
		for k, v := range set.data {
			newSet.data[k] = v
		}
		if set != other {
			for k, v := range other.data {
				newSet.data[k] = v
			}
		}
		if set != other {
			other.mu.RUnlock()
		}
	}

	return
}

// Diff returns a new set which is the difference set from `set` to `others`.
// Which means, all the items in `newSet` are in `set` but not in `others`.
func (set *Set[V]) Diff(others ...*Set[V]) (newSet *Set[V]) {
	newSet = NewSet[V]()
	set.mu.RLock()
	defer set.mu.RUnlock()
	for _, other := range others {
		if set == other {
			continue
		}
		other.mu.RLock()
		for k, v := range set.data {
			if _, ok := other.data[k]; !ok {
				newSet.data[k] = v
			}
		}
		other.mu.RUnlock()
	}
	return
}

// Intersect returns a new set which is the intersection from `set` to `others`.
// Which means, all the items in `newSet` are in `set` and also in `others`.
func (set *Set[V]) Intersect(others ...*Set[V]) (newSet *Set[V]) {
	newSet = NewSet[V]()
	set.mu.RLock()
	defer set.mu.RUnlock()
	for _, other := range others {
		if set != other {
			other.mu.RLock()
		}
		for k, v := range set.data {
			if _, ok := other.data[k]; ok {
				newSet.data[k] = v
			}
		}
		if set != other {
			other.mu.RUnlock()
		}
	}
	return
}

// Complement returns a new set which is the complement from `set` to `full`.
// Which means, all the items in `newSet` are in `full` and not in `set`.
//
// It returns the difference between `full` and `set`
// if the given set `full` is not the full set of `set`.
func (set *Set[V]) Complement(full *Set[V]) (newSet *Set[V]) {
	newSet = NewSet[V]()
	set.mu.RLock()
	defer set.mu.RUnlock()
	if set != full {
		full.mu.RLock()
		defer full.mu.RUnlock()
	}
	for k, v := range full.data {
		if _, ok := set.data[k]; !ok {
			newSet.data[k] = v
		}
	}
	return
}

// Merge adds items from `others` sets into `set`.
func (set *Set[V]) Merge(others ...*Set[V]) *Set[V] {
	set.mu.Lock()
	defer set.mu.Unlock()
	for _, other := range others {
		if set != other {
			other.mu.RLock()
		}
		for k, v := range other.data {
			set.data[k] = v
		}
		if set != other {
			other.mu.RUnlock()
		}
	}
	return set
}

// Pop randomly pops an item from set.
func (set *Set[V]) Pop() (value V) {
	set.mu.Lock()
	defer set.mu.Unlock()
	for k, _ := range set.data {
		value = k
		delete(set.data, k)
		return k
	}
	return
}

// Pops randomly pops `size` items from set.
// It returns all items if size == -1.
func (set *Set[V]) Pops(size int) []V {
	set.mu.Lock()
	defer set.mu.Unlock()
	if size > len(set.data) || size == -1 {
		size = len(set.data)
	}
	if size <= 0 {
		return nil
	}
	index := 0
	array := make([]V, size)
	for k, _ := range set.data {
		delete(set.data, k)
		array[index] = k
		index++
		if index == size {
			break
		}
	}
	return array
}

// Walk applies a user supplied function `f` to every item of set.
func (set *Set[V]) Walk(f func(item V) V) *Set[V] {
	set.mu.Lock()
	defer set.mu.Unlock()
	m := make(map[V]struct{}, len(set.data))
	for k, v := range set.data {
		m[f(k)] = v
	}
	set.data = m
	return set
}

// MarshalJSON implements the interface MarshalJSON for json.Marshal.
func (set Set[V]) MarshalJSON() ([]byte, error) {
	return json.Marshal(set.Slice())
}

// UnmarshalJSON implements the interface UnmarshalJSON for json.Unmarshal.
func (set *Set[V]) UnmarshalJSON(b []byte) error {
	set.mu.Lock()
	defer set.mu.Unlock()
	if set.data == nil {
		set.data = make(map[V]struct{})
	}
	var array []V
	if err := json.Unmarshal(b, &array); err != nil {
		return err
	}
	for _, v := range array {
		set.data[v] = struct{}{}
	}
	return nil
}
