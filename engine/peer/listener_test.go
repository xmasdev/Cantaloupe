package peer

import (
	"net"
	"testing"
)

func TestListen(t *testing.T) {
	listener, err := Listen(0)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	port := listener.Port()

	if port <= 0 {
		t.Fatalf("expected a valid port, got %d", port)
	}

	t.Logf("listener is using port %d", port)
}

func TestListenPort(t *testing.T) {
	listener, err := Listen(0)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	actualPort := listener.listener.Addr().(*net.TCPAddr).Port

	if listener.Port() != actualPort {
		t.Errorf(
			"Port() returned %d, but listener is actually using %d",
			listener.Port(),
			actualPort,
		)
	}
}
