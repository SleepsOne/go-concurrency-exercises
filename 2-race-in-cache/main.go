//////////////////////////////////////////////////////////////////////
//
// Given is some code to cache key-value pairs from a database into
// the main memory (to reduce access time). Note that golang's map are
// not entirely thread safe. Multiple readers are fine, but multiple
// writers are not. Change the code to make this thread safe.
//

package main

import (
	"container/list"
	"sync"
	"testing"

	"golang.org/x/sync/singleflight"
)

// CacheSize determines how big the cache can grow
const CacheSize = 100

// KeyStoreCacheLoader is an interface for the KeyStoreCache
type KeyStoreCacheLoader interface {
	// Load implements a function where the cache should gets it's content from
	Load(string) string
}

type page struct {
	Key   string
	Value string
}

// KeyStoreCache is a LRU cache for string key-value pairs
type KeyStoreCache struct {
	mu    sync.RWMutex
	cache map[string]*list.Element
	pages list.List
	load  func(string) string
	sf    singleflight.Group
}

// New creates a new KeyStoreCache
func New(load KeyStoreCacheLoader) *KeyStoreCache {
	return &KeyStoreCache{
		load:  load.Load,
		cache: make(map[string]*list.Element),
		mu:    sync.RWMutex{},
		sf:    singleflight.Group{},
	}
}

// Get gets the key from cache, loads it from the source if needed
func (k *KeyStoreCache) Get(key string) string {
	// -------- Fast path (read lock) --------
	k.mu.RLock()
	if e, ok := k.cache[key]; ok {
		val := e.Value.(page).Value
		k.mu.RUnlock()

		// MoveToFront là mutation => cần write lock
		k.mu.Lock()
		k.pages.MoveToFront(e)
		k.mu.Unlock()

		return val
	}
	k.mu.RUnlock()

	// -------- Slow path (miss) --------
	// Dùng singleflight tránh nhiều goroutine load cùng 1 key
	val, _, _ := k.sf.Do(key, func() (interface{}, error) {
		return k.load(key), nil
	})

	p := page{Key: key, Value: val.(string)}

	// -------- Update cache (write lock) --------
	k.mu.Lock()
	defer k.mu.Unlock()

	// Double-check: trong lúc chờ load, goroutine khác có thể đã thêm key
	if e, ok := k.cache[key]; ok {
		k.pages.MoveToFront(e)
		return e.Value.(page).Value
	}

	// Nếu cache đầy thì evict LRU
	if len(k.cache) >= CacheSize {
		end := k.pages.Back()
		if end != nil {
			delete(k.cache, end.Value.(page).Key)
			k.pages.Remove(end)
		}
	}

	// Insert mới vào đầu list
	k.pages.PushFront(p)
	k.cache[key] = k.pages.Front()

	return p.Value
}

// Loader implements KeyStoreLoader
type Loader struct {
	DB *MockDB
}

// Load gets the data from the database
func (l *Loader) Load(key string) string {
	val, err := l.DB.Get(key)
	if err != nil {
		panic(err)
	}

	return val
}

func run(t *testing.T) (*KeyStoreCache, *MockDB) {
	loader := Loader{
		DB: GetMockDB(),
	}
	cache := New(&loader)

	RunMockServer(cache, t)

	return cache, loader.DB
}

func main() {
	run(nil)
}
