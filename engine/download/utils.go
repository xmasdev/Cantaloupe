package download

import (
	"errors"

	"github.com/xmasdev/Cantaloupe/engine/types"
)

func totalLength(info *types.Info) (int64, error) {
	if info == nil {
		return 0, errors.New("torrent info cannot be nil")
	}

	if len(info.Files) == 0 {
		if info.Length < 0 {
			return 0, errors.New("torrent length cannot be negative")
		}
		return info.Length, nil
	}

	var total int64
	for _, file := range info.Files {
		if file.Length < 0 {
			return 0, errors.New("file length cannot be negative")
		}
		total += file.Length
	}

	return total, nil
}

func (t *TorrentDownload) TotalLength() int64 {
	if t == nil {
		return 0
	}
	var total int64

	for _, piece := range t.Pieces {
		total += int64(piece.Length)
	}

	return total
}
