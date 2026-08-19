package peer

import (
	"errors"

	"github.com/xmasdev/Cantaloupe/engine/peer/messages"
)

type PeerSession struct {
	Connection *Connection

	PeerID [20]byte

	// Remote peer state.
	Choked bool

	// Whether we have told the remote peer that we're interested.
	Interested bool

	// Pieces currently available from the remote peer.
	RemoteBitfield Bitfield
}

func NewPeerSession(
	address string,
	infoHash [20]byte,
	peerID [20]byte,
) (*PeerSession, error) {
	connection, err := Connect(address)
	if err != nil {
		return nil, err
	}

	if err := Handshake(connection, infoHash, peerID); err != nil {
		connection.Close()
		return nil, err
	}

	session := &PeerSession{
		Connection: connection,
		PeerID:     connection.peerId,
		Choked:     true,
	}

	return session, nil
}

func (s *PeerSession) Close() error {
	if s.Connection == nil {
		return nil
	}

	return s.Connection.Close()
}

func (s *PeerSession) SendInterested() error {
	if s.Interested {
		return nil
	}

	err := s.Connection.WriteMessage(
		messages.InterestedMessage(),
	)
	if err != nil {
		return err
	}

	s.Interested = true

	return nil
}

func (s *PeerSession) ReadMessage() error {
	message, err := s.Connection.ReadMessage()
	if err != nil {
		return err
	}

	if message.KeepAlive {
		return nil
	}

	switch message.ID {
	case messages.Choke:
		s.Choked = true

	case messages.Unchoke:
		s.Choked = false

	case messages.Interested:
		// The remote peer is interested in us.
		// We don't need to maintain anything yet.

	case messages.NotInterested:
		// The remote peer is no longer interested.

	case messages.Have:
		have, err := messages.ParseHave(message)
		if err != nil {
			return err
		}

		s.RemoteBitfield.SetPiece(int(have.PieceIndex))

	case messages.Bitfield:
		bitfield, err := messages.ParseBitfield(message)
		if err != nil {
			return err
		}

		s.RemoteBitfield = bitfield.Bitfield

	case messages.Piece:
		// We'll handle downloaded blocks later.

	case messages.Request:
		// We'll handle upload requests later.

	case messages.Cancel:
		// We'll handle cancellations later.

	case messages.Port:
		// DHT support later.

	default:
		return errors.New("unknown peer message ID")
	}

	return nil
}
