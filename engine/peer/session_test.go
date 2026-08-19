package peer

import (
	"net"
	"testing"

	"github.com/xmasdev/Cantaloupe/engine/peer/messages"
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

	if err := session.ReadMessage(); err != nil {
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

	if err := session.ReadMessage(); err != nil {
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

	if err := session.ReadMessage(); err != nil {
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
	session.RemoteBitfield = make(Bitfield, 6)

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

	if err := session.ReadMessage(); err != nil {
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

	if err := session.ReadMessage(); err != nil {
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
