package main

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

func TestHeaderRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := writeHeader(&buf, "example.com:443"); err != nil {
		t.Fatal(err)
	}
	got, err := readHeader(&buf)
	if err != nil || got != "example.com:443" {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestReadHeaderRejectsBadMagic(t *testing.T) {
	if _, err := readHeader(bytes.NewReader([]byte("XXXX\x00\x01a"))); err == nil {
		t.Fatal("expected error")
	}
}

func TestIsForbiddenIP(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1":       true,
		"10.1.2.3":        true,
		"192.168.0.1":     true,
		"172.16.0.1":      true,
		"169.254.169.254": true,
		"0.0.0.0":         true,
		"::1":             true,
		"fe80::1":         true,
		"8.8.8.8":         false,
		"2606:4700::1111": false,
	}
	for s, want := range cases {
		if got := isForbiddenIP(net.ParseIP(s)); got != want {
			t.Errorf("%s: got %v want %v", s, got, want)
		}
	}
}

func TestDialDestinationBlocksPrivate(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		if c, err := ln.Accept(); err == nil {
			c.Close()
		}
	}()

	cfg := config{dialTimeout: time.Second}
	if _, err := dialDestination(cfg, ln.Addr().String()); err == nil {
		t.Fatal("loopback must be blocked by default")
	}

	cfg.allowPrivate = true
	c, err := dialDestination(cfg, ln.Addr().String())
	if err != nil {
		t.Fatalf("allowPrivate dial failed: %v", err)
	}
	c.Close()
}

func TestSocks5Handshake(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		client.Write([]byte{5, 1, 0})
		io.ReadFull(client, make([]byte, 2))
		client.Write([]byte{5, 1, 0, 3, 7, 'a', '.', 'b', 'c', 'd', 'e', 'f', 0x01, 0xBB})
	}()

	dst, err := socks5Handshake(server)
	if err != nil || dst != "a.bcdef:443" {
		t.Fatalf("got %q, err %v", dst, err)
	}
}

func TestLoadConfig(t *testing.T) {
	t.Setenv("NODE_ROLE", "bogus")
	if _, err := loadConfig(); err == nil {
		t.Fatal("invalid role must fail")
	}

	t.Setenv("NODE_ROLE", "exit")
	t.Setenv("MAX_CONNECTIONS", "0")
	if _, err := loadConfig(); err == nil {
		t.Fatal("invalid MAX_CONNECTIONS must fail")
	}

	t.Setenv("MAX_CONNECTIONS", "5")
	cfg, err := loadConfig()
	if err != nil || cfg.maxConns != 5 || cfg.allowPrivate {
		t.Fatalf("unexpected config %+v err %v", cfg, err)
	}
}

func TestTunnelCopiesBothWaysAndIdleTimeout(t *testing.T) {
	a1, a2 := net.Pipe()
	b1, b2 := net.Pipe()
	cfg := config{idleTimeout: 300 * time.Millisecond}

	done := make(chan struct{})
	go func() {
		tunnel(cfg, a2, b1)
		close(done)
	}()

	// Traffic beyond one idle period keeps the tunnel alive.
	buf := make([]byte, 1)
	for i := 0; i < 4; i++ {
		time.Sleep(150 * time.Millisecond)
		go a1.Write([]byte{'x'})
		if _, err := io.ReadFull(b2, buf); err != nil || buf[0] != 'x' {
			t.Fatalf("a->b failed at %d: %v", i, err)
		}
		go b2.Write([]byte{'y'})
		if _, err := io.ReadFull(a1, buf); err != nil || buf[0] != 'y' {
			t.Fatalf("b->a failed at %d: %v", i, err)
		}
	}

	// Silence ends the tunnel.
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("tunnel did not close after idle timeout")
	}
	a1.Close()
	b2.Close()
}
