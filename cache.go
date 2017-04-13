package tcache

import (
	"errors"
	"sync"
)

var (
	ErrNotInitialized = errors.New("initialize the cache by calling New, not creating an empty struct")
)

type Cache struct {
	items              map[string]interface{}
	itemsLock          sync.RWMutex
	fetch              func(key string) (interface{}, error)
	isBeingFetchedMap  map[string]bool
	isBeingFetchedLock sync.RWMutex
	isBeingFetchedWG   map[string]*sync.WaitGroup
	preWarmInit        *func() (map[string]interface{}, error)
}

// Pass in the function that fetches the values when there's a cache miss
func New(fetch func(string) (interface{}, error), preWarmInit *func() (map[string]interface{}, error)) (cache *Cache, err error) {
	var items map[string]interface{}

	// prewarm the cache if preWarmInit is defined
	if preWarmInit != nil {
		items, err = (*preWarmInit)()
		if err != nil {
			return
		}
	}

	// Initialize empty map without prewarming
	if preWarmInit == nil {
		items = make(map[string]interface{})
	}

	cache = &Cache{
		items:             items,
		fetch:             fetch,
		isBeingFetchedMap: make(map[string]bool),
		isBeingFetchedWG:  make(map[string]*sync.WaitGroup),
		preWarmInit:       preWarmInit,
	}
	return
}

/*  transparently fetches a result if it's a cache miss, while
    also blocking other gets to the same key and insuring that
    only 1 fetch per key happens at a time to prevent a cache
    stampede
*/
func (m *Cache) Get(key string) (value interface{}, err error) {
	var ok bool
	m.itemsLock.RLock()
	if m.items == nil {
		m.itemsLock.RUnlock()
		err = ErrNotInitialized
		return
	}
	value, ok = m.items[key]
	m.itemsLock.RUnlock()

	if !ok {
		// check if it's already being fetched
		m.isBeingFetchedLock.RLock()
		beingFetched := m.isBeingFetchedMap[key]
		m.isBeingFetchedLock.RUnlock()

		if !beingFetched {
			m.isBeingFetchedLock.Lock()
			m.isBeingFetchedMap[key] = true
			wg := m.isBeingFetchedWG[key]
			if wg == nil {
				wg = &sync.WaitGroup{}
				m.isBeingFetchedWG[key] = wg
			}
			wg.Add(1)
			m.isBeingFetchedLock.Unlock()
			defer wg.Done()

			// fetch value
			value, err = m.fetch(key)
			if err != nil {
				return
			}

			m.itemsLock.Lock()
			m.items[key] = value
			m.itemsLock.Unlock()

			m.isBeingFetchedLock.Lock()
			m.isBeingFetchedMap[key] = false
			m.isBeingFetchedLock.Unlock()
		} else { // prevent thundering herd
			m.isBeingFetchedWG[key].Wait()
			m.itemsLock.RLock()
			value = m.items[key]
			m.itemsLock.RUnlock()
		}
	}
	return
}

// useful when comparing caches that should be identical amongst servers
// TODO rename the function so it doesn't imply that fetches are involved
func (m *Cache) GetAll() map[string]interface{} {
	items := map[string]interface{}{}
	m.itemsLock.RLock()
	defer m.itemsLock.RUnlock()
	if m.items == nil {
		return nil
	}
	for k, v := range m.items {
		items[k] = v
	}
	return items
}

func (m *Cache) Clear() {
	m.itemsLock.Lock()
	m.items = make(map[string]interface{})
	m.itemsLock.Unlock()
	return
}

// TODO needs to be rewritten to accomidate for updating an element when a get calls is already fetching it

func (m *Cache) Update(key string) (err error) {
	m.isBeingFetchedLock.RLock()
	beingFetched := m.isBeingFetchedMap[key]
	m.isBeingFetchedLock.RUnlock()

	var value interface{}
	if !beingFetched {
		m.isBeingFetchedLock.Lock()
		m.isBeingFetchedMap[key] = true
		wg := m.isBeingFetchedWG[key]
		if wg == nil {
			wg = &sync.WaitGroup{}
			m.isBeingFetchedWG[key] = wg
		}
		wg.Add(1)
		m.isBeingFetchedLock.Unlock()
		defer wg.Done()

		// fetch value
		value, err = m.fetch(key)
		if err != nil {
			return
		}

		m.itemsLock.Lock()
		m.items[key] = value
		m.itemsLock.Unlock()

		m.isBeingFetchedLock.Lock()
		m.isBeingFetchedMap[key] = false
		m.isBeingFetchedLock.Unlock()
	} else {
		m.isBeingFetchedWG[key].Wait()
		value, err = m.fetch(key)
		if err != nil {
			return
		}
		m.itemsLock.Lock()
		m.items[key] = value
		m.itemsLock.Unlock()
	}
	return
}

/*
func (m *Cache) PurgeAndInit() {
	var items map[string]interface{}
	var err error
	if m.preWarmInit != nil {
		items, err = (*m.preWarmInit)()
		if err != nil {
			common.LogErr(m.name+" preWarmInit", err)
		}
	}
	if err != nil || m.preWarmInit == nil {
		items = make(map[string]interface{})
	}
	m.itemsLock.Lock()
	m.items = items
	m.itemsLock.Unlock()
}
*/
