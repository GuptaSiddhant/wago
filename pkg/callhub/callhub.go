// Package callhub fans out live call events to the browser dashboard.
//
// A call can be triggered in different places (an inbound webhook, an agent
// starting an outbound call, a worker) while the SSE endpoint that pushes the
// event to a browser lives in another package. callhub is a tiny in-memory
// publish/subscribe bus so any package can broadcast a call update and the
// dashboard can subscribe by org.
package callhub

import (
	"sync"
)

// Event types broadcast by the hub.
const (
	// EventIncoming is published when an inbound call starts ringing. The
	// dashboard reacts by showing the "answer" banner.
	EventIncoming = "incoming"
	// EventUpdate is published for state changes of an already-known call
	// (ringing -> active -> ended).
	EventUpdate = "update"
)

// CallEvent is the payload pushed to a subscriber. CallID is the PocketBase
// voice_calls record id; Direction/State let the UI render the banner.
type CallEvent struct {
	ID        string `json:"id"`
	Direction string `json:"direction"`
	State     string `json:"state"`
	CallerID  string `json:"caller_id,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Name      string `json:"name,omitempty"`
}

// DefaultHub is the process-wide call event bus. Both the HTTP API and the
// inbound webhook handlers publish to it, and SSE subscriptions read from it.
var DefaultHub = NewHub()

// Hub fans out call events to subscribers, keyed by org id so orgs never see
// each other's calls.
type Hub struct {
	mu   sync.Mutex
	subs map[string]map[chan CallEvent]struct{}
}

// NewHub creates an empty hub. The process keeps a single shared hub.
func NewHub() *Hub {
	return &Hub{subs: map[string]map[chan CallEvent]struct{}{}}
}

// Subscribe returns a channel that receives every call event for the org. The
// caller must drain it and call Unsubscribe when done.
func (h *Hub) Subscribe(orgID string) chan CallEvent {
	ch := make(chan CallEvent, 16)
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.subs[orgID]; set != nil {
		set[ch] = struct{}{}
	} else {
		h.subs[orgID] = map[chan CallEvent]struct{}{ch: {}}
	}
	return ch
}

// Unsubscribe stops deliveries to the channel.
func (h *Hub) Unsubscribe(orgID string, ch chan CallEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.subs[orgID]; set != nil {
		delete(set, ch)
	}
}

// Publish sends an event to every subscriber of the org. It never blocks:
// slow or full channels are dropped rather than stalling the sender.
func (h *Hub) Publish(orgID string, ev CallEvent) {
	h.mu.Lock()
	set := h.subs[orgID]
	setCopy := make([]chan CallEvent, 0, len(set))
	for ch := range set {
		setCopy = append(setCopy, ch)
	}
	h.mu.Unlock()

	for _, ch := range setCopy {
		select {
		case ch <- ev:
		default:
			// Subscriber is not draining; skip so a lagging reader can't
			// block a call being started.
		}
	}
}
