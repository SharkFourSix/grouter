package grouter

import (
	"sync"
	"time"
)

// Routing state. This is attached to a Session
type routerState struct {
	state     string
	timestamp time.Time
}

type stateCache struct {
	store  sync.Map
	ticker *time.Ticker
	done   chan bool
}

func newStateCache(frequency, stateTTL time.Duration) *stateCache {
	c := &stateCache{
		ticker: time.NewTicker(frequency),
		done:   make(chan bool),
	}
	go func(store *stateCache) {
		for {
			select {
			case <-store.done:
				store.ticker.Stop()
				return
			case <-store.ticker.C:
				store.evict(stateTTL)
			}
		}
	}(c)
	return c
}

func (c *stateCache) get(name string) (string, bool) {
	if vi, ok := c.store.Load(name); ok {
		return vi.(*routerState).state, ok
	}
	return "", false
}

func (c *stateCache) set(name string, state string) {
	c.store.Store(name, &routerState{timestamp: time.Now(), state: state})
}

func (c *stateCache) evict(ttl time.Duration) {
	now := time.Now()
	c.store.Range(func(key, value any) bool {
		state := value.(*routerState)
		if now.Sub(state.timestamp) >= ttl {
			c.store.Delete(key)
		}
		return true
	})
}
