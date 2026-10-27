package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	offlineTTL            = 300 * time.Second
	heartbeatEvery        = 10 * time.Second
	pongWait              = 20 * time.Second
	writeWait             = 10 * time.Second
	maxIncomingBytes      = 64 * 1024
	proxyRegistryRedisKey = "anonchat:proxies"
	proxyRefreshEvery     = 5 * time.Minute
	proxyDialTimeout      = 3 * time.Second
	proxyFetchTimeout     = 8 * time.Second
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
	ID           string  `json:"id"`
	Address      string  `json:"address"`
	CountryCode  string  `json:"country_code"`
	PublicKeyB64 string  `json:"public_key_b64"`
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

func proxySources() []string {
	raw := strings.TrimSpace(os.Getenv("PROXY_SOURCES"))
	if raw == "" {
		return []string{
			"https://raw.githubusercontent.com/TheSpeedX/SOCKS-List/master/socks.txt",
			"https://raw.githubusercontent.com/jetkai/proxy-list/main/online-proxies/txt/proxies-socks5.txt",
		}
	}

	var out []string
	for _, item := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' || r == ' ' || r == '\t' }) {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func normalizeProxyAddress(raw string) string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return ""
	}
	candidate = strings.TrimPrefix(candidate, "socks5://")
	candidate = strings.TrimPrefix(candidate, "socks://")
	candidate = strings.TrimPrefix(candidate, "http://")
	candidate = strings.TrimPrefix(candidate, "https://")
	if i := strings.IndexAny(candidate, " \t\r\n/"); i >= 0 {
		candidate = candidate[:i]
	}
	if strings.Count(candidate, ":") < 1 {
		return ""
	}
	if host, port, err := net.SplitHostPort(candidate); err == nil {
		candidate = net.JoinHostPort(host, port)
	}
	return candidate
}

func fetchProxyCandidates() []string {
	sources := proxySources()
	if len(sources) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var list []string
	for _, src := range sources {
		ctx, cancel := context.WithTimeout(context.Background(), proxyFetchTimeout)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
		if err != nil {
			cancel()
			continue
		}
		resp, err := http.DefaultClient.Do(req)
		cancel()
		if err != nil || resp == nil {
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(body), "\n") {
			candidate := normalizeProxyAddress(line)
			if candidate == "" {
				continue
			}
			if _, exists := seen[candidate]; exists {
				continue
			}
			seen[candidate] = struct{}{}
			list = append(list, candidate)
		}
	}
	return list
}

func countryCodeFromIP(host string) string {
	if host == "" {
		return "UN"
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return "UN"
	}
	if strings.EqualFold(host, "localhost") {
		return "UN"
	}
	resp, err := http.Get("https://ipapi.co/" + host + "/json/")
	if err != nil || resp == nil {
		return "UN"
	}
	defer resp.Body.Close()
	var payload struct {
		CountryCode string `json:"country_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "UN"
	}
	if payload.CountryCode == "" {
		return "UN"
	}
	return strings.ToUpper(payload.CountryCode)
}

func countryCoordinates(code string) (float64, float64) {
	switch strings.ToUpper(code) {
	case "DE":
		return 51.1657, 10.4515
	case "US":
		return 37.7749, -122.4194
	case "FR":
		return 46.2276, 2.2137
	case "GB":
		return 55.3781, -3.4360
	case "NL":
		return 52.3676, 4.9041
	case "JP":
		return 36.2048, 138.2529
	case "SG":
		return 1.3521, 103.8198
	case "BR":
		return -14.2350, -51.9253
	default:
		return 0, 0
	}
}

func proxyNodeForAddress(addr string) *proxyNode {
	if addr == "" {
		return nil
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil
	}
	if port == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), proxyDialTimeout)
	defer cancel()
	conn, err := (&net.Dialer{Timeout: proxyDialTimeout}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil
	}
	conn.Close()

	country := countryCodeFromIP(host)
	lat, lon := countryCoordinates(country)
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil
	}
	return &proxyNode{
		ID:           fmt.Sprintf("proxy-%s-%s", strings.ReplaceAll(host, ".", "-"), port),
		Address:      addr,
		CountryCode:  country,
		PublicKeyB64: base64.StdEncoding.EncodeToString(key),
		Latitude:     lat,
		Longitude:    lon,
	}
}

func refreshProxyRegistry(ctx context.Context, rdb *redis.Client) error {
	candidates := fetchProxyCandidates()
	if len(candidates) == 0 {
		return rdb.Set(ctx, proxyRegistryRedisKey, "[]", proxyRefreshEvery).Err()
	}

	results := make(chan *proxyNode, len(candidates))
	for _, candidate := range candidates {
		go func(addr string) {
			results <- proxyNodeForAddress(addr)
		}(candidate)
	}

	registered := make([]proxyNode, 0, len(candidates))
	for range candidates {
		if node := <-results; node != nil {
			registered = append(registered, *node)
		}
	}
	sort.Slice(registered, func(i, j int) bool {
		return strings.ToLower(registered[i].CountryCode) < strings.ToLower(registered[j].CountryCode)
	})
	payload, err := json.Marshal(registered)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, proxyRegistryRedisKey, payload, proxyRefreshEvery).Err()
}

func activeProxyNodes() ([]proxyNode, error) {
	raw := strings.TrimSpace(os.Getenv("ACTIVE_PROXIES"))
	if raw != "" {
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

	redisURL := getenv("REDIS_URL", "redis://127.0.0.1:6379")
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()

	data, err := rdb.Get(context.Background(), proxyRegistryRedisKey).Result()
	if err == redis.Nil {
		return []proxyNode{}, nil
	} else if err != nil {
		return nil, err
	}

	var nodes []proxyNode
	if err := json.Unmarshal([]byte(data), &nodes); err != nil {
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

	if err := refreshProxyRegistry(context.Background(), rdb); err != nil {
		log.Print("router: proxy registry refresh failed: ", err)
	}
	go func() {
		ticker := time.NewTicker(proxyRefreshEvery)
		defer ticker.Stop()
		for range ticker.C {
			if err := refreshProxyRegistry(context.Background(), rdb); err != nil {
				log.Print("router: proxy registry refresh failed: ", err)
			}
		}
	}()

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
