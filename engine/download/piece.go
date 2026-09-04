package download

import (
	"crypto/sha1"
	"errors"
)

const BlockSize = 16 * 1024

type Piece struct {
	Index  int
	Length int
	Hash   [20]byte
	Blocks []Block
}

func NewPiece(
	index int,
	length int,
	hash [20]byte,
) (*Piece, error) {
	if index < 0 {
		return nil, errors.New("piece index cannot be negative")
	}

	if length <= 0 {
		return nil, errors.New("piece length must be positive")
	}

	piece := &Piece{
		Index:  index,
		Length: length,
		Hash:   hash,
	}

	piece.Blocks = makeBlocks(length)

	return piece, nil
}

func makeBlocks(pieceLength int) []Block {
	blockCount := (pieceLength + BlockSize - 1) / BlockSize

	blocks := make([]Block, 0, blockCount)

	for begin := 0; begin < pieceLength; begin += BlockSize {
		length := BlockSize

		remaining := pieceLength - begin
		if remaining < length {
			length = remaining
		}

		blocks = append(blocks, Block{
			Begin:  begin,
			Length: length,
		})
	}

	return blocks
}

func (p *Piece) Complete() bool {
	for _, block := range p.Blocks {
		if len(block.Data) != block.Length {
			return false
		}
	}

	return true
}

func (p *Piece) Data() ([]byte, error) {
	if !p.Complete() {
		return nil, errors.New("piece is incomplete")
	}

	data := make([]byte, p.Length)

	for _, block := range p.Blocks {
		copy(data[block.Begin:], block.Data)
	}

	return data, nil
}

func (p *Piece) Verify() (bool, error) {
	data, err := p.Data()
	if err != nil {
		return false, err
	}

	hash := sha1.Sum(data)

	return hash == p.Hash, nil
}

func (p *Piece) SetBlock(begin int, data []byte) error {
	if begin < 0 {
		return errors.New("block begin cannot be negative")
	}

	if begin >= p.Length {
		return errors.New("block begin is outside piece")
	}

	if len(data) == 0 {
		return errors.New("block data cannot be empty")
	}

	if begin+len(data) > p.Length {
		return errors.New("block extends beyond piece")
	}

	for i := range p.Blocks {
		block := &p.Blocks[i]

		if block.Begin != begin {
			continue
		}

		if len(data) != block.Length {
			return errors.New("block has incorrect length")
		}

		block.Data = data

		return nil
	}

	return errors.New("block begin does not match any block")
}
