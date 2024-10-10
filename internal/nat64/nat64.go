package nat64

import (
	"errors"
	"log/slog"
	"net"
	"strconv"
	"syscall"

	"github.com/clementd64/proxy64/internal/utils"
)

func Listen(port int) error {
	addr, err := net.ResolveTCPAddr("tcp6", net.JoinHostPort("::", strconv.Itoa(port)))
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP("tcp6", addr)
	if err != nil {
		return errors.New("failed to listen: " + err.Error())
	}
	defer listener.Close()

	if err := setIPTransparent(listener); err != nil {
		return err
	}

	slog.Info("proxy listening", "port", port)

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			return err
		}

		go handleConn(conn)
	}
}

func handleConn(c *net.TCPConn) {
	defer c.Close()

	addr := c.LocalAddr().(*net.TCPAddr)
	ip := addr.IP[12:16]

	slog.Info("connection", "src", c.RemoteAddr(), "dst", ip.String(), "dport", addr.Port)

	if err := utils.ProxyTCP(c, net.JoinHostPort(ip.String(), strconv.Itoa(addr.Port))); err != nil {
		slog.Error("failed to connect", "dst", ip, "dport", addr.Port, "err", err)
	}
}

func setIPTransparent(listener *net.TCPListener) error {
	fd, err := listener.File()
	if err != nil {
		return errors.New("failed to get file descriptor: " + err.Error())
	}
	defer fd.Close()

	if err := syscall.SetsockoptInt(int(fd.Fd()), syscall.IPPROTO_IP, syscall.IP_TRANSPARENT, 1); err != nil {
		return errors.New("failed to set IP_TRANSPARENT: " + err.Error())
	}

	return nil
}
