package sni

import (
	"io"
	"log/slog"
	"net"
	"sync"
	"time"
)

func Listen(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			slog.Error("failed to accept connection", "err", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	if err := clientConn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		slog.Error("failed to set read deadline", "err", err)
		return
	}

	serverName, clientReader, err := peekServerName(clientConn)
	if err != nil {
		slog.Error("failed to read server name", "err", err)
		return
	}

	if serverName == "" {
		slog.Error("failed to read server name")
		return
	}

	if err := clientConn.SetReadDeadline(time.Time{}); err != nil {
		slog.Error("failed to clear read deadline", "err", err)
		return
	}

	backendConn, err := net.DialTimeout("tcp", net.JoinHostPort(serverName, "443"), 5*time.Second)
	if err != nil {
		slog.Error("failed to connect to backend", "serverName", serverName, "err", err)
		return
	}
	defer backendConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		io.Copy(clientConn, backendConn)
		clientConn.(*net.TCPConn).CloseWrite()
		wg.Done()
	}()
	go func() {
		io.Copy(backendConn, clientReader)
		backendConn.(*net.TCPConn).CloseWrite()
		wg.Done()
	}()

	wg.Wait()
}
