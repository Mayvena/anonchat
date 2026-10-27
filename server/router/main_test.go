package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
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
		send:   make(chan outboundMessage, buf),
		done:   make(chan struct{}),
		hub:    h,
	}
}

func dialSession(t *testing.T, serverURL, query string, protocols ...string) (*websocket.Conn, map[string]string) {
	t.Helper()
	endpoint := "ws" + strings.TrimPrefix(serverURL, "http")
	if query != "" {
		endpoint += "?" + query
	}
	dialer := *websocket.DefaultDialer
	dialer.Subprotocols = protocols
	conn, _, err := dialer.Dial(endpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Type    string            `json:"type"`
		Payload map[string]string `json:"payload"`
	}
	if err := conn.ReadJSON(&envelope); err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}
	if envelope.Type != "session" {
		_ = conn.Close()
		t.Fatalf("expected session acknowledgment, got %q", envelope.Type)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn, envelope.Payload
}

func TestWebsocketSessionIdentityAndCredentialOwnership(t *testing.T) {
	h, _ := newTestHub(t)
	server := httptest.NewServer(websocketHandler(h))
	t.Cleanup(server.Close)

	requestedID := uuid.NewString()
	_, first := dialSession(t, server.URL, "user_id="+url.QueryEscape(requestedID))
	firstID, firstCredential := first["user_id"], first["credential"]
	if _, err := uuid.Parse(firstID); err != nil || firstID != strings.ToLower(firstID) {
		t.Fatalf("server did not issue a UUID identity: %q", firstID)
	}
	if firstID == requestedID {
		t.Fatal("server accepted a caller-selected identity")
	}
	firstClient := h.getClient(firstID)
	if firstClient == nil {
		t.Fatal("new session was not registered")
	}
	if len(firstCredential) != 43 {
		t.Fatalf("expected a 256-bit URL-safe credential, got length %d", len(firstCredential))
	}

	_, unauthenticated := dialSession(t, server.URL, "user_id="+url.QueryEscape(firstID))
	if unauthenticated["user_id"] == firstID {
		t.Fatal("missing credential reclaimed an existing identity")
	}
	if h.getClient(firstID) != firstClient {
		t.Fatal("missing credential replaced the existing session connection")
	}

	badProtocol := sessionAuthPrefix + firstID + "." + strings.Repeat("A", 43)
	endpoint := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := *websocket.DefaultDialer
	dialer.Subprotocols = []string{badProtocol}
	if conn, response, err := dialer.Dial(endpoint, nil); err == nil {
		_ = conn.Close()
		t.Fatal("wrong credential unexpectedly connected")
	} else if response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized response for wrong credential, got response=%v err=%v", response, err)
	}

	validProtocol := sessionAuthPrefix + firstID + "." + firstCredential
	_, reconnected := dialSession(t, server.URL, "", validProtocol)
	if reconnected["user_id"] != firstID || reconnected["credential"] != firstCredential {
		t.Fatal("valid credential did not reconnect to the original session")
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

	var message outboundMessage
	select {
	case message = <-c.send:
	default:
		t.Fatal("expected message on client channel")
	}
	key := offlineMessageKey("b", "m1")
	if !mr.Exists(key) {
		t.Fatal("online message must remain buffered until the socket write succeeds")
	}
	if err := c.writeMessage(message); err != nil {
		t.Fatalf("expected socket write to succeed: %v", err)
	}
	if mr.Exists(key) {
		t.Fatal("successfully written message must be removed from redis")
	}
}

func TestRoomMessageBroadcastsToConnectedPeersWithoutOfflineStorage(t *testing.T) {
	h, mr := newTestHub(t)
	sender := newTestClient(t, h, "sender", 4)
	firstPeer := newTestClient(t, h, "first-peer", 4)
	secondPeer := newTestClient(t, h, "second-peer", 4)
	h.register(sender)
	h.register(firstPeer)
	h.register(secondPeer)

	h.routeMessage(context.Background(), chatMessage{
		ID:   "room-message",
		From: sender.userID,
		To:   roomRecipientID,
		Body: "room payload",
	})

	for _, peer := range []*client{firstPeer, secondPeer} {
		select {
		case message := <-peer.send:
			if message.offlineKey != "" {
				t.Fatal("room message must not use offline acknowledgement storage")
			}
			var envelope struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(message.payload, &envelope); err != nil || envelope.Type != "message" {
				t.Fatalf("expected room message envelope, got %s (err=%v)", message.payload, err)
			}
		default:
			t.Fatalf("expected room message for connected peer %q", peer.userID)
		}
	}
	if len(sender.send) != 0 {
		t.Fatal("sender must not receive a duplicate room message")
	}
	if mr.Exists(offlineMessageKey(roomRecipientID, "room-message")) {
		t.Fatal("room message must not be stored for offline delivery")
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
	key := offlineMessageKey("b", "m1")
	if !mr.Exists(key) {
		t.Fatal("replayed message must remain buffered until the socket write succeeds")
	}
	if err := c.writeMessage(<-c.send); err != nil {
		t.Fatalf("expected replay socket write to succeed: %v", err)
	}
	if mr.Exists(key) {
		t.Fatal("successfully written replay must be removed from redis")
	}
}

func TestFailedSocketWriteRetainsOfflineMessage(t *testing.T) {
	h, mr := newTestHub(t)
	c := newTestClient(t, h, "b", 4)
	h.register(c)
	h.routeMessage(context.Background(), chatMessage{ID: "m1", From: "a", To: "b", Body: "x"})

	key := offlineMessageKey("b", "m1")
	message := <-c.send
	_ = c.conn.Close()
	if err := c.writeMessage(message); err == nil {
		t.Fatal("expected socket write to fail after connection close")
	}
	if !mr.Exists(key) {
		t.Fatal("failed socket write must retain the offline message")
	}
}

func TestSlowConsumerDoesNotPanic(t *testing.T) {
	h, _ := newTestHub(t)
	c := newTestClient(t, h, "b", 1)
	h.register(c)

	if !h.deliver("b", outboundMessage{payload: []byte("1")}) {
		t.Fatal("first delivery should succeed")
	}
	if h.deliver("b", outboundMessage{payload: []byte("2")}) {
		t.Fatal("second delivery should fail on full buffer")
	}
	// Repeated shutdown and delivery after close must be safe.
	c.shutdown()
	c.shutdown()
	if h.deliver("b", outboundMessage{payload: []byte("3")}) {
		t.Fatal("delivery to removed client must fail")
	}
	select {
	case <-c.done:
	case <-time.After(time.Second):
		t.Fatal("done channel not closed")
	}
}

func TestRefreshProxyRegistryStoresBuiltSet(t *testing.T) {
	_, mr := newTestHub(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("127.0.0.1:" + strconv.Itoa(ln.Addr().(*net.TCPAddr).Port) + "\n"))
	}))
	defer proxySrv.Close()

	t.Setenv("PROXY_SOURCES", proxySrv.URL)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	if err := refreshProxyRegistry(context.Background(), rdb); err != nil {
		t.Fatalf("refreshProxyRegistry returned error: %v", err)
	}

	stored, err := rdb.Get(context.Background(), proxyRegistryRedisKey).Result()
	if err != nil {
		t.Fatalf("expected registry value in Redis: %v", err)
	}
	if !strings.Contains(stored, "127.0.0.1:") || !strings.Contains(stored, "public_key_b64") {
		t.Fatalf("unexpected registry payload: %s", stored)
	}
}
