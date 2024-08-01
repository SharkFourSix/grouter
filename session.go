// Session storage
package grouter

import (
	"sync"
	"time"
)

type UssdSession interface {
	// Unique ID of this session
	ID() string
	Set(key string, value any)
	Get(key string) (any, bool)
	MustGet(key string) any
	Del(key string)
	CreatedAt() time.Time
}

type Storage interface {
	Set(key string, session UssdSession)
	Get(key string) UssdSession
	Del(key string)
	// Vacuum removes sessions that are as old as the given duration
	Vacuum(duration time.Duration)
}

type inMemoryStore struct {
	vacuumch chan bool
	ticker   *time.Ticker
	//store    cmap.ConcurrentMap
	store sync.Map
}

func (mss *inMemoryStore) Set(key string, sess UssdSession) {
	mss.store.Store(key, sess)
}

func (mss *inMemoryStore) Get(key string) UssdSession {
	if sess, ok := mss.store.Load(key); ok && sess != nil {
		return sess.(UssdSession)
	}
	return nil
}

func (mss *inMemoryStore) Del(key string) {
	mss.store.Delete(key)
}

func (mss *inMemoryStore) Vacuum(duration time.Duration) {
	var now = time.Now()
	mss.store.Range(func(key, value any) bool {
		session := value.(UssdSession)
		lifespan := now.Sub(session.CreatedAt())
		if lifespan >= duration {
			mss.store.Delete(key)
		}
		return true
	})
}

func NewInMemorySessionStorage(frequency, sessionTTL time.Duration) Storage {
	s := &inMemoryStore{
		store:    sync.Map{},
		vacuumch: make(chan bool),
		ticker:   time.NewTicker(frequency),
	}
	go func(store *inMemoryStore) {
		for {
			select {
			case <-store.vacuumch:
				store.ticker.Stop()
				return
			case <-store.ticker.C:
				store.Vacuum(sessionTTL)
			}
		}
	}(s)
	return s
}
