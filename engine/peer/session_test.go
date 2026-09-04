package peer

import (
	"fmt"
	"net"
	"testing"

	"github.com/xmasdev/Cantaloupe/engine/peer/messages"
	"github.com/xmasdev/Cantaloupe/engine/types"
)

func newTestSession(t *testing.T) (*PeerSession, net.Conn) {
	t.Helper()

	client, server := net.Pipe()

	session := &PeerSession{
		Connection: &Connection{
			conn: server,
		},
		Choked: true,
	}

	t.Cleanup(func() {
		client.Close()
		server.Close()
	})

	return session, client
}

func sendMessage(t *testing.T, conn net.Conn, message messages.Message) {
	t.Helper()

	connection := &Connection{
		conn: conn,
	}

	if err := connection.WriteMessage(message); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}
}

func TestPeerSessionChoke(t *testing.T) {
	session, remote := newTestSession(t)

	// Session starts choked.
	if !session.Choked {
		t.Fatal("expected session to start choked")
	}

	done := make(chan error, 1)

	go func() {
		sendMessage(t, remote, messages.ChokeMessage())
		done <- nil
	}()

	if _, err := session.ReadMessage(); err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if err := <-done; err != nil {
		t.Fatal(err)
	}

	if !session.Choked {
		t.Fatal("expected session to be choked")
	}
}

func TestPeerSessionUnchoke(t *testing.T) {
	session, remote := newTestSession(t)

	done := make(chan error, 1)

	go func() {
		sendMessage(t, remote, messages.UnchokeMessage())
		done <- nil
	}()

	if _, err := session.ReadMessage(); err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if err := <-done; err != nil {
		t.Fatal(err)
	}

	if session.Choked {
		t.Fatal("expected session to be unchoked")
	}
}

func TestPeerSessionBitfield(t *testing.T) {
	session, remote := newTestSession(t)

	bitfield := []byte{
		0b10110000,
		0b01010000,
	}

	done := make(chan error, 1)

	go func() {
		sendMessage(
			t,
			remote,
			messages.BitfieldMessage(bitfield),
		)
		done <- nil
	}()

	if _, err := session.ReadMessage(); err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if err := <-done; err != nil {
		t.Fatal(err)
	}

	if !session.RemoteBitfield.HasPiece(0) {
		t.Error("expected peer to have piece 0")
	}

	if session.RemoteBitfield.HasPiece(1) {
		t.Error("expected peer to not have piece 1")
	}

	if !session.RemoteBitfield.HasPiece(2) {
		t.Error("expected peer to have piece 2")
	}

	if !session.RemoteBitfield.HasPiece(3) {
		t.Error("expected peer to have piece 3")
	}

	if session.RemoteBitfield.HasPiece(8) {
		t.Error("expected peer to not have piece 8")
	}

	if !session.RemoteBitfield.HasPiece(9) {
		t.Error("expected peer to have piece 9")
	}
}

func TestPeerSessionHave(t *testing.T) {
	session, remote := newTestSession(t)

	// Initially the peer doesn't have piece 42.
	session.RemoteBitfield = make(types.Bitfield, 6)

	if session.RemoteBitfield.HasPiece(42) {
		t.Fatal("piece 42 should not initially be available")
	}

	done := make(chan error, 1)

	go func() {
		sendMessage(
			t,
			remote,
			messages.HaveMessage(42),
		)
		done <- nil
	}()

	if _, err := session.ReadMessage(); err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if err := <-done; err != nil {
		t.Fatal(err)
	}

	if !session.RemoteBitfield.HasPiece(42) {
		t.Fatal("expected peer to have piece 42 after HAVE")
	}
}

func TestPeerSessionKeepAlive(t *testing.T) {
	session, remote := newTestSession(t)

	done := make(chan error, 1)

	go func() {
		sendMessage(
			t,
			remote,
			messages.KeepAliveMessage(),
		)
		done <- nil
	}()

	if _, err := session.ReadMessage(); err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if err := <-done; err != nil {
		t.Fatal(err)
	}

	// Keep-alive should not change peer state.
	if !session.Choked {
		t.Error("keep-alive unexpectedly changed choke state")
	}
}

func TestPeerSession_RequestBlock(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	session := &PeerSession{
		Connection: &Connection{
			conn: client,
		},
	}

	errCh := make(chan error, 1)

	go func() {
		message, err := (&Connection{conn: server}).ReadMessage()
		if err != nil {
			errCh <- err
			return
		}

		request, err := messages.ParseRequest(message)
		if err != nil {
			errCh <- err
			return
		}

		if request.PieceIndex != 3 {
			errCh <- fmt.Errorf(
				"expected piece index 3, got %d",
				request.PieceIndex,
			)
			return
		}

		if request.Begin != 16384 {
			errCh <- fmt.Errorf(
				"expected begin 16384, got %d",
				request.Begin,
			)
			return
		}

		if request.Length != 16384 {
			errCh <- fmt.Errorf(
				"expected length 16384, got %d",
				request.Length,
			)
			return
		}

		errCh <- nil
	}()

	err := session.RequestBlock(3, 16384, 16384)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestPeerSession_RequestBlock_InvalidArguments(t *testing.T) {
	client, _ := net.Pipe()
	defer client.Close()

	session := &PeerSession{
		Connection: &Connection{
			conn: client,
		},
	}

	tests := []struct {
		name       string
		pieceIndex int
		begin      int
		length     int
	}{
		{
			name:       "negative piece index",
			pieceIndex: -1,
			begin:      0,
			length:     16384,
		},
		{
			name:       "negative begin",
			pieceIndex: 0,
			begin:      -1,
			length:     16384,
		},
		{
			name:       "zero length",
			pieceIndex: 0,
			begin:      0,
			length:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := session.RequestBlock(
				tt.pieceIndex,
				tt.begin,
				tt.length,
			)

			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
