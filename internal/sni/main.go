package sni

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/clementd64/proxy64/internal/utils"
)

func Listen(ctx context.Context, log *slog.Logger, env func(string) string) error {
	addr := env("SNI_LISTEN")
	if addr == "" {
		log.InfoContext(ctx, "SNI_LISTEN not set, skipping sni service")
		return nil
	}

	allowed, err := parseAllowedCIDRs(env("SNI_ALLOWED_CIDRS"))
	if err != nil {
		return fmt.Errorf("failed to parse SNI_ALLOWED_CIDRS: %w", err)
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	go func() {
		<-ctx.Done()
		log.InfoContext(ctx, "Shutting down server")
		listener.Close()
	}()

	log.InfoContext(ctx, "Starting server", "addr", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		go handleConnection(ctx, log, conn, allowed)
	}
}

func handleConnection(ctx context.Context, log *slog.Logger, clientConn net.Conn, allowed []net.IPNet) {
	defer clientConn.Close()

	if err := clientConn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		log.ErrorContext(ctx, "failed to set read deadline", "err", err)
		return
	}

	serverName, clientReader, err := peekServerName(clientConn)
	if err != nil {
		log.ErrorContext(ctx, "failed to read server name", "err", err)
		return
	}

	if serverName == "" {
		log.ErrorContext(ctx, "failed to read server name")
		return
	}

	if err := clientConn.SetReadDeadline(time.Time{}); err != nil {
		log.ErrorContext(ctx, "failed to clear read deadline", "err", err)
		return
	}

	backendAddr, err := resolveAllowedBackendAddr(ctx, serverName, allowed)
	if err != nil {
		log.ErrorContext(ctx, "server name is not allowed", "serverName", serverName, "err", err)
		return
	}

	log.InfoContext(ctx, "connection", "src", clientConn.RemoteAddr(), "serverName", serverName, "dst", backendAddr)

	if err := utils.ProxyTCP(clientConn.(*net.TCPConn), backendAddr, clientReader); err != nil {
		log.ErrorContext(ctx, "failed to connect", "dst", backendAddr, "err", err)
	}
}

func resolveAllowedBackendAddr(ctx context.Context, serverName string, allowed []net.IPNet) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", serverName)
	if err != nil {
		return "", err
	}

	for _, ip := range ips {
		for _, cidr := range allowed {
			if cidr.Contains(ip) {
				return net.JoinHostPort(ip.String(), "443"), nil
			}
		}
	}

	return "", fmt.Errorf("no resolved IP for %q is in allowed ranges", serverName)
}

func parseAllowedCIDRs(value string) ([]net.IPNet, error) {
	var allowed []net.IPNet

	for _, part := range strings.Split(value, ",") {
		_, ip, err := net.ParseCIDR(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}

		if ip.IP.To4() != nil {
			return nil, fmt.Errorf("not an IPv6 address: %s", part)
		}

		allowed = append(allowed, *ip)
	}

	return allowed, nil
}
