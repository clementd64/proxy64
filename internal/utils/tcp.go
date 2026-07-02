package utils

import (
	"context"
	"io"
	"net"
	"sync"
	"syscall"
	"time"
)

func proxyTCP(ctx context.Context, conn *net.TCPConn, target string, localAddr net.Addr, control func(network, address string, c syscall.RawConn) error, bufferedClientConn ...io.Reader) error {
	dialer := net.Dialer{
		Timeout:   time.Second * 5,
		LocalAddr: localAddr,
		Control:   control,
	}

	targetConn, err := dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		return err
	}
	defer targetConn.Close()

	BidiCopy(conn, targetConn.(*net.TCPConn), bufferedClientConn...)
	return nil
}

func ProxyTCP(ctx context.Context, conn *net.TCPConn, target string, bufferedClientConn ...io.Reader) error {
	return proxyTCP(ctx, conn, target, nil, nil, bufferedClientConn...)
}

func ProxyTCPFrom(ctx context.Context, conn *net.TCPConn, target string, localAddr net.Addr, bufferedClientConn ...io.Reader) error {
	return proxyTCP(ctx, conn, target, localAddr, func(network, address string, c syscall.RawConn) error {
		var sockErr error
		if err := c.Control(func(fd uintptr) {
			sockErr = syscall.SetsockoptInt(int(fd), syscall.SOL_IP, syscall.IP_FREEBIND, 1)
			if sockErr != nil {
				return
			}
			sockErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			if sockErr != nil {
				return
			}
		}); err != nil {
			return err
		}
		return sockErr
	}, bufferedClientConn...)

}

func BidiCopy(conn1, conn2 *net.TCPConn, bufferedClientConn ...io.Reader) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		io.Copy(conn1, conn2)
		conn1.CloseWrite()
		wg.Done()
	}()

	go func() {
		if len(bufferedClientConn) > 0 {
			io.Copy(conn2, io.MultiReader(bufferedClientConn[0], conn1))
		} else {
			io.Copy(conn2, conn1)
		}
		conn2.CloseWrite()
		wg.Done()
	}()

	wg.Wait()
}
