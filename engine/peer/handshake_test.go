package peer

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"
)

func TestHandshake(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create test listener: %v", err)
	}
	defer listener.Close()

	var infoHash [20]byte
	var ourPeerID [20]byte
	var remotePeerID [20]byte

	for i := range infoHash {
		infoHash[i] = byte(i)
	}

	for i := range ourPeerID {
		ourPeerID[i] = byte(i + 20)
	}

	for i := range remotePeerID {
		remotePeerID[i] = byte(i + 40)
	}

	serverDone := make(chan error, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()

		// Read client's handshake.
		clientHandshake := make([]byte, 68)

		if _, err := io.ReadFull(conn, clientHandshake); err != nil {
			serverDone <- err
			return
		}

		// Validate protocol.
		if clientHandshake[0] != 19 {
			serverDone <- errors.New("invalid handshake")
			return
		}

		if string(clientHandshake[1:20]) != "BitTorrent protocol" {
			serverDone <- errors.New("invalid handshake")
			return
		}

		// Validate info hash.
		if !bytes.Equal(
			clientHandshake[28:48],
			infoHash[:],
		) {
			serverDone <- errors.New("invalid handshake")
			return
		}

		// Validate our peer ID.
		if !bytes.Equal(
			clientHandshake[48:68],
			ourPeerID[:],
		) {
			serverDone <- errors.New("invalid handshake")
			return
		}

		// Construct server handshake.
		serverHandshake := make([]byte, 68)

		serverHandshake[0] = 19
		copy(serverHandshake[1:20], "BitTorrent protocol")
		copy(serverHandshake[28:48], infoHash[:])
		copy(serverHandshake[48:68], remotePeerID[:])

		_, err = conn.Write(serverHandshake)
		serverDone <- err
	}()

	conn, err := Connect(listener.Addr().String())
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer conn.Close()

	if err := Handshake(conn, infoHash, ourPeerID); err != nil {
		t.Fatalf("Handshake failed: %v", err)
	}

	if !bytes.Equal(conn.peerId[:], remotePeerID[:]) {
		t.Errorf(
			"wrong remote peer ID: expected %x, got %x",
			remotePeerID,
			conn.peerId,
		)
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake peer failed: %v", err)
	}
}
