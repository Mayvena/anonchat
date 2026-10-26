package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	roleGuard  = "guard"
	roleMiddle = "middle"
	roleExit   = "exit"

	headerMagic = "DST1"

	socksVersion5       = 0x05
	socksAuthNoRequired = 0x00
	socksCmdConnect     = 0x01
	socksAtypIPv4       = 0x01
	socksAtypDomain     = 0x03
	socksAtypIPv6       = 0x04
)

type config struct {
	role     string
	listen   string
	nextHop  string
	certFile string
	keyFile  string

	insecureSkipVerify bool
	allowPrivate       bool
	maxConns           int
	dialTimeout        time.Duration
	idleTimeout        time.Duration
}

// main stays silent by policy; a non-zero exit code is the only failure signal.
func main() {
	cfg, err := loadConfig()
	if err != nil {
		os.Exit(2)
	}
	if err := run(cfg); err != nil {
		os.Exit(1)
	}
}

func loadConfig() (config, error) {
	role := strings.ToLower(strings.TrimSpace(os.Getenv("NODE_ROLE")))
	if role != roleGuard && role != roleMiddle && role != roleExit {
		return config{}, errors.New("invalid NODE_ROLE")
	}

	listenPort := strings.TrimSpace(os.Getenv("LISTEN_PORT"))
	if listenPort == "" {
		listenPort = "9001"
	}

	certFile := strings.TrimSpace(os.Getenv("TLS_CERT_FILE"))
	keyFile := strings.TrimSpace(os.Getenv("TLS_KEY_FILE"))
	if (certFile == "") != (keyFile == "") {
		return config{}, errors.New("TLS_CERT_FILE and TLS_KEY_FILE must be provided together")
	}

	nextHop := strings.TrimSpace(os.Getenv("NEXT_HOP_ADDR"))
	if nextHop == "" {
		switch role {
		case roleGuard:
			nextHop = "proxy-middle:9002"
		case roleMiddle:
			nextHop = "proxy-exit:9003"
		}
	}
	if (role == roleGuard || role == roleMiddle) && nextHop == "" {
		return config{}, errors.New("NEXT_HOP_ADDR is required for guard and middle")
	}

	insecure := strings.EqualFold(strings.TrimSpace(os.Getenv("TLS_INSECURE_SKIP_VERIFY")), "true")
	allowPrivate := strings.EqualFold(strings.TrimSpace(os.Getenv("EXIT_ALLOW_PRIVATE")), "true")

	maxConns := 1024
	if v := strings.TrimSpace(os.Getenv("MAX_CONNECTIONS")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return config{}, errors.New("invalid MAX_CONNECTIONS")
		}
		maxConns = n
	}

	return config{
		role:               role,
		listen:             ":" + listenPort,
		nextHop:            nextHop,
		certFile:           certFile,
		keyFile:            keyFile,
		insecureSkipVerify: insecure,
		allowPrivate:       allowPrivate,
		maxConns:           maxConns,
		dialTimeout:        10 * time.Second,
		idleTimeout:        90 * time.Second,
	}, nil
}

func run(cfg config) error {
	cert, err := loadOrGenerateCertificate(cfg)
	if err != nil {
		return err
	}

	ln, err := tls.Listen("tcp", cfg.listen, &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	})
	if err != nil {
		return err
	}
	defer ln.Close()

	sem := make(chan struct{}, cfg.maxConns)
	backoff := 5 * time.Millisecond
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return err
			}
			time.Sleep(backoff)
			if backoff < time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = 5 * time.Millisecond

		select {
		case sem <- struct{}{}:
			go func() {
				defer func() { <-sem }()
				handleIncoming(cfg, conn)
			}()
		default:
			_ = conn.Close()
		}
	}
}

func loadOrGenerateCertificate(cfg config) (tls.Certificate, error) {
	if cfg.certFile != "" && cfg.keyFile != "" {
		return tls.LoadX509KeyPair(cfg.certFile, cfg.keyFile)
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	serialNumber, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return tls.Certificate{}, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "anonchat-proxy",
		},
		NotBefore:             time.Now().Add(-5 * time.Minute),
		NotAfter:              time.Now().Add(24 * time.Hour),
		DNSNames:              []string{"localhost"},
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return tls.X509KeyPair(certPEM, keyPEM)
}

func handleIncoming(cfg config, incoming net.Conn) {
	defer incoming.Close()
	_ = incoming.SetDeadline(time.Now().Add(cfg.idleTimeout))

	switch cfg.role {
	case roleGuard:
		handleGuard(cfg, incoming)
	case roleMiddle:
		handleMiddle(cfg, incoming)
	case roleExit:
		handleExit(cfg, incoming)
	}
}

func handleGuard(cfg config, client net.Conn) {
	dst, err := socks5Handshake(client)
	if err != nil {
		return
	}

	upstream, err := dialTLS(cfg, cfg.nextHop)
	if err != nil {
		_ = writeSocks5Reply(client, 0x01, nil)
		return
	}
	defer upstream.Close()

	if err := writeHeader(upstream, dst); err != nil {
		_ = writeSocks5Reply(client, 0x01, nil)
		return
	}

	boundAddr := upstream.LocalAddr()
	if err := writeSocks5Reply(client, 0x00, boundAddr); err != nil {
		return
	}

	tunnel(cfg, client, upstream)
}

func handleMiddle(cfg config, incoming net.Conn) {
	upstream, err := dialTLS(cfg, cfg.nextHop)
	if err != nil {
		return
	}
	defer upstream.Close()

	tunnel(cfg, incoming, upstream)
}

func handleExit(cfg config, incoming net.Conn) {
	dst, err := readHeader(incoming)
	if err != nil {
		return
	}

	target, err := dialDestination(cfg, dst)
	if err != nil {
		return
	}
	defer target.Close()

	tunnel(cfg, incoming, target)
}

// dialDestination resolves once, validates every address, and dials the
// validated IP so a second DNS answer cannot bypass the check.
func dialDestination(cfg config, dst string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(dst)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.dialTimeout)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, errors.New("no addresses")
	}
	if !cfg.allowPrivate {
		for _, ip := range ips {
			if isForbiddenIP(ip.IP) {
				return nil, errors.New("destination not allowed")
			}
		}
	}

	d := net.Dialer{Timeout: cfg.dialTimeout}
	return d.DialContext(ctx, "tcp", net.JoinHostPort(ips[0].IP.String(), port))
}

func isForbiddenIP(ip net.IP) bool {
	return ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast()
}

func dialTLS(cfg config, addr string) (net.Conn, error) {
	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: cfg.insecureSkipVerify,
	}
	d := net.Dialer{Timeout: cfg.dialTimeout}
	return tls.DialWithDialer(&d, "tcp", addr, tlsCfg)
}

// idleConn renews the deadline on every successful transfer, so only
// connections that stay silent for the idle timeout are dropped.
type idleConn struct {
	net.Conn
	timeout time.Duration
}

func (c *idleConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		_ = c.Conn.SetDeadline(time.Now().Add(c.timeout))
	}
	return n, err
}

func (c *idleConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	if n > 0 {
		_ = c.Conn.SetDeadline(time.Now().Add(c.timeout))
	}
	return n, err
}

func tunnel(cfg config, rawA net.Conn, rawB net.Conn) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := &idleConn{Conn: rawA, timeout: cfg.idleTimeout}
	b := &idleConn{Conn: rawB, timeout: cfg.idleTimeout}
	_ = rawA.SetDeadline(time.Now().Add(cfg.idleTimeout))
	_ = rawB.SetDeadline(time.Now().Add(cfg.idleTimeout))

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(b, a)
		cancel()
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(a, b)
		cancel()
	}()

	<-ctx.Done()
	_ = rawA.SetDeadline(time.Now().Add(100 * time.Millisecond))
	_ = rawB.SetDeadline(time.Now().Add(100 * time.Millisecond))
	wg.Wait()
}

func socks5Handshake(conn net.Conn) (string, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", err
	}
	if header[0] != socksVersion5 {
		return "", errors.New("unsupported socks version")
	}

	nMethods := int(header[1])
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return "", err
	}

	methodAccepted := false
	for _, m := range methods {
		if m == socksAuthNoRequired {
			methodAccepted = true
			break
		}
	}
	if !methodAccepted {
		_, _ = conn.Write([]byte{socksVersion5, 0xFF})
		return "", errors.New("no supported auth method")
	}

	if _, err := conn.Write([]byte{socksVersion5, socksAuthNoRequired}); err != nil {
		return "", err
	}

	req := make([]byte, 4)
	if _, err := io.ReadFull(conn, req); err != nil {
		return "", err
	}
	if req[0] != socksVersion5 || req[1] != socksCmdConnect {
		return "", errors.New("unsupported socks command")
	}

	host, err := readSocksAddress(conn, req[3])
	if err != nil {
		return "", err
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return "", err
	}
	port := binary.BigEndian.Uint16(portBuf)
	return net.JoinHostPort(host, itoa(int(port))), nil
}

func writeSocks5Reply(conn net.Conn, rep byte, addr net.Addr) error {
	ip := net.IPv4(0, 0, 0, 0)
	port := uint16(0)

	if ta, ok := addr.(*net.TCPAddr); ok {
		if v4 := ta.IP.To4(); v4 != nil {
			ip = v4
		} else {
			ip = net.IPv4(0, 0, 0, 0)
		}
		if ta.Port >= 0 && ta.Port <= 65535 {
			port = uint16(ta.Port)
		}
	}

	resp := []byte{socksVersion5, rep, 0x00, socksAtypIPv4, ip[0], ip[1], ip[2], ip[3], 0x00, 0x00}
	binary.BigEndian.PutUint16(resp[8:], port)
	_, err := conn.Write(resp)
	return err
}

func readSocksAddress(r io.Reader, atyp byte) (string, error) {
	switch atyp {
	case socksAtypIPv4:
		buf := make([]byte, net.IPv4len)
		if _, err := io.ReadFull(r, buf); err != nil {
			return "", err
		}
		return net.IP(buf).String(), nil
	case socksAtypIPv6:
		buf := make([]byte, net.IPv6len)
		if _, err := io.ReadFull(r, buf); err != nil {
			return "", err
		}
		return net.IP(buf).String(), nil
	case socksAtypDomain:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(r, lenBuf); err != nil {
			return "", err
		}
		domainLen := int(lenBuf[0])
		if domainLen == 0 {
			return "", errors.New("invalid domain")
		}
		buf := make([]byte, domainLen)
		if _, err := io.ReadFull(r, buf); err != nil {
			return "", err
		}
		return string(buf), nil
	default:
		return "", errors.New("unsupported atyp")
	}
}

func writeHeader(w io.Writer, dst string) error {
	if len(dst) > 65535 {
		return errors.New("destination too long")
	}
	buf := make([]byte, 0, 6+len(dst))
	buf = append(buf, []byte(headerMagic)...)
	lenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBytes, uint16(len(dst)))
	buf = append(buf, lenBytes...)
	buf = append(buf, dst...)
	_, err := w.Write(buf)
	return err
}

func readHeader(r io.Reader) (string, error) {
	prefix := make([]byte, len(headerMagic))
	if _, err := io.ReadFull(r, prefix); err != nil {
		return "", err
	}
	if string(prefix) != headerMagic {
		return "", errors.New("invalid header magic")
	}

	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return "", err
	}
	sz := int(binary.BigEndian.Uint16(lenBuf))
	if sz <= 0 {
		return "", errors.New("invalid destination size")
	}

	dst := make([]byte, sz)
	if _, err := io.ReadFull(r, dst); err != nil {
		return "", err
	}
	return string(dst), nil
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}

	negative := v < 0
	if negative {
		v = -v
	}

	var digits [20]byte
	i := len(digits)
	for v > 0 {
		i--
		digits[i] = byte('0' + v%10)
		v /= 10
	}
	if negative {
		i--
		digits[i] = '-'
	}
	return string(digits[i:])
}
