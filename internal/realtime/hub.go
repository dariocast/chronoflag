package realtime

import "sync"

type Message struct {
	ID    int64
	Event string
	Data  []byte
}
type Hub struct {
	mu          sync.Mutex
	buffer      int
	subscribers map[string]map[chan Message]struct{}
}

func NewHub(buffer int) *Hub {
	if buffer < 1 {
		buffer = 1
	}
	return &Hub{buffer: buffer, subscribers: map[string]map[chan Message]struct{}{}}
}
func (h *Hub) Subscribe(instance string) (<-chan Message, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan Message, h.buffer)
	if h.subscribers[instance] == nil {
		h.subscribers[instance] = map[chan Message]struct{}{}
	}
	h.subscribers[instance][ch] = struct{}{}
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			if _, ok := h.subscribers[instance][ch]; ok {
				delete(h.subscribers[instance], ch)
				close(ch)
			}
		})
	}
}
func (h *Hub) Publish(instance string, msg Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers[instance] {
		select {
		case ch <- msg:
		default:
			delete(h.subscribers[instance], ch)
			close(ch)
		}
	}
}
