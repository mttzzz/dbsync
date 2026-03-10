package services

import (
	"bufio"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/models"
)

func TestProxyTunnel_HTTPConnectPreservesBufferedData(t *testing.T) {
	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start target listener: %v", err)
	}
	defer targetListener.Close()

	const greeting = "mysql-handshake"
	targetDone := make(chan struct{})
	go func() {
		defer close(targetDone)
		conn, err := targetListener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = conn.Write([]byte(greeting))
		_, _ = io.Copy(io.Discard, conn)
	}()

	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start proxy listener: %v", err)
	}
	defer proxyListener.Close()

	proxyDone := make(chan struct{})
	go func() {
		defer close(proxyDone)
		conn, err := proxyListener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			if line == "\r\n" {
				break
			}
		}

		targetConn, err := net.Dial("tcp", targetListener.Addr().String())
		if err != nil {
			return
		}
		defer targetConn.Close()

		initial := make([]byte, len(greeting))
		if _, err := io.ReadFull(targetConn, initial); err != nil {
			return
		}

		response := "HTTP/1.1 200 Connection Established\r\n\r\n" + string(initial)
		if _, err := conn.Write([]byte(response)); err != nil {
			return
		}
	}()

	proxyURL, err := url.Parse("http://" + proxyListener.Addr().String())
	if err != nil {
		t.Fatalf("failed to parse proxy URL: %v", err)
	}

	tunnel := &proxyTunnel{
		proxyURL: proxyURL,
		target:   targetListener.Addr().String(),
	}

	upstreamConn, err := tunnel.dialHTTPConnect()
	if err != nil {
		t.Fatalf("dialHTTPConnect() error = %v", err)
	}
	defer upstreamConn.Close()
	_ = upstreamConn.SetReadDeadline(time.Now().Add(2 * time.Second))

	buffer := make([]byte, len(greeting))
	if _, err := io.ReadFull(upstreamConn, buffer); err != nil {
		t.Fatalf("failed to read buffered greeting: %v", err)
	}

	if got := string(buffer); got != greeting {
		t.Fatalf("greeting = %q, want %q", got, greeting)
	}

	_ = targetListener.Close()
	_ = proxyListener.Close()
	<-targetDone
	<-proxyDone
}

func TestProxyTunnel_HTTPConnectSendsProxyAuthorization(t *testing.T) {
	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start proxy listener: %v", err)
	}
	defer proxyListener.Close()

	authHeader := make(chan string, 1)
	proxyDone := make(chan struct{})
	go func() {
		defer close(proxyDone)
		conn, err := proxyListener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			if strings.HasPrefix(strings.ToLower(line), "proxy-authorization:") {
				authHeader <- strings.TrimSpace(strings.TrimPrefix(line, "Proxy-Authorization:"))
			}
			if line == "\r\n" {
				break
			}
		}

		_, _ = conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	}()

	proxyURL, err := url.Parse("http://alice:secret@" + proxyListener.Addr().String())
	if err != nil {
		t.Fatalf("failed to parse proxy URL: %v", err)
	}

	tunnel := &proxyTunnel{
		proxyURL: proxyURL,
		target:   "127.0.0.1:3306",
	}

	conn, err := tunnel.dialHTTPConnect()
	if err != nil {
		t.Fatalf("dialHTTPConnect() error = %v", err)
	}
	_ = conn.Close()
	<-proxyDone

	select {
	case got := <-authHeader:
		if got != "Basic YWxpY2U6c2VjcmV0" {
			t.Fatalf("Proxy-Authorization = %q, want %q", got, "Basic YWxpY2U6c2VjcmV0")
		}
	default:
		t.Fatal("expected Proxy-Authorization header")
	}
}

func TestProxyTunnel_DirectModeTracksTraffic(t *testing.T) {
	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start target listener: %v", err)
	}
	defer targetListener.Close()

	const responsePayload = "pong"
	targetDone := make(chan struct{})
	go func() {
		defer close(targetDone)
		conn, err := targetListener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		buffer := make([]byte, 4)
		if _, err := io.ReadFull(conn, buffer); err != nil {
			return
		}
		_, _ = conn.Write([]byte(responsePayload))
	}()

	host, rawPort, err := net.SplitHostPort(targetListener.Addr().String())
	if err != nil {
		t.Fatalf("failed to split host/port: %v", err)
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	tunnel, err := newProxyTunnel(config.MySQLConfig{Host: host, Port: port})
	if err != nil {
		t.Fatalf("newProxyTunnel() error = %v", err)
	}
	defer tunnel.Close()

	clientConn, err := net.Dial("tcp", net.JoinHostPort(tunnel.Host(), strconv.Itoa(tunnel.Port())))
	if err != nil {
		t.Fatalf("failed to connect to relay listener: %v", err)
	}
	defer clientConn.Close()

	if _, err := clientConn.Write([]byte("ping")); err != nil {
		t.Fatalf("failed to write request: %v", err)
	}

	response := make([]byte, len(responsePayload))
	if _, err := io.ReadFull(clientConn, response); err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	if got := string(response); got != responsePayload {
		t.Fatalf("response = %q, want %q", got, responsePayload)
	}

	_ = clientConn.Close()
	_ = tunnel.Close()
	<-targetDone

	metrics := tunnel.Metrics()
	if metrics.Mode != models.TransportModeDirect {
		t.Fatalf("Metrics mode = %q, want %q", metrics.Mode, models.TransportModeDirect)
	}
	if metrics.BytesOut < 4 {
		t.Fatalf("BytesOut = %d, want >= 4", metrics.BytesOut)
	}
	if metrics.BytesIn < int64(len(responsePayload)) {
		t.Fatalf("BytesIn = %d, want >= %d", metrics.BytesIn, len(responsePayload))
	}
	if metrics.AverageBytesPerSecond <= 0 {
		t.Fatalf("AverageBytesPerSecond = %f, want > 0", metrics.AverageBytesPerSecond)
	}
}
