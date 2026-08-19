package peer

import (
	"encoding/binary"
	"errors"
	"io"
	"net"

	"github.com/xmasdev/Cantaloupe/engine/peer/messages"
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

func (c *Connection) WriteMessage(message messages.Message) error {
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
func (c *Connection) ReadMessage() (messages.Message, error) {
	var lengthBuffer [4]byte

	if _, err := io.ReadFull(c.conn, lengthBuffer[:]); err != nil {
		return messages.Message{}, err
	}

	length := binary.BigEndian.Uint32(lengthBuffer[:])

	if length > maxMessageLength {
		return messages.Message{}, errors.New("message length exceeds maximum message size")
	}

	if length == 0 {
		return messages.Message{
			KeepAlive: true,
		}, nil
	}

	messageBuffer := make([]byte, length)

	if _, err := io.ReadFull(c.conn, messageBuffer); err != nil {
		return messages.Message{}, err
	}

	return messages.Message{
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
