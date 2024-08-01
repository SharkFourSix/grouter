package at

import (
	"fmt"
	"strings"
	"time"

	"github.com/SharkFourSix/grouter"
	cmap "github.com/orcaman/concurrent-map"
)

type africasTalkingUssdSession struct {
	startTime   time.Time
	readPointer int
	store       cmap.ConcurrentMap
	state       grouter.RouterState
	option      string
	input       string
	id          string
}

func (s *africasTalkingUssdSession) Read(request *requestData) {
	value := strings.Clone(request.Text[s.readPointer:])
	s.readPointer = len(request.Text) + 1 // adjust pointer (account for asteriks)
	switch s.state {
	case grouter.READ_INPUT: // previous option is kept
		s.input = value
	case grouter.READ_OPTION:
		s.input = ""
		s.option = value
	}
}

func (s *africasTalkingUssdSession) ID() string {
	return s.id
}

func (s *africasTalkingUssdSession) CreatedAt() time.Time {
	return s.startTime
}

func (s *africasTalkingUssdSession) Set(key string, value any) {
	s.store.Set(key, value)
}

func (s *africasTalkingUssdSession) Del(key string) {
	s.store.Remove(key)
}

func (s *africasTalkingUssdSession) Get(key string) (any, bool) {
	return s.store.Get(key)
}

func (s *africasTalkingUssdSession) MustGet(key string) any {
	if value, ok := s.store.Get(key); ok {
		return value
	}
	panic(fmt.Errorf("%s: not found", key))
}
