package nat64

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strconv"
	"syscall"

	"github.com/clementd64/proxy64/internal/utils"
)

func Listen(ctx context.Context, log *slog.Logger, env func(string) string) error {
	port := env("NAT64_PORT")
	if port == "" {
		log.InfoContext(ctx, "NAT64_PORT not set, skipping nat64 service")
		return nil
	}

	addr, err := net.ResolveTCPAddr("tcp6", net.JoinHostPort("::", port))
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP("tcp6", addr)
	if err != nil {
		return errors.New("failed to listen: " + err.Error())
	}
	defer listener.Close()

	go func() {
		<-ctx.Done()
		log.InfoContext(ctx, "Shutting down server")
		listener.Close()
	}()

	if err := setIPTransparent(listener); err != nil {
		return err
	}

	log.InfoContext(ctx, "Starting server", "port", port)

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}

		go handleConn(ctx, log, conn)
	}
}

func handleConn(ctx context.Context, log *slog.Logger, c *net.TCPConn) {
	defer c.Close()

	addr := c.LocalAddr().(*net.TCPAddr)
	target := net.JoinHostPort(addr.IP[12:16].String(), strconv.Itoa(addr.Port))

	log.InfoContext(ctx, "connection", "src", c.RemoteAddr(), "dst", target)

	if err := utils.ProxyTCP(c, target); err != nil {
		log.ErrorContext(ctx, "failed to connect", "dst", target, "err", err)
	}
}

func setIPTransparent(listener *net.TCPListener) error {
	raw, err := listener.SyscallConn()
	if err != nil {
		return errors.New("failed to get raw listener: " + err.Error())
	}

	var sockErr error
	if err := raw.Control(func(fd uintptr) {
		sockErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TRANSPARENT, 1)
	}); err != nil {
		return errors.New("failed to control listener socket: " + err.Error())
	}
	if sockErr != nil {
		return errors.New("failed to set IP_TRANSPARENT: " + sockErr.Error())
	}

	return nil
}
