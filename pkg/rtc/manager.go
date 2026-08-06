package rtc

import "sync"

// Manager keeps a reference to the live WebRTC bridge for each active call so
// API handlers can find (and tear down) a call's remote peer by call id.
type Manager struct {
	mu      sync.Mutex
	bridges map[string]*Bridge
}

// NewManager returns an empty manager.
func NewManager() *Manager {
	return &Manager{bridges: map[string]*Bridge{}}
}

// NewCall creates (and tracks) a fresh bridge for the call.
func (m *Manager) NewCall(callID string) (*Bridge, error) {
	b, err := NewBridge(callID)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.bridges[callID] = b
	m.mu.Unlock()
	return b, nil
}

// Get returns the bridge for the call, if any.
func (m *Manager) Get(callID string) (*Bridge, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.bridges[callID]
	return b, ok
}

// End closes and forgets the bridge for the call (no-op if it doesn't exist).
func (m *Manager) End(callID string) error {
	m.mu.Lock()
	b, ok := m.bridges[callID]
	delete(m.bridges, callID)
	m.mu.Unlock()
	if !ok {
		return nil
	}
	return b.Close()
}
