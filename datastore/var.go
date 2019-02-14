package datastore

import (
	"sync"
	"time"
)

// Var is a cache for a single variable that is considered invalid if not updated
// within the last specified duration. All operations on it are thread-safe.
type Var struct {
	mux     sync.RWMutex
	expires time.Duration
	updated time.Time
	value   interface{}
}

// NewVar creates a new Var with the specified expiration period.
func NewVar(expires time.Duration) Var {
	return Var{expires: expires}
}

// Get returns the value if still valid, otherwise returns nil.
func (v *Var) Get() interface{} {
	v.mux.RLock()
	defer v.mux.RUnlock()
	if time.Since(v.updated) < v.expires {
		return v.value
	}
	v.value = nil
	return nil
}

// Set updates the value
func (v *Var) Set(val interface{}) {
	v.mux.Lock()
	defer v.mux.Unlock()
	v.value = val
	v.updated = time.Now()
}

// Update will return the value if still valid, otherwise it will call updater
// to update the value then return it.
func (v *Var) Update(updater func() (interface{}, error)) (interface{}, error) {
	v.mux.Lock()
	defer v.mux.Unlock()
	if time.Since(v.updated) < v.expires {
		return v.value, nil
	}
	val, err := updater()
	if err != nil {
		return nil, err
	}
	v.value = val
	v.updated = time.Now()
	return v.value, nil
}
