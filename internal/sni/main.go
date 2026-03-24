package sni

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"
)

func Listen(addr string, allowed []net.IPNet) error {
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
		go handleConnection(conn, &allowed)
	}
}

func handleConnection(clientConn net.Conn, allowed *[]net.IPNet) {
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

	backendAddr, backendIP, err := resolveAllowedBackendAddr(serverName, *allowed)
	if err != nil {
		slog.Error("server name is not allowed", "serverName", serverName, "err", err)
		return
	}

	backendConn, err := net.DialTimeout("tcp", backendAddr, 5*time.Second)
	if err != nil {
		slog.Error("failed to connect to backend", "serverName", serverName, "backendIP", backendIP.String(), "err", err)
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

func resolveAllowedBackendAddr(serverName string, allowed []net.IPNet) (string, net.IP, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", serverName)
	if err != nil {
		return "", nil, err
	}

	for _, ip := range ips {
		for _, cidr := range allowed {
			if cidr.Contains(ip) {
				return net.JoinHostPort(ip.String(), "443"), ip, nil
			}
		}
	}

	return "", nil, fmt.Errorf("no resolved IP for %q is in allowed ranges", serverName)
}
