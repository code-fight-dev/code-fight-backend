package realtime

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

type Event struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type Broker struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]map[chan Event]struct{}
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[uuid.UUID]map[chan Event]struct{}),
	}
}

func (b *Broker) Subscribe(userID uuid.UUID) (<-chan Event, func()) {
	ch := make(chan Event, 32)

	b.mu.Lock()
	if b.subscribers[userID] == nil {
		b.subscribers[userID] = make(map[chan Event]struct{})
	}
	b.subscribers[userID][ch] = struct{}{}
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		if subs := b.subscribers[userID]; subs != nil {
			delete(subs, ch)
			if len(subs) == 0 {
				delete(b.subscribers, userID)
			}
		}
		close(ch)
	}

	return ch, unsubscribe
}

func (b *Broker) Publish(userID uuid.UUID, eventType string, payload any) {
	if userID == uuid.Nil || eventType == "" {
		return
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	event := Event{
		ID:   uuid.NewString(),
		Type: eventType,
		Data: data,
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers[userID] {
		select {
		case ch <- event:
		default:
		}
	}
}

func (b *Broker) PublishMany(userIDs []uuid.UUID, eventType string, payload any) {
	seen := make(map[uuid.UUID]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		b.Publish(userID, eventType, payload)
	}
}
