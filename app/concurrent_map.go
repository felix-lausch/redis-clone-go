package main

import (
	"sync"
)

type StoredValue struct {
	val       string
	expiresBy int64
}

type ConcurrentMap struct {
	mu sync.RWMutex
	db map[string]StoredValue
}

func (cm *ConcurrentMap) Set(key string, val StoredValue) {
	cm.mu.Lock()
	cm.db[key] = val
	cm.mu.Unlock()
}

func (cm *ConcurrentMap) Get(key string) (val StoredValue, ok bool) {
	cm.mu.RLock()
	val, ok = cm.db[key]
	cm.mu.RUnlock()

	return val, ok
}

func (cm *ConcurrentMap) Delete(key string) {
	cm.mu.Lock()
	delete(cm.db, key)
	cm.mu.Unlock()
}
