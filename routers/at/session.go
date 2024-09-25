package at

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SharkFourSix/grouter"
	cmap "github.com/orcaman/concurrent-map"
)

type africasTalkingUssdSession struct {
	startTime             time.Time
	readPointer           int
	store                 cmap.ConcurrentMap
	state                 grouter.RouterState
	option                string
	input                 string
	id                    string
	autoAdjustReadPointer bool
}

func (s *africasTalkingUssdSession) Read(request *requestData) {
	var (
		value       string
		inputLength = len(request.Text)
		pointer     = s.readPointer
	)
	if s.autoAdjustReadPointer {
		pointer = min(pointer, inputLength)
	}
	value = strings.Clone(request.Text[pointer:])
	s.readPointer = inputLength + 1 // adjust pointer (account for asteriks)
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

// SetAutoAdjustReadPointer Sets the read pointer to auto adjust when using
// manual route handling using grouter.UssdRequest.Prompt() functions, which
// can cause index out of bounds error.
//
// The function will panic if the request is not provided by this module.
func SetAutoAdjustReadPointer(request grouter.UssdRequest, autoAdjust bool) {
	atRequest, ok := request.(*ussd_request)
	if !ok {
		panic(errors.New("expected AT ussd request instance"))
	}
	atRequest.sess.autoAdjustReadPointer = autoAdjust
}

// IsReadPointerAutoAdjusted Gets whether the read pointer is set to auto 
// adjust when using manual route handling using grouter.UssdRequest.Prompt() 
// functions.
//
// The function will panic if the request is not provided by this module.
func IsReadPointerAutoAdjusted(request grouter.UssdRequest) bool {
	atRequest, ok := request.(*ussd_request)
	if !ok {
		panic(errors.New("expected AT ussd request instance"))
	}
	return atRequest.sess.autoAdjustReadPointer
}
