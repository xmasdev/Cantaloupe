package peer

import (
	"fmt"
	"net"
)

type Listener struct {
	listener net.Listener
	port     int
}

func Listen(port int) (*Listener, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port

	return &Listener{
		listener: listener,
		port:     actualPort,
	}, nil
}

func (l *Listener) Port() int {
	return l.port
}

func (l *Listener) Close() error {
	return l.listener.Close()
}
