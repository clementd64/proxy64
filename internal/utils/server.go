package utils

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"syscall"
)

type TCPServer struct {
	Network string
	Addr    string
	Handler func(context.Context, *slog.Logger, *net.TCPConn)
	Control func(listener *net.TCPListener) error
}

func ListenTCP(ctx context.Context, log *slog.Logger, server TCPServer) error {
	listener, err := net.Listen(server.Network, server.Addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	if server.Control != nil {
		if err := server.Control(listener.(*net.TCPListener)); err != nil {
			return err
		}
	}

	go func() {
		<-ctx.Done()
		log.InfoContext(ctx, "Shutting down server")
		listener.Close()
	}()

	log.InfoContext(ctx, "Starting server", "addr", server.Addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		go func() {
			defer conn.Close()
			server.Handler(ctx, log, conn.(*net.TCPConn))
		}()
	}
}

func ListenerIPTransparent(listener *net.TCPListener) error {
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
