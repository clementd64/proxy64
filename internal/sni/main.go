package sni

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/clementd64/proxy64/internal/utils"
)

type server struct {
	allowed  []*net.IPNet
	prefix   net.IP
	listenIP net.IP
}

func Run(ctx context.Context, log *slog.Logger, env func(string) string) error {
	addrValue := strings.TrimSpace(env("SNI_ADDR"))
	portValue := strings.TrimSpace(env("SNI_PORT"))
	if addrValue == "" && portValue == "" {
		log.InfoContext(ctx, "SNI_ADDR and SNI_PORT not set, skipping sni service")
		return nil
	}

	allowed, err := parseAllowedCIDRs(env("SNI_ALLOWED_CIDRS"))
	if err != nil {
		return fmt.Errorf("failed to parse SNI_ALLOWED_CIDRS: %w", err)
	}

	_, prefix, err := net.ParseCIDR(strings.TrimSpace(env("SNI_PREFIX")))
	if err != nil {
		return fmt.Errorf("failed to parse SNI_PREFIX: %w", err)
	}

	server := &server{
		allowed:  allowed,
		prefix:   prefix.IP,
		listenIP: net.ParseIP(addrValue),
	}

	return utils.ListenTCP(ctx, log, utils.TCPServer{
		Network: "tcp4",
		Addr:    net.JoinHostPort(addrValue, portValue),
		Handler: server.handle,
	})
}

func (s *server) handle(ctx context.Context, log *slog.Logger, clientConn *net.TCPConn) {
	if err := clientConn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		log.ErrorContext(ctx, "failed to set read deadline", "err", err)
		return
	}

	serverName, bufferedHandshake, err := peekServerName(clientConn)
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

	target, err := s.resolveTarget(ctx, serverName)
	if err != nil {
		log.ErrorContext(ctx, "server name is not allowed", "serverName", serverName, "err", err)
		return
	}

	log.InfoContext(ctx, "connection", "src", clientConn.RemoteAddr(), "serverName", serverName, "dst", target)

	if err := utils.ProxyTCPFrom(ctx, clientConn, target, s.getSourceAddr(clientConn), bufferedHandshake); err != nil {
		log.ErrorContext(ctx, "failed to connect", "dst", target, "err", err)
	}
}

func (s *server) getSourceAddr(conn *net.TCPConn) *net.TCPAddr {
	tcpAddr := conn.RemoteAddr().(*net.TCPAddr)
	addr := &net.TCPAddr{
		IP:   make(net.IP, net.IPv6len),
		Port: tcpAddr.Port,
	}
	copy(addr.IP, s.prefix)
	copy(addr.IP[12:16], tcpAddr.IP.To4())
	return addr
}

func (s *server) resolveTarget(ctx context.Context, serverName string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", serverName)
	if err != nil {
		return "", fmt.Errorf("failed to resolve A records for %q: %w", serverName, err)
	}

	pointsToListener := false
	for _, ip := range ips {
		if ip.Equal(s.listenIP) {
			pointsToListener = true
			break
		}
	}
	if !pointsToListener {
		return "", fmt.Errorf("A records for %q do not point to SNI_ADDR %s", serverName, s.listenIP)
	}

	ips, err = net.DefaultResolver.LookupIP(ctx, "ip6", serverName)
	if err != nil {
		return "", fmt.Errorf("failed to resolve AAAA records for %q: %w", serverName, err)
	}

	for _, ip := range ips {
		for _, cidr := range s.allowed {
			if cidr.Contains(ip) {
				return net.JoinHostPort(ip.String(), "443"), nil
			}
		}
	}

	return "", fmt.Errorf("no resolved IP for %q is in allowed ranges", serverName)
}

func parseAllowedCIDRs(value string) ([]*net.IPNet, error) {
	var allowed []*net.IPNet

	for _, part := range strings.Split(value, ",") {
		_, ip, err := net.ParseCIDR(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}

		if ip.IP.To4() != nil {
			return nil, fmt.Errorf("not an IPv6 address: %s", part)
		}

		allowed = append(allowed, ip)
	}

	return allowed, nil
}
