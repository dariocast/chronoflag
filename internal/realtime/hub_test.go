package realtime_test

import (
	"chronograph/internal/realtime"
	"testing"
	"time"
)

func TestHubPublishesWithoutBlockingOnSlowSubscriber(t *testing.T) {
	h := realtime.NewHub(1)
	_, cancelSlow := h.Subscribe("i")
	defer cancelSlow()
	fast, cancelFast := h.Subscribe("i")
	defer cancelFast()
	h.Publish("i", realtime.Message{ID: 1, Data: []byte(`{"v":1}`)})
	h.Publish("i", realtime.Message{ID: 2, Data: []byte(`{"v":2}`)})
	select {
	case msg := <-fast:
		if msg.ID != 1 && msg.ID != 2 {
			t.Fatalf("id=%d", msg.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("publish blocked")
	}
}
