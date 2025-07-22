package main

import (
	"fmt"
	"sync"
)

type StoredValue struct {
	val       string
	lval      []string
	isList    bool
	expiresBy int64
	listeners []chan string
}

func (sv *StoredValue) AddChannel(c chan string) {
	if sv.listeners == nil {
		sv.listeners = []chan string{c}
	} else {
		sv.listeners = append(sv.listeners, c)
	}
}

type ConcurrentMap[T any] struct {
	mu sync.RWMutex
	db map[string]T
}

func (cm *ConcurrentMap[T]) Set(key string, val T) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.db[key] = val
}

func (cm *ConcurrentMap[T]) Get(key string) (val T, ok bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	val, ok = cm.db[key]
	return val, ok
}

func (cm *ConcurrentMap[T]) SetOrUpdate(
	key string,
	set func() T,
	update func(*T) error,
) (T, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	val, ok := cm.db[key]
	if !ok {
		val = set()
	} else {
		if err := update(&val); err != nil {
			return val, err
		}
	}

	cm.db[key] = val
	return val, nil
}

func (cm *ConcurrentMap[T]) Update(key string, update func(val *T) error) (T, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	val, ok := cm.db[key]
	if !ok {
		return val, fmt.Errorf("key %q: %w", key, errKeyNotFound)
	}

	if err := update(&val); err != nil {
		return val, err
	}

	cm.db[key] = val
	return val, nil
}

func (cm *ConcurrentMap[T]) Delete(key string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.db, key)
}
