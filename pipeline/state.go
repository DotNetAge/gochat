package pipeline

import (
	"sync"
)

// State provides a thread-safe key-value store for sharing data between
// pipeline steps. It supports type-safe retrieval through generic getters.
//
// State is safe for concurrent use from multiple goroutines.
// All read and write operations are protected by a RWMutex.
//
// Example:
//
//	state := pipeline.NewState()
//	state.Set("user_id", 12345)
//	state.Set("name", "Alice")
//
//	// Type-safe retrieval
//	if id, ok := state.GetInt("user_id"); ok {
//	    fmt.Println(id) // 12345
//	}
type State struct {
	data  map[string]interface{}
	mutex sync.RWMutex
}

// NewState creates a new, empty State for use in a pipeline execution.
func NewState() *State {
	return &State{
		data: make(map[string]interface{}),
	}
}

// Set stores a value in the state with the given key.
// Any type can be stored; use the type-safe getters to retrieve values.
//
// Parameters:
//   - key: The identifier for this value
//   - value: The value to store (can be any type)
func (s *State) Set(key string, value interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.data[key] = value
}

// Get retrieves a raw value by key. Returns (value, true) if found,
// or (nil, false) if the key doesn't exist.
//
// For type-safe retrieval, use one of the typed getters:
// GetString, GetInt, GetFloat, GetBool, GetStringSlice, GetMap.
//
// Parameters:
//   - key: The identifier to look up
//
// Returns the value and whether it was found
func (s *State) Get(key string) (interface{}, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

// GetString retrieves a string value by key.
// Returns (value, true) if the key exists and holds a string,
// or ("", false) otherwise.
//
// Parameters:
//   - key: The identifier to look up
//
// Returns the string value and whether retrieval succeeded
func (s *State) GetString(key string) (string, bool) {
	val, ok := s.Get(key)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetInt retrieves an int value by key.
// Returns (value, true) if the key exists and holds an int,
// or (0, false) otherwise.
//
// Parameters:
//   - key: The identifier to look up
//
// Returns the int value and whether retrieval succeeded
func (s *State) GetInt(key string) (int, bool) {
	val, ok := s.Get(key)
	if !ok {
		return 0, false
	}
	i, ok := val.(int)
	return i, ok
}

// GetFloat retrieves a float64 value by key.
// Returns (value, true) if the key exists and holds a float64,
// or (0.0, false) otherwise.
//
// Parameters:
//   - key: The identifier to look up
//
// Returns the float64 value and whether retrieval succeeded
func (s *State) GetFloat(key string) (float64, bool) {
	val, ok := s.Get(key)
	if !ok {
		return 0, false
	}
	f, ok := val.(float64)
	return f, ok
}

// GetBool retrieves a bool value by key.
// Returns (value, true) if the key exists and holds a bool,
// or (false, false) otherwise.
//
// Parameters:
//   - key: The identifier to look up
//
// Returns the bool value and whether retrieval succeeded
func (s *State) GetBool(key string) (bool, bool) {
	val, ok := s.Get(key)
	if !ok {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

// GetStringSlice retrieves a []string value by key.
// Returns (value, true) if the key exists and holds a []string,
// or (nil, false) otherwise.
//
// Parameters:
//   - key: The identifier to look up
//
// Returns the []string value and whether retrieval succeeded
func (s *State) GetStringSlice(key string) ([]string, bool) {
	val, ok := s.Get(key)
	if !ok {
		return nil, false
	}
	slice, ok := val.([]string)
	return slice, ok
}

// GetMap retrieves a map[string]interface{} value by key.
// Returns (value, true) if the key exists and holds a map,
// or (nil, false) otherwise.
//
// Parameters:
//   - key: The identifier to look up
//
// Returns the map value and whether retrieval succeeded
func (s *State) GetMap(key string) (map[string]interface{}, bool) {
	val, ok := s.Get(key)
	if !ok {
		return nil, false
	}
	m, ok := val.(map[string]interface{})
	return m, ok
}

// Delete removes a key-value pair from the state.
//
// Parameters:
//   - key: The identifier to remove
func (s *State) Delete(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.data, key)
}

// Clone creates a shallow copy of the state.
// The returned State has a copy of the data map but the values
// themselves are shared references.
//
// Returns a new State with the same data
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
