// Package pipeline provides a flexible framework for composing and executing
// sequences of operations (Steps) for LLM workflows like RAG and data processing.
package pipeline

import (
	"sync"
)

// State is a thread-safe container for data passed between pipeline steps.
type State struct {
	data  map[string]interface{}
	mutex sync.RWMutex
}

// NewState creates a new, empty pipeline state.
func NewState() *State {
	return &State{
		data: make(map[string]interface{}),
	}
}

// Set stores a value in the state.
func (s *State) Set(key string, value interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.data[key] = value
}

// Get retrieves a value from the state. Returns nil and false if the key does not exist.
func (s *State) Get(key string) (interface{}, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

// GetString is a helper to retrieve a string value. Returns empty string if missing or wrong type.
func (s *State) GetString(key string) string {
	val, ok := s.Get(key)
	if !ok {
		return ""
	}
	str, ok := val.(string)
	if !ok {
		return ""
	}
	return str
}

// Delete removes a key from the state.
func (s *State) Delete(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.data, key)
}

// Clone creates a shallow copy of the current state.
func (s *State) Clone() *State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	newData := make(map[string]interface{}, len(s.data))
	for k, v := range s.data {
		newData[k] = v
	}

	return &State{
		data: newData,
	}
}
