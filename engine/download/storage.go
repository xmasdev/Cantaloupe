package download

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/xmasdev/Cantaloupe/engine/types"
)

type Storage struct {
	root         string
	info         *types.Info
	verifyPieces atomic.Bool
}

func NewStorage(root string, info *types.Info) (*Storage, error) {
	if root == "" {
		return nil, errors.New("storage root cannot be empty")
	}

	if info == nil {
		return nil, errors.New("torrent info cannot be nil")
	}

	if len(info.Files) == 0 {
		if info.Length < 0 {
			return nil, errors.New("torrent length cannot be negative")
		}
	} else {
		for _, file := range info.Files {
			if file.Length < 0 {
				return nil, errors.New("file length cannot be negative")
			}
		}
	}

	storage := &Storage{
		root: root,
		info: info,
	}
	storage.verifyPieces.Store(true)
	return storage, nil
}

func (s *Storage) SetVerifyPieces(enabled bool) { s.verifyPieces.Store(enabled) }
func (s *Storage) VerifyPieces() bool           { return s.verifyPieces.Load() }

func (s *Storage) WritePiece(piece *Piece) error {
	if piece == nil {
		return errors.New("piece cannot be nil")
	}

	if !piece.Complete() {
		return fmt.Errorf("piece %d is incomplete", piece.Index)
	}

	if s.verifyPieces.Load() {
		valid, err := piece.Verify()
		if err != nil {
			return fmt.Errorf("failed to verify piece %d: %w", piece.Index, err)
		}
		if !valid {
			return fmt.Errorf("piece %d failed SHA-1 verification", piece.Index)
		}
	}

	data, err := piece.Data()
	if err != nil {
		return fmt.Errorf("failed to assemble piece %d: %w", piece.Index, err)
	}

	offset := int64(piece.Index) * s.info.PieceLength

	if len(s.info.Files) == 0 {
		return s.writeSingleFile(data, offset)
	}

	return s.writeMultiFile(data, offset)
}

func (s *Storage) writeSingleFile(data []byte, offset int64) error {
	path := filepath.Join(s.root, s.info.Name)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_RDWR,
		0644,
	)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteAt(data, offset); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (s *Storage) writeMultiFile(data []byte, offset int64) error {
	remaining := data
	pieceOffset := offset
	fileStart := int64(0)

	for _, fileInfo := range s.info.Files {
		if len(remaining) == 0 {
			break
		}

		fileEnd := fileStart + fileInfo.Length

		// This file is entirely before the piece.
		if pieceOffset >= fileEnd {
			fileStart = fileEnd
			continue
		}

		// Offset inside this file where we start writing.
		fileOffset := int64(0)
		if pieceOffset > fileStart {
			fileOffset = pieceOffset - fileStart
		}

		writable := fileInfo.Length - fileOffset
		if writable > int64(len(remaining)) {
			writable = int64(len(remaining))
		}

		path := filepath.Join(s.root, s.info.Name)

		for _, component := range fileInfo.Path {
			path = filepath.Join(path, component)
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		file, err := os.OpenFile(
			path,
			os.O_CREATE|os.O_RDWR,
			0644,
		)
		if err != nil {
			return err
		}

		_, err = file.WriteAt(
			remaining[:writable],
			fileOffset,
		)

		closeErr := file.Close()

		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		if closeErr != nil {
			return closeErr
		}

		remaining = remaining[writable:]
		pieceOffset += writable
		fileStart = fileEnd
	}

	if len(remaining) != 0 {
		return errors.New("piece extends beyond torrent files")
	}

	return nil
}
