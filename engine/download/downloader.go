package download

import (
	"errors"
	"fmt"

	"github.com/xmasdev/Cantaloupe/engine/peer"
	"github.com/xmasdev/Cantaloupe/engine/peer/messages"
)

func DownloadPiece(
	torrent *TorrentDownload,
	session *peer.PeerSession,
) error {
	if torrent == nil {
		return errors.New("torrent download cannot be nil")
	}

	if session == nil {
		return errors.New("peer session cannot be nil")
	}

	if session.Choked {
		return errors.New("peer is choked")
	}

	piece := torrent.NextPieceForPeer(session.RemoteBitfield)
	if piece == nil {
		return errors.New("peer has no missing pieces")
	}

	return downloadPiece(torrent, session, piece)
}

func DownloadPieceFor(
	torrent *TorrentDownload,
	session *peer.PeerSession,
	piece *Piece,
) error {
	if torrent == nil {
		return errors.New("torrent download cannot be nil")
	}
	if session == nil {
		return errors.New("peer session cannot be nil")
	}
	if piece == nil {
		return errors.New("piece cannot be nil")
	}
	if session.Choked {
		return errors.New("peer is choked")
	}

	return downloadPiece(torrent, session, piece)
}

func downloadPiece(
	torrent *TorrentDownload,
	session *peer.PeerSession,
	piece *Piece,
) error {

	for _, block := range piece.Blocks {
		if len(block.Data) == block.Length {
			continue
		}

		err := session.RequestBlock(
			piece.Index,
			block.Begin,
			block.Length,
		)
		if err != nil {
			return fmt.Errorf(
				"failed to request piece %d block at %d: %w",
				piece.Index,
				block.Begin,
				err,
			)
		}

		for {
			message, err := session.ReadMessage()
			if err != nil {
				return fmt.Errorf(
					"failed to read response for piece %d block at %d: %w",
					piece.Index,
					block.Begin,
					err,
				)
			}

			if message.KeepAlive {
				continue
			}

			if message.ID != messages.Piece {
				continue
			}

			pieceData, err := messages.ParsePiece(message)
			if err != nil {
				return fmt.Errorf(
					"failed to parse piece message: %w",
					err,
				)
			}

			if pieceData.PieceIndex != uint32(piece.Index) {
				continue
			}

			if pieceData.Begin != uint32(block.Begin) {
				continue
			}

			if err := torrent.HandlePiece(pieceData); err != nil {
				return fmt.Errorf(
					"failed to handle piece %d block at %d: %w",
					piece.Index,
					block.Begin,
					err,
				)
			}

			break
		}
	}

	if !piece.Complete() {
		return fmt.Errorf(
			"piece %d is still incomplete",
			piece.Index,
		)
	}

	valid, err := piece.Verify()
	if err != nil {
		return fmt.Errorf(
			"failed to verify piece %d: %w",
			piece.Index,
			err,
		)
	}

	if !valid {
		return fmt.Errorf(
			"piece %d failed SHA-1 verification",
			piece.Index,
		)
	}

	return nil
}

func DownloadTorrent(
	torrent *TorrentDownload,
	session *peer.PeerSession,
) error {
	if torrent == nil {
		return errors.New("torrent download cannot be nil")
	}

	if session == nil {
		return errors.New("peer session cannot be nil")
	}

	for {
		piece := torrent.NextPieceForPeer(session.RemoteBitfield)
		if piece == nil {
			break
		}

		if err := DownloadPiece(torrent, session); err != nil {
			return fmt.Errorf(
				"failed to download piece %d: %w",
				piece.Index,
				err,
			)
		}
	}

	return nil
}
