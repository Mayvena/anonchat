package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	offlineTTL       = 300 * time.Second
	heartbeatEvery   = 10 * time.Second
	pongWait         = 20 * time.Second
	writeWait        = 10 * time.Second
	maxIncomingBytes = 64 * 1024
)

type inboundMessage struct {
	ID   string `json:"id"`
	To   string `json:"to"`
	Body string `json:"body"`
}

type chatMessage struct {
	ID     string `json:"id"`
	From   string `json:"from"`
	To     string `json:"to"`
	Body   string `json:"body"`
	SentAt int64  `json:"sent_at"`
}

type outboundEnvelope struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type proxyNode struct {
	ID           string `json:"id"`
	Address      string `json:"address"`
	CountryCode  string `json:"country_code"`
	PublicKeyB64 string `json:"public_key_b64"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
}

type client struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
	done   chan struct{}
	once   sync.Once
	hub    *hub
}

// shutdown is idempotent; send is never closed, so concurrent senders cannot panic.
func (c *client) shutdown() {
	c.once.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

type hub struct {
	redis   *redis.Client
	mu      sync.RWMutex
	clients map[string]*client
}

func newHub(rdb *redis.Client) *hub {
	return &hub{
		redis:   rdb,
		clients: make(map[string]*client),
	}
}

func (h *hub) register(c *client) {
	h.mu.Lock()
	h.clients[c.userID] = c
	h.mu.Unlock()
}

func (h *hub) unregister(c *client) {
	h.mu.Lock()
	if curr, ok := h.clients[c.userID]; ok && curr == c {
		delete(h.clients, c.userID)
	}
	h.mu.Unlock()
}

func (h *hub) getClient(userID string) *client {
	h.mu.RLock()
	c := h.clients[userID]
	h.mu.RUnlock()
	return c
}

func (h *hub) deliver(userID string, payload []byte) bool {
	c := h.getClient(userID)
	if c == nil {
		return false
	}

	select {
	case <-c.done:
		return false
	default:
	}

	select {
	case c.send <- payload:
		return true
	default:
		h.unregister(c)
		c.shutdown()
		return false
	}
}

func (h *hub) routeMessage(ctx context.Context, m chatMessage) {
	encoded, err := json.Marshal(outboundEnvelope{Type: "message", Payload: m})
	if err != nil {
		return
	}

	key := offlineMessageKey(m.To, m.ID)
	if h.deliver(m.To, encoded) {
		_, _ = h.redis.Del(ctx, key).Result()
		return
	}

	stored, err := json.Marshal(m)
	if err != nil {
		return
	}
	_ = h.redis.Set(ctx, key, stored, offlineTTL).Err()
}

func (h *hub) deliverOffline(ctx context.Context, userID string) {
	pattern := "offline:" + userID + ":*"
	keys, err := h.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return
	}

	for _, key := range keys {
		data, err := h.redis.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var msg chatMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			_, _ = h.redis.Del(ctx, key).Result()
			continue
		}

		encoded, err := json.Marshal(outboundEnvelope{Type: "message", Payload: msg})
		if err != nil {
			continue
		}
		if h.deliver(userID, encoded) {
			_, _ = h.redis.Del(ctx, key).Result()
		}
	}
}

func (c *client) readPump() {
	defer func() {
		c.hub.unregister(c)
		c.shutdown()
	}()

	c.conn.SetReadLimit(maxIncomingBytes)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	ctx := context.Background()
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var inbound inboundMessage
		if err := json.Unmarshal(data, &inbound); err != nil {
			continue
		}
		inbound.To = strings.TrimSpace(inbound.To)
		if inbound.To == "" {
			continue
		}
		if inbound.ID == "" {
			inbound.ID = uuid.NewString()
		}

		msg := chatMessage{
			ID:     inbound.ID,
			From:   c.userID,
			To:     inbound.To,
			Body:   inbound.Body,
			SentAt: time.Now().Unix(),
		}
		c.hub.routeMessage(ctx, msg)
	}
}

func (c *client) writePump() {
	ticker := time.NewTicker(heartbeatEvery)
	defer func() {
		ticker.Stop()
		c.shutdown()
	}()

	for {
		select {
		case <-c.done:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		case payload := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func offlineMessageKey(toUser, msgID string) string {
	return "offline:" + toUser + ":" + msgID
}

func getenv(name, fallback string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	return v
}

func activeProxyNodes() ([]proxyNode, error) {
	raw := strings.TrimSpace(os.Getenv("ACTIVE_PROXIES"))
	if raw == "" {
		return []proxyNode{}, nil
	}

	var nodes []proxyNode
	if err := json.Unmarshal([]byte(raw), &nodes); err != nil {
		return nil, err
	}
	for _, node := range nodes {
		if strings.TrimSpace(node.ID) == "" ||
			strings.TrimSpace(node.Address) == "" ||
			strings.TrimSpace(node.CountryCode) == "" ||
			strings.TrimSpace(node.PublicKeyB64) == "" {
			return nil, os.ErrInvalid
		}
	}
	return nodes, nil
}

func main() {
	redisURL := getenv("REDIS_URL", "redis://127.0.0.1:6379")
	port := getenv("PORT", "8080")

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("router: invalid REDIS_URL")
	}

	rdb := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Print("router: redis ping failed, offline queue unavailable until it recovers")
	}
	cancel()

	h := newHub(rdb)
	upgrader := websocket.Upgrader{
		ReadBufferSize:  2048,
		WriteBufferSize: 2048,
		CheckOrigin: func(*http.Request) bool {
			return true
		},
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler.HandleFunc("/api/proxies/active", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		nodes, err := activeProxyNodes()
		if err != nil {
			log.Print("router: invalid ACTIVE_PROXIES configuration")
			http.Error(w, "proxy registry unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(nodes); err != nil {
			log.Print("router: proxy registry response failed")
		}
	})

	handler.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
		if userID == "" {
			userID = uuid.NewString()
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		c := &client{
			userID: userID,
			conn:   conn,
			send:   make(chan []byte, 64),
			done:   make(chan struct{}),
			hub:    h,
		}
		h.register(c)
		h.deliverOffline(context.Background(), userID)

		ack, err := json.Marshal(outboundEnvelope{Type: "session", Payload: map[string]string{"user_id": userID}})
		if err == nil {
			_ = h.deliver(userID, ack)
		}

		go c.writePump()
		c.readPump()
	})

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("router: http server stopped: ", err)
	}
}
