package peer

import (
	"errors"
	"fmt"

	"github.com/xmasdev/Cantaloupe/engine/peer/messages"
	"github.com/xmasdev/Cantaloupe/engine/types"
)

type PeerSession struct {
	Connection *Connection

	PeerID [20]byte

	// Remote peer state.
	Choked bool

	// Whether we have told the remote peer that we're interested.
	Interested bool

	// Pieces currently available from the remote peer.
	RemoteBitfield types.Bitfield
}

// NewPeerSessionFromConnection creates a session after the protocol
// handshake has been completed by the caller.
func NewPeerSessionFromConnection(connection *Connection) *PeerSession {
	if connection == nil {
		return nil
	}

	return &PeerSession{
		Connection: connection,
		PeerID:     connection.peerId,
		Choked:     true,
	}
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

func (p *PeerSession) ReadMessage() (messages.Message, error) {
	message, err := p.Connection.ReadMessage()
	if err != nil {
		return messages.Message{}, err
	}

	switch message.ID {
	case messages.Choke:
		p.Choked = true

	case messages.Unchoke:
		p.Choked = false

	case messages.Bitfield:
		data, err := messages.ParseBitfield(message)
		if err != nil {
			return messages.Message{}, err
		}
		p.RemoteBitfield = types.Bitfield(data.Bitfield)

	case messages.Have:
		data, err := messages.ParseHave(message)
		if err != nil {
			return messages.Message{}, err
		}
		p.RemoteBitfield.SetPiece(int(data.PieceIndex))
	}

	return message, nil
}

func (p *PeerSession) HasPiece(index int) bool {
	return p.RemoteBitfield.HasPiece(index)
}

func (p *PeerSession) WaitForBitfield() error {
	for {
		message, err := p.ReadMessage()
		if err != nil {
			return fmt.Errorf("failed while waiting for bitfield: %w", err)
		}

		if message.KeepAlive {
			continue
		}

		if message.ID == messages.Bitfield {
			return nil
		}
	}
}

func (p *PeerSession) WaitForUnchoke() error {
	for {
		message, err := p.ReadMessage()
		if err != nil {
			return fmt.Errorf(
				"failed while waiting for unchoke: %w",
				err,
			)
		}

		if message.KeepAlive {
			continue
		}

		if message.ID == messages.Unchoke {
			return nil
		}
	}
}

func (p *PeerSession) RequestBlock(pieceIndex, begin, length int) error {
	if pieceIndex < 0 {
		return errors.New("piece index cannot be negative")
	}

	if begin < 0 {
		return errors.New("block begin cannot be negative")
	}

	if length <= 0 {
		return errors.New("block length must be positive")
	}

	message := messages.RequestMessage(
		uint32(pieceIndex),
		uint32(begin),
		uint32(length),
	)

	return p.Connection.WriteMessage(message)
}
