package peer

import (
	"net"
)

type Connection struct {
	conn   net.Conn
	peerId [20]byte
}

func Connect(address string) (*Connection, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	return &Connection{
		conn: conn,
	}, nil
}

func (c *Connection) Close() error {
	return c.conn.Close()
}
