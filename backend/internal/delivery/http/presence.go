package http

import (
	"encoding/json"
	"sync"
	"time"

	nethttp "net/http"
	api "sibur-petrochem-price-service/internal/generated/api"
)

// PresenceHub — SSE-поток числа аналитиков онлайн (одно соединение = один аналитик).
// Реализован вне strict-сервера: долгоживущему потоку нужен http.Flusher,
// которого strict-обвязка oapi-codegen не даёт (ответ буферизуется до конца хендлера).
type PresenceHub struct {
	mu   sync.Mutex
	subs map[chan int]struct{}
}

func NewPresenceHub() *PresenceHub {
	return &PresenceHub{subs: map[chan int]struct{}{}}
}

func (p *PresenceHub) subscribe() chan int {
	p.mu.Lock()
	defer p.mu.Unlock()

	stream := make(chan int, 1)
	p.subs[stream] = struct{}{}
	p.broadcastLocked()

	return stream
}

func (p *PresenceHub) unsubscribe(stream chan int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.subs, stream)
	p.broadcastLocked()
}

// broadcastLocked — рассылка текущего числа подключений; вызывается под mu.
// Канал размера 1: устаревшее значение вытесняется, важно только последнее.
func (p *PresenceHub) broadcastLocked() {
	online := len(p.subs)
	for stream := range p.subs {
		select {
		case <-stream:
		default:
		}
		select {
		case stream <- online:
		default:
		}
	}
}

func (p *PresenceHub) ServeHTTP(w nethttp.ResponseWriter, r *nethttp.Request) {
	const keepAliveEvery = 25 * time.Second

	flusher, ok := w.(nethttp.Flusher)
	if !ok {
		writeError(w, nethttp.StatusInternalServerError, "streaming_unsupported", "sse не поддерживается")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(nethttp.StatusOK)

	stream := p.subscribe()
	defer p.unsubscribe(stream)

	keepAlive := time.NewTicker(keepAliveEvery)
	defer keepAlive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case online := <-stream:
			payload, err := json.Marshal(api.PresenceEvent{AnalystsOnline: online})
			if err != nil {
				return
			}
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(payload)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		case <-keepAlive.C:
			// комментарий-пинг, чтобы прокси не закрывал простаивающее соединение
			_, _ = w.Write([]byte(": ping\n\n"))
			flusher.Flush()
		}
	}
}
