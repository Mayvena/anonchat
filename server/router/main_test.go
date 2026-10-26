package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

func TestActiveProxyRegistry(t *testing.T) {
	t.Setenv("ACTIVE_PROXIES", `[{"id":"guard-1","address":"guard:9001","country_code":"DE","public_key_b64":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}]`)
	nodes, err := activeProxyNodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || nodes[0].CountryCode != "DE" {
		t.Fatalf("unexpected registry: %+v", nodes)
	}
}

func TestActiveProxyRegistryRejectsMalformedConfiguration(t *testing.T) {
	t.Setenv("ACTIVE_PROXIES", `{`)
	if _, err := activeProxyNodes(); err == nil {
		t.Fatal("expected malformed registry configuration to fail")
	}
}

func TestProxyNodeJSONShape(t *testing.T) {
	node := proxyNode{ID: "n1", Address: "node:9001", CountryCode: "DE", PublicKeyB64: base64.StdEncoding.EncodeToString(make([]byte, 32))}
	data, err := json.Marshal(node)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"public_key_b64"`) {
		t.Fatalf("unexpected JSON: %s", data)
	}
}

func newTestHub(t *testing.T) (*hub, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return newHub(rdb), mr
}

func newTestConn(t *testing.T) *websocket.Conn {
	t.Helper()
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func newTestClient(t *testing.T, h *hub, id string, buf int) *client {
	return &client{
		userID: id,
		conn:   newTestConn(t),
		send:   make(chan []byte, buf),
		done:   make(chan struct{}),
		hub:    h,
	}
}

func TestOfflineMessageStoredWithTTL(t *testing.T) {
	h, mr := newTestHub(t)
	h.routeMessage(context.Background(), chatMessage{ID: "m1", From: "a", To: "b", Body: "x"})

	key := offlineMessageKey("b", "m1")
	if !mr.Exists(key) {
		t.Fatal("expected offline message to be stored")
	}
	if ttl := mr.TTL(key); ttl != offlineTTL {
		t.Fatalf("expected ttl %v, got %v", offlineTTL, ttl)
	}
}

func TestOnlineMessageDeliveredAndNotStored(t *testing.T) {
	h, mr := newTestHub(t)
	c := newTestClient(t, h, "b", 4)
	h.register(c)

	h.routeMessage(context.Background(), chatMessage{ID: "m1", From: "a", To: "b", Body: "x"})

	select {
	case <-c.send:
	default:
		t.Fatal("expected message on client channel")
	}
	if mr.Exists(offlineMessageKey("b", "m1")) {
		t.Fatal("online delivery must not leave a key in redis")
	}
}

func TestOfflineReplayDeletesKey(t *testing.T) {
	h, mr := newTestHub(t)
	h.routeMessage(context.Background(), chatMessage{ID: "m1", From: "a", To: "b", Body: "x"})

	c := newTestClient(t, h, "b", 4)
	h.register(c)
	h.deliverOffline(context.Background(), "b")

	if len(c.send) != 1 {
		t.Fatalf("expected 1 replayed message, got %d", len(c.send))
	}
	if mr.Exists(offlineMessageKey("b", "m1")) {
		t.Fatal("replayed message key must be deleted")
	}
}

func TestSlowConsumerDoesNotPanic(t *testing.T) {
	h, _ := newTestHub(t)
	c := newTestClient(t, h, "b", 1)
	h.register(c)

	if !h.deliver("b", []byte("1")) {
		t.Fatal("first delivery should succeed")
	}
	if h.deliver("b", []byte("2")) {
		t.Fatal("second delivery should fail on full buffer")
	}
	// Repeated shutdown and delivery after close must be safe.
	c.shutdown()
	c.shutdown()
	if h.deliver("b", []byte("3")) {
		t.Fatal("delivery to removed client must fail")
	}
	select {
	case <-c.done:
	case <-time.After(time.Second):
		t.Fatal("done channel not closed")
	}
}
