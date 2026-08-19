package peer

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
)

type Message struct {
	ID        byte
	Payload   []byte
	KeepAlive bool
}

const (
	Choke         byte = 0
	Unchoke       byte = 1
	Interested    byte = 2
	NotInterested byte = 3
	Have          byte = 4
	Bitfield      byte = 5
	Request       byte = 6
	Piece         byte = 7
	Cancel        byte = 8
	Port          byte = 9
)

const maxMessageLength = 1 << 14 // 16 KiB

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

// Read and Write message

func (c *Connection) WriteMessage(message Message) error {
	var length uint32

	if message.KeepAlive {
		length = 0
	} else {
		length = uint32(1 + len(message.Payload))
	}

	buffer := make([]byte, 4+length)

	binary.BigEndian.PutUint32(buffer[:4], length)

	if !message.KeepAlive {
		buffer[4] = message.ID
		copy(buffer[5:], message.Payload)
	}

	return writeFull(c.conn, buffer)
}
func (c *Connection) ReadMessage() (Message, error) {
	var lengthBuffer [4]byte

	if _, err := io.ReadFull(c.conn, lengthBuffer[:]); err != nil {
		return Message{}, err
	}

	length := binary.BigEndian.Uint32(lengthBuffer[:])

	if length > maxMessageLength {
		return Message{}, errors.New("message length exceeds maximum message size")
	}

	if length == 0 {
		return Message{
			KeepAlive: true,
		}, nil
	}

	messageBuffer := make([]byte, length)

	if _, err := io.ReadFull(c.conn, messageBuffer); err != nil {
		return Message{}, err
	}

	return Message{
		ID:      messageBuffer[0],
		Payload: messageBuffer[1:],
	}, nil
}

func writeFull(conn net.Conn, data []byte) error {
	for len(data) > 0 {
		n, err := conn.Write(data)
		if err != nil {
			return err
		}

		data = data[n:]
	}

	return nil
}
