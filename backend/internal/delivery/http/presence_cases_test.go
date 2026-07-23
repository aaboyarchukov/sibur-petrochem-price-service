package http

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	nethttp "net/http"
)

// presenceClient — открытое SSE-соединение presence-потока.
type presenceClient struct {
	cancel context.CancelFunc
	events chan int
}

// connectPresence — подключение к /presence/events; события читаются в фоне.
func connectPresence(t *testing.T, baseURL string) presenceClient {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	request, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodGet, baseURL+"/api/v1/presence/events", nil)
	require.NoError(t, err)

	response, err := nethttp.DefaultClient.Do(request)
	require.NoError(t, err)
	require.Equal(t, nethttp.StatusOK, response.StatusCode)
	require.Equal(t, "text/event-stream", response.Header.Get("Content-Type"))

	events := make(chan int, 1)
	go func() {
		defer response.Body.Close()
		defer close(events)
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var event struct {
				AnalystsOnline int `json:"analysts_online"`
			}
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event) == nil {
				events <- event.AnalystsOnline
			}
		}
	}()

	t.Cleanup(cancel)

	return presenceClient{cancel: cancel, events: events}
}

// waitCount — дождаться события с ожидаемым значением (события с size-1 канала
// могут схлопываться, поэтому ждём именно целевое число).
func waitCount(t *testing.T, client presenceClient, want int) {
	t.Helper()

	const timeout = 3 * time.Second
	deadline := time.After(timeout)
	for {
		select {
		case got, ok := <-client.events:
			require.True(t, ok, "поток закрылся до ожидаемого события")
			if got == want {
				return
			}
		case <-deadline:
			t.Fatalf("не дождались analysts_online=%d", want)
		}
	}
}

func TestPresenceCountsConnections(t *testing.T) {
	tc := newTestContext(t)

	server := httptest.NewServer(tc.SUT)
	t.Cleanup(server.Close)

	first := connectPresence(t, server.URL)
	waitCount(t, first, 1)

	second := connectPresence(t, server.URL)
	waitCount(t, second, 2)
	waitCount(t, first, 2)

	// отключение второго — первый видит уменьшение
	second.cancel()
	waitCount(t, first, 1)

	assert.NotNil(t, first.events)
}
