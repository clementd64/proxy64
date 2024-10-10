package utils

import (
	"io"
	"net"
	"sync"
	"time"
)

func ProxyTCP(conn *net.TCPConn, target string) error {
	targetConn, err := net.DialTimeout("tcp", target, time.Second*5)
	if err != nil {
		return err
	}
	defer targetConn.Close()

	BidiCopy(conn, targetConn.(*net.TCPConn))
	return nil
}

func BidiCopy(conn1, conn2 *net.TCPConn) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		io.Copy(conn1, conn2)
		conn1.CloseWrite()
		wg.Done()
	}()

	go func() {
		io.Copy(conn2, conn1)
		conn2.CloseWrite()
		wg.Done()
	}()

	wg.Wait()
}
