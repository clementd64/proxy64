package utils

import (
	"io"
	"net"
	"sync"
	"time"
)

func ProxyTCP(conn *net.TCPConn, target string, readers ...io.Reader) error {
	targetConn, err := net.DialTimeout("tcp", target, time.Second*5)
	if err != nil {
		return err
	}
	defer targetConn.Close()

	BidiCopy(conn, targetConn.(*net.TCPConn), readers...)
	return nil
}

func BidiCopy(conn1, conn2 *net.TCPConn, readers ...io.Reader) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		io.Copy(conn1, conn2)
		conn1.CloseWrite()
		wg.Done()
	}()

	go func() {
		if len(readers) > 0 {
			io.Copy(conn2, readers[0])
		} else {
			io.Copy(conn2, conn1)
		}
		conn2.CloseWrite()
		wg.Done()
	}()

	wg.Wait()
}
