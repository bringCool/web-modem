package handler

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rehiy/web-modem/service"
)

type SseHandler struct {
	broker *sseBroker
}

type sseBroker struct {
	source  <-chan string
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

var modemEventBroker = newSseBroker(service.ModemEvent)

func init() {
	go modemEventBroker.run()
}

func NewSseHandler() *SseHandler {
	return &SseHandler{
		broker: modemEventBroker,
	}
}

func newSseBroker(source <-chan string) *sseBroker {
	return &sseBroker{
		source:  source,
		clients: make(map[chan string]struct{}),
	}
}

func (b *sseBroker) run() {
	for event := range b.source {
		b.broadcast(event)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	for client := range b.clients {
		close(client)
		delete(b.clients, client)
	}
}

func (b *sseBroker) subscribe() (chan string, func()) {
	client := make(chan string, 16)

	b.mu.Lock()
	b.clients[client] = struct{}{}
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.clients[client]; ok {
			delete(b.clients, client)
			close(client)
		}
	}

	return client, unsubscribe
}

func (b *sseBroker) broadcast(event string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for client := range b.clients {
		select {
		case client <- event:
		default:
			log.Printf("SSE client event queue full, dropping modem event: %s", event)
		}
	}
}

func (h *SseHandler) HandleModemEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		respondJSON(w, http.StatusInternalServerError, H{"error": "streaming unsupported"})
		return
	}

	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")

	client, unsubscribe := h.broker.subscribe()
	defer unsubscribe()

	log.Printf("SSE client connected: %s", r.RemoteAddr)
	defer log.Printf("SSE client disconnected: %s", r.RemoteAddr)

	fmt.Fprint(w, "retry: 5000\n\n")
	flusher.Flush()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case event, ok := <-client:
			if !ok {
				return
			}
			writeSseEvent(w, "message", event)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func writeSseEvent(w http.ResponseWriter, eventName, data string) {
	if eventName != "" {
		fmt.Fprintf(w, "event: %s\n", eventName)
	}
	for _, line := range strings.Split(data, "\n") {
		fmt.Fprintf(w, "data: %s\n", line)
	}
	fmt.Fprint(w, "\n")
}
