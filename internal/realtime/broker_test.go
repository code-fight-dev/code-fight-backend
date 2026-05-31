package realtime

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBrokerPublishDeliversEventToUserSubscribers(t *testing.T) {
	broker := NewBroker()
	userID := uuid.New()
	events, unsubscribe := broker.Subscribe(userID)
	defer unsubscribe()

	broker.Publish(userID, "match.started", map[string]string{"matchId": "match-1"})

	event := receiveEvent(t, events)
	if event.ID == "" {
		t.Fatal("event id is empty")
	}
	if event.Type != "match.started" {
		t.Fatalf("event type = %q, want match.started", event.Type)
	}

	var payload map[string]string
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		t.Fatalf("unmarshal event data: %v", err)
	}
	if payload["matchId"] != "match-1" {
		t.Fatalf("matchId = %q, want match-1", payload["matchId"])
	}
}

func TestBrokerPublishDoesNotDeliverToOtherUsers(t *testing.T) {
	broker := NewBroker()
	userID := uuid.New()
	otherUserID := uuid.New()
	events, unsubscribe := broker.Subscribe(userID)
	defer unsubscribe()

	broker.Publish(otherUserID, "match.started", map[string]string{"matchId": "match-1"})

	assertNoEvent(t, events)
}

func TestBrokerUnsubscribeClosesChannelAndStopsDelivery(t *testing.T) {
	broker := NewBroker()
	userID := uuid.New()
	events, unsubscribe := broker.Subscribe(userID)

	unsubscribe()

	if _, ok := <-events; ok {
		t.Fatal("events channel is open, want closed")
	}

	broker.Publish(userID, "match.started", map[string]string{"matchId": "match-1"})
}

func TestBrokerPublishManyDeduplicatesRecipients(t *testing.T) {
	broker := NewBroker()
	userID := uuid.New()
	events, unsubscribe := broker.Subscribe(userID)
	defer unsubscribe()

	broker.PublishMany([]uuid.UUID{userID, userID}, "match.updated", map[string]int{"score": 10})

	receiveEvent(t, events)
	assertNoEvent(t, events)
}

func TestBrokerPublishIgnoresInvalidEvents(t *testing.T) {
	broker := NewBroker()
	userID := uuid.New()
	events, unsubscribe := broker.Subscribe(userID)
	defer unsubscribe()

	broker.Publish(uuid.Nil, "match.started", map[string]string{"matchId": "match-1"})
	broker.Publish(userID, "", map[string]string{"matchId": "match-1"})
	broker.Publish(userID, "match.started", func() {})

	assertNoEvent(t, events)
}

func receiveEvent(t *testing.T, events <-chan Event) Event {
	t.Helper()

	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("events channel closed")
		}
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	return Event{}
}

func assertNoEvent(t *testing.T, events <-chan Event) {
	t.Helper()

	select {
	case event, ok := <-events:
		if !ok {
			return
		}
		t.Fatalf("received unexpected event: %+v", event)
	case <-time.After(25 * time.Millisecond):
		return
	}
}
