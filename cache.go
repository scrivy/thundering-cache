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

func (m *Cache) Clear() error {
	if m.items == nil {
		return ErrNotInitialized
	}
	m.itemsLock.Lock()
	m.items = make(map[string]interface{})
	m.itemsLock.Unlock()
	return nil
}

func (m *Cache) Get(key string) (value interface{}, ok bool) {
	m.itemsLock.RLock()
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

			// fetch value
			var err error
			value, err = m.fetch(key)
			if err != nil {
				value = err
			} else {
				ok = true
			}

			m.itemsLock.Lock()
			m.items[key] = value
			m.itemsLock.Unlock()

			m.isBeingFetchedLock.Lock()
			m.isBeingFetchedMap[key] = false
			m.isBeingFetchedLock.Unlock()

			wg.Done()
		} else { // prevent thundering herd
			m.isBeingFetchedWG[key].Wait()
			m.itemsLock.RLock()
			value = m.items[key]
			m.itemsLock.RUnlock()
			_, ok = value.(error)
		}
	}
	return
}

// useful when comparing caches that should be identical amongst servers
func (m *Cache) GetAll() map[string]interface{} {
	items := map[string]interface{}{}
	m.itemsLock.RLock()
	for k, v := range m.items {
		items[k] = v
	}
	m.itemsLock.RUnlock()
	return items
}

// TODO needs to be rewritten to accomidate for updating an element when a get calls is already fetching it
/*
func (m *Cache) Update(key string) {
	// get the updated value from the db before purging
	value, err := m.fetch(key)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			m.itemsLock.Lock()
			delete(m.items, key)
			m.itemsLock.Unlock()
		default:
			common.LogErr(m.name+":", err)
		}
	} else {
		m.itemsLock.Lock()
		m.items[key] = value
		m.itemsLock.Unlock()
	}
}

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
