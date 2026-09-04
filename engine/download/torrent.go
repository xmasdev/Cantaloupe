package download

import (
	"errors"
	"fmt"

	"github.com/xmasdev/Cantaloupe/engine/peer/messages"
	"github.com/xmasdev/Cantaloupe/engine/types"
)

type TorrentDownload struct {
	Pieces []Piece
}

func NewTorrentDownload(info *types.Info) (*TorrentDownload, error) {
	if info == nil {
		return nil, errors.New("torrent info cannot be nil")
	}

	total, err := totalLength(info)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate total length: %w", err)
	}
	if info.PieceLength <= 0 {
		return nil, errors.New("piece length must be positive")
	}

	if total < 0 {
		return nil, errors.New("torrent length cannot be negative")
	}

	if len(info.Pieces)%20 != 0 {
		return nil, errors.New("pieces data must be a multiple of 20 bytes")
	}

	pieceCount := len(info.Pieces) / 20

	if pieceCount == 0 {
		return nil, errors.New("torrent contains no pieces")
	}

	expectedPieceCount := (total + info.PieceLength - 1) / info.PieceLength

	if pieceCount != int(expectedPieceCount) {
		return nil, fmt.Errorf(
			"piece count mismatch: metadata contains %d pieces, expected %d",
			pieceCount,
			expectedPieceCount,
		)
	}

	pieces := make([]Piece, 0, pieceCount)

	for i := 0; i < pieceCount; i++ {
		pieceLength := info.PieceLength

		// Last piece can be shorter.
		if int64(i+1)*pieceLength > total {
			pieceLength = total - int64(i)*pieceLength
		}

		var hash [20]byte
		copy(hash[:], info.Pieces[i*20:(i+1)*20])

		piece, err := NewPiece(
			i,
			int(pieceLength),
			hash,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create piece %d: %w",
				i,
				err,
			)
		}

		pieces = append(pieces, *piece)
	}

	return &TorrentDownload{
		Pieces: pieces,
	}, nil
}

func (t *TorrentDownload) NextMissingPiece() *Piece {
	for i := range t.Pieces {
		piece := &t.Pieces[i]
		if !piece.Complete() {
			return piece
		}
	}

	return nil
}

func (t *TorrentDownload) NextPieceForPeer(bitfield types.Bitfield) *Piece {
	for i := range t.Pieces {
		piece := &t.Pieces[i]
		if piece.Complete() {
			continue
		}

		if bitfield.HasPiece(piece.Index) {
			return piece
		}
	}

	return nil
}

func (t *TorrentDownload) HandlePiece(data messages.PieceData) error {
	if uint64(data.PieceIndex) >= uint64(len(t.Pieces)) {
		return fmt.Errorf("piece index out of range: %d", data.PieceIndex)
	}

	piece := &t.Pieces[data.PieceIndex]

	if err := piece.SetBlock(int(data.Begin), data.Block); err != nil {
		return fmt.Errorf(
			"failed to set block for piece %d: %w",
			data.PieceIndex,
			err,
		)
	}

	return nil
}
