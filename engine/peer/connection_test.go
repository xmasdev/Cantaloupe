package peer

import (
	"net"
	"testing"
)

func TestConnect(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create test listener: %v", err)
	}
	defer listener.Close()

	address := listener.Addr().String()

	// Accept the connection in the background.
	accepted := make(chan error, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			accepted <- err
			return
		}

		conn.Close()
		accepted <- nil
	}()

	conn, err := Connect(address)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer conn.Close()

	if err := <-accepted; err != nil {
		t.Fatalf("server failed to accept connection: %v", err)
	}
}
