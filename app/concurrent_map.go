package main

import (
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
	// cm.mu.Lock()
	cm.db[key] = val
	// cm.mu.Unlock()
}

func (cm *ConcurrentMap[T]) Get(key string) (val T, ok bool) {
	// cm.mu.RLock()
	val, ok = cm.db[key]
	// cm.mu.RUnlock()
	return val, ok
}

func (cm *ConcurrentMap[T]) Delete(key string) {
	cm.mu.Lock()
	delete(cm.db, key)
	cm.mu.Unlock()
}
