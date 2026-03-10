package services

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"db-sync-cli/internal/config"

	"golang.org/x/net/proxy"
)

type proxyTunnel struct {
	listener net.Listener
	proxyURL *url.URL
	target   string

	closeOnce sync.Once
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func newProxyTunnel(mysqlConfig config.MySQLConfig) (*proxyTunnel, error) {
	proxyURL, err := url.Parse(mysqlConfig.ProxyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy URL: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to create local proxy tunnel: %w", err)
	}

	tunnel := &proxyTunnel{
		listener: listener,
		proxyURL: proxyURL,
		target:   net.JoinHostPort(mysqlConfig.Host, strconv.Itoa(mysqlConfig.Port)),
	}

	go tunnel.serve()

	return tunnel, nil
}

func (t *proxyTunnel) Host() string {
	host, _, err := net.SplitHostPort(t.listener.Addr().String())
	if err != nil {
		return "127.0.0.1"
	}

	return host
}

func (t *proxyTunnel) Port() int {
	_, rawPort, err := net.SplitHostPort(t.listener.Addr().String())
	if err != nil {
		return 0
	}

	port, err := strconv.Atoi(rawPort)
	if err != nil {
		return 0
	}

	return port
}

func (t *proxyTunnel) Close() error {
	var closeErr error
	t.closeOnce.Do(func() {
		closeErr = t.listener.Close()
	})

	return closeErr
}

func (t *proxyTunnel) serve() {
	for {
		clientConn, err := t.listener.Accept()
		if err != nil {
			if isListenerClosed(err) {
				return
			}
			continue
		}

		go t.handle(clientConn)
	}
}

func (t *proxyTunnel) handle(clientConn net.Conn) {
	upstreamConn, err := t.dialTarget()
	if err != nil {
		_ = clientConn.Close()
		return
	}

	proxyConnections(clientConn, upstreamConn)
}

func (t *proxyTunnel) dialTarget() (net.Conn, error) {
	switch strings.ToLower(t.proxyURL.Scheme) {
	case "socks5", "socks5h":
		return t.dialSOCKS5()
	case "http", "https":
		return t.dialHTTPConnect()
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", t.proxyURL.Scheme)
	}
}

func (t *proxyTunnel) dialSOCKS5() (net.Conn, error) {
	var auth *proxy.Auth
	if t.proxyURL.User != nil {
		password, _ := t.proxyURL.User.Password()
		auth = &proxy.Auth{
			User:     t.proxyURL.User.Username(),
			Password: password,
		}
	}

	dialer, err := proxy.SOCKS5("tcp", t.proxyURL.Host, auth, &net.Dialer{Timeout: 10 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to configure SOCKS5 proxy: %w", err)
	}

	conn, err := dialer.Dial("tcp", t.target)
	if err != nil {
		return nil, fmt.Errorf("failed to connect through SOCKS5 proxy: %w", err)
	}

	return conn, nil
}

func (t *proxyTunnel) dialHTTPConnect() (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	var conn net.Conn
	var err error
	if strings.EqualFold(t.proxyURL.Scheme, "https") {
		tlsConfig := &tls.Config{ServerName: t.proxyURL.Hostname(), MinVersion: tls.VersionTLS12}
		conn, err = tls.DialWithDialer(dialer, "tcp", t.proxyURL.Host, tlsConfig)
	} else {
		conn, err = dialer.DialContext(context.Background(), "tcp", t.proxyURL.Host)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to HTTP proxy: %w", err)
	}

	request := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: t.target},
		Host:   t.target,
		Header: make(http.Header),
	}
	request.Header.Set("Host", t.target)
	if t.proxyURL.User != nil {
		password, _ := t.proxyURL.User.Password()
		credentials := base64.StdEncoding.EncodeToString([]byte(t.proxyURL.User.Username() + ":" + password))
		request.Header.Set("Proxy-Authorization", "Basic "+credentials)
	}

	if err := request.Write(conn); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to create HTTP CONNECT tunnel: %w", err)
	}

	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, request)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to read HTTP CONNECT response: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		_ = conn.Close()
		return nil, fmt.Errorf("HTTP proxy refused CONNECT tunnel: %s", response.Status)
	}

	return &bufferedConn{Conn: conn, reader: reader}, nil
}

func proxyConnections(clientConn net.Conn, upstreamConn net.Conn) {
	var closeOnce sync.Once
	closeBoth := func() {
		closeOnce.Do(func() {
			_ = clientConn.Close()
			_ = upstreamConn.Close()
		})
	}

	var waitGroup sync.WaitGroup
	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()
		_, _ = io.Copy(upstreamConn, clientConn)
		closeBoth()
	}()

	go func() {
		defer waitGroup.Done()
		_, _ = io.Copy(clientConn, upstreamConn)
		closeBoth()
	}()

	waitGroup.Wait()
}

func isListenerClosed(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(strings.ToLower(err.Error()), "use of closed network connection")
}
