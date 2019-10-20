package plugin

import "sync"

// EntryMap is a thread-safe map of <entry_cname> => <entry_object>.
// It's API is (mostly) symmetric with sync.Map.
type EntryMap struct {
	mp  map[string]Entry
	mux sync.RWMutex
}

func newEntryMap() *EntryMap {
	return &EntryMap{
		mp: make(map[string]Entry),
	}
}

// Load retrieves an entry
func (m *EntryMap) Load(cname string) (Entry, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	entry, ok := m.mp[cname]
	return entry, ok
}

// Delete deletes the entry from the map
func (m *EntryMap) Delete(cname string) {
	m.mux.Lock()
	defer m.mux.Unlock()

	delete(m.mp, cname)
}

// Len returns the number of entries in the map
func (m *EntryMap) Len() int {
	m.mux.RLock()
	defer m.mux.RUnlock()

	return len(m.mp)
}

// Range iterates over the map, applying f to each (cname, entry)
// pair. If f returns false, then Each will break out of the loop.
func (m *EntryMap) Range(f func(string, Entry) bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	for cname, entry := range m.mp {
		if !f(cname, entry) {
			break
		}
	}
}

// Map returns m's underlying map. It can only be called by the tests.
func (m *EntryMap) Map() map[string]Entry {
	if notRunningTests() {
		panic("plugin.EntryMap#Map can only be called by the tests")
	}
	return m.mp
}
