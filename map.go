package structures

import (
	"encoding/json"
	"errors"
	"sync"
)

var (
	KeyNotFound  = errors.New("key was not found")
	DuplicateKey = errors.New("key already exists")
)

type Map[K comparable, V any] struct {
	data map[K]V
	sync.RWMutex
}

// NewMap will create a new, empty instance of Map
func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		data:    make(map[K]V),
		RWMutex: sync.RWMutex{},
	}
}

// Add sets key-value to the hash map. returns an error if the key already exists
func (l *Map[K, V]) Add(key K, value V) error {
	l.Lock()
	defer l.Unlock()

	if l.data == nil {
		l.data = make(map[K]V)
	}

	if _, exists := l.data[key]; exists {
		return DuplicateKey
	}

	l.data[key] = value
	return nil
}

// Set sets key-value to the hash map.
func (l *Map[K, V]) Set(key K, value V) {
	l.Lock()
	defer l.Unlock()

	if l.data == nil {
		l.data = make(map[K]V)
	}
	l.data[key] = value
}

// Get returns the value by given `key`.
func (l *Map[K, V]) Get(key K) (value V) {
	l.RLock()
	defer l.RUnlock()

	if l.data != nil {
		value, _ = l.data[key]
	}
	return
}

// Size returns the size of the map.
func (l *Map[K, V]) Size() int {
	l.RLock()
	defer l.RUnlock()

	return len(l.data)
}

// IsEmpty checks whether the map is empty. It returns true if map is empty, or else false.
func (l *Map[K, V]) IsEmpty() bool {
	return l.Size() == 0
}

// Iterator iterates the hash map readonly with custom callback function `f`.  If `f` returns true, then it continues iterating; or false to stop.
func (l *Map[K, V]) Iterator(f func(key K, value V) bool) {
	l.RLock()
	defer l.RUnlock()
	for k, v := range l.data {
		if !f(k, v) {
			break
		}
	}
}

// Sets batch sets key-values to the hash map.
func (l *Map[K, V]) Sets(data map[K]V) {
	l.Lock()
	defer l.Unlock()

	if l.data == nil {
		l.data = data
	} else {
		for k, v := range data {
			l.data[k] = v
		}
	}
}

// Search searches the map with given `key`. Second return parameter `found` is true if key was found, otherwise false.
func (l *Map[K, V]) Search(key K) (value V, found bool) {
	l.RLock()
	defer l.RUnlock()

	if l.data != nil {
		value, found = l.data[key]
	}
	return
}

// GetOrSet returns the value by key, or sets value with given `value` if it does not exist and then returns this value.
func (l *Map[K, V]) GetOrSet(key K, value V) interface{} {
	if v, ok := l.Search(key); !ok {
		return l.doSetWithLockCheck(key, value)
	} else {
		return v
	}
}

// GetOrSetFuncLock returns the value by key,
// or sets value with returned value of callback function `f` if it does not exist
// and then returns this value.
//
// GetOrSetFuncLock differs with GetOrSetFunc function is that it executes function `f`
// with mutex.Lock of the hash map.
func (l *Map[K, V]) GetOrSetFuncLock(key K, f func() V) V {
	if v, ok := l.Search(key); !ok {
		return l.doSetWithLockCheck(key, f)
	} else {
		return v
	}
}

// Remove deletes value from map by given `key`, and return this deleted value.
func (l *Map[K, V]) Remove(key K) (value V) {
	l.Lock()
	defer l.Unlock()

	if l.data != nil {
		var ok bool
		if value, ok = l.data[key]; ok {
			delete(l.data, key)
		}
	}
	return
}

// Removes batch deletes values of the map by keys.
func (l *Map[K, V]) Removes(keys ...K) {
	l.Lock()
	defer l.Unlock()

	if l.data != nil {
		for _, key := range keys {
			delete(l.data, key)
		}
	}
}

// Keys returns all keys of the map as a slice.
func (l *Map[K, V]) Keys() []K {
	l.RLock()
	defer l.RUnlock()

	var (
		keys  = make([]K, len(l.data))
		index = 0
	)
	for key := range l.data {
		keys[index] = key
		index++
	}
	return keys
}

// Values returns all values of the map as a slice.
func (l *Map[K, V]) Values() []V {
	l.RLock()
	defer l.RUnlock()

	var (
		values = make([]V, len(l.data))
		index  = 0
	)
	for _, value := range l.data {
		values[index] = value
		index++
	}
	return values
}

// Contains checks whether a key exists. It returns true if the `key` exists, or else false.
func (l *Map[K, V]) Contains(key K) (ok bool) {
	l.RLock()
	defer l.RUnlock()

	if l.data != nil {
		_, ok = l.data[key]
	}
	return
}

// Clear deletes all data of the map, it will remake a new underlying data map.
func (l *Map[K, V]) Clear() {
	l.Lock()
	defer l.Unlock()

	l.data = make(map[K]V)
}

// Merge merges two hash maps.
// The `other` map will be merged into the map `m`.
func (l *Map[K, V]) Merge(others ...*Map[K, V]) {
	l.Lock()
	defer l.Unlock()

	if l.data == nil {
		l.data = make(map[K]V)
	}
	for _, other := range others {
		other.Iterator(func(key K, val V) bool {
			l.data[key] = val
			return true
		})
	}
}

// MapCopy returns a copy of the underlying data of the hash map.
func (l *Map[K, V]) MapCopy() map[K]V {
	l.RLock()
	defer l.RUnlock()

	data := make(map[K]V, len(l.data))
	for k, v := range l.data {
		data[k] = v
	}
	return data
}

// doSetWithLockCheck checks whether value of the key exists with mutex.Lock,
// if not exists, set value to the map with given `key`,
// or else just return the existing value.
//
// When setting value, if `value` is type of `func() interface {}`,
// it will be executed with mutex.Lock of the hash map,
// and its return value will be set to the map with `key`.
//
// It returns value with given `key`.
func (l *Map[K, V]) doSetWithLockCheck(key K, value interface{}) V {
	l.Lock()
	defer l.Unlock()

	if l.data == nil {
		l.data = make(map[K]V)
	}
	if v, ok := l.data[key]; ok {
		return v
	}
	if f, ok := value.(func() V); ok {
		value = f()
	}
	val, ok := value.(V)
	if ok {
		l.data[key] = val
		return val
	}
	return val
}

// MarshalJSON implements the interface MarshalJSON for json.Marshal.
func (l *Map[K, V]) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.data)
}

// UnmarshalJSON implements the interface UnmarshalJSON for json.Unmarshal.
func (l *Map[K, V]) UnmarshalJSON(b []byte) error {
	l.Lock()
	defer l.Unlock()

	if l.data == nil {
		l.data = make(map[K]V)
	}
	var data map[K]V
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	for k, v := range data {
		l.data[k] = v
	}
	return nil
}
