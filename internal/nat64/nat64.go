package nat64

import (
	"context"
	"log/slog"
	"net"
	"strconv"

	"github.com/clementd64/proxy64/internal/utils"
)

func Run(ctx context.Context, log *slog.Logger, env func(string) string) error {
	port := env("NAT64_PORT")
	if port == "" {
		log.InfoContext(ctx, "NAT64_PORT not set, skipping nat64 service")
		return nil
	}

	return utils.ListenTCP(ctx, log, utils.TCPServer{
		Network: "tcp6",
		Addr:    net.JoinHostPort("::", port),
		Handler: handle,
		Control: utils.ListenerIPTransparent,
	})
}

func handle(ctx context.Context, log *slog.Logger, c *net.TCPConn) {
	addr := c.LocalAddr().(*net.TCPAddr)
	target := net.JoinHostPort(addr.IP[12:16].String(), strconv.Itoa(addr.Port))

	log.InfoContext(ctx, "connection", "src", c.RemoteAddr(), "dst", target)

	if err := utils.ProxyTCP(ctx, c, target); err != nil {
		log.ErrorContext(ctx, "failed to connect", "dst", target, "err", err)
	}
}
