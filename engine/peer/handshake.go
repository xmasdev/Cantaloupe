package peer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

func Handshake(conn *Connection, infoHash [20]byte, peerID [20]byte) error {
	message := make([]byte, 68)
	message[0] = 19
	copy(message[1:20], "BitTorrent protocol")
	copy(message[28:48], infoHash[:])
	copy(message[48:68], peerID[:])

	// writing handshake request
	n, err := conn.conn.Write(message)
	if err != nil {
		return err
	}

	if n != len(message) {
		return io.ErrShortWrite
	}

	// reading handshake response
	buffer := make([]byte, 68)

	if _, err := io.ReadFull(conn.conn, buffer); err != nil {
		return err
	}

	// validate response
	if buffer[0] != 19 {
		return errors.New("Invalid protocol string length recieved in handshake")
	}
	if string(buffer[1:20]) != "BitTorrent protocol" {
		return fmt.Errorf("Invalid protocol name, %s", string(buffer[1:20]))
	}
	if !bytes.Equal(buffer[28:48], infoHash[:]) {
		return errors.New("wrong info hash recieved")
	}
	copy(conn.peerId[:], buffer[48:68])
	return nil
}
