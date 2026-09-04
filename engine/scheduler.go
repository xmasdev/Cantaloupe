package engine

import (
	"sync"

	"github.com/xmasdev/Cantaloupe/engine/download"
	"github.com/xmasdev/Cantaloupe/engine/peer"
)

type pieceScheduler struct {
	mu sync.Mutex

	torrent *download.TorrentDownload
	active  map[int]bool
}

func newPieceScheduler(
	torrent *download.TorrentDownload,
) *pieceScheduler {
	return &pieceScheduler{
		torrent: torrent,
		active:  make(map[int]bool),
	}
}

func (s *pieceScheduler) Next(
	session *peer.PeerSession,
) *download.Piece {
	if session == nil || s == nil || s.torrent == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var selected *download.Piece
	bestRarity := int(^uint(0) >> 1)

	for i := range s.torrent.Pieces {
		piece := &s.torrent.Pieces[i]
		if piece.Complete() {
			continue
		}

		if s.active[piece.Index] {
			continue
		}

		if !session.HasPiece(piece.Index) {
			continue
		}

		// All currently available pieces have equal rarity because the
		// scheduler does not own the engine's peer set. Keep the stable
		// first-match selection while reserving the piece atomically.
		if bestRarity > 0 {
			bestRarity = 0
			selected = piece
		}
	}

	if selected != nil {
		s.active[selected.Index] = true
	}

	return selected
}

func (s *pieceScheduler) pieceRarity(index int) int {
	// This function needs access to the Engine's peers,
	// so don't use this version yet.
	return 0
}

func (s *pieceScheduler) Release(index int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.active, index)
}
