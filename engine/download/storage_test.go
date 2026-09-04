package download

import (
	"bytes"
	"crypto/sha1"
	"os"
	"path/filepath"
	"testing"

	"github.com/xmasdev/Cantaloupe/engine/types"
)

func makeVerifiedPiece(
	t *testing.T,
	index int,
	data []byte,
	pieceLength int64,
) *Piece {
	t.Helper()

	hash := sha1.Sum(data)

	piece, err := NewPiece(index, len(data), hash)
	if err != nil {
		t.Fatalf("failed to create piece: %v", err)
	}

	// Fill the piece block-by-block.
	offset := 0
	for _, block := range piece.Blocks {
		end := offset + block.Length
		if end > len(data) {
			end = len(data)
		}

		if err := piece.SetBlock(block.Begin, data[offset:end]); err != nil {
			t.Fatalf("failed to set block: %v", err)
		}

		offset = end
	}

	return piece
}

func TestStorage_WritePiece_SingleFile(t *testing.T) {
	root := t.TempDir()

	data := []byte("hello world")

	info := &types.Info{
		Name:        "test.txt",
		Length:      int64(len(data)),
		PieceLength: int64(len(data)),
	}

	storage, err := NewStorage(root, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	piece := makeVerifiedPiece(t, 0, data, info.PieceLength)

	if err := storage.WritePiece(piece); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(root, "test.txt")

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !bytes.Equal(got, data) {
		t.Fatalf("expected %q, got %q", data, got)
	}
}

func TestStorage_WritePiece_SingleFileCorrectOffset(t *testing.T) {
	root := t.TempDir()

	pieceLength := int64(5)

	info := &types.Info{
		Name:        "test.txt",
		Length:      10,
		PieceLength: pieceLength,
	}

	storage, err := NewStorage(root, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	piece0Data := []byte("hello")
	piece1Data := []byte("world")

	piece0 := makeVerifiedPiece(t, 0, piece0Data, pieceLength)
	piece1 := makeVerifiedPiece(t, 1, piece1Data, pieceLength)

	if err := storage.WritePiece(piece1); err != nil {
		t.Fatalf("failed to write piece 1: %v", err)
	}

	if err := storage.WritePiece(piece0); err != nil {
		t.Fatalf("failed to write piece 0: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "test.txt"))
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	expected := []byte("helloworld")

	if !bytes.Equal(got, expected) {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestStorage_WritePiece_MultiFile(t *testing.T) {
	root := t.TempDir()

	info := &types.Info{
		Name: "torrent",
		Files: []types.File{
			{
				Length: 10,
				Path:   []string{"a.txt"},
			},
			{
				Length: 10,
				Path:   []string{"b.txt"},
			},
		},
		PieceLength: 20,
	}

	storage, err := NewStorage(root, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := []byte("01234567890123456789")

	piece := makeVerifiedPiece(t, 0, data, info.PieceLength)

	if err := storage.WritePiece(piece); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	a, err := os.ReadFile(filepath.Join(root, "torrent", "a.txt"))
	if err != nil {
		t.Fatalf("failed to read a.txt: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(root, "torrent", "b.txt"))
	if err != nil {
		t.Fatalf("failed to read b.txt: %v", err)
	}

	if !bytes.Equal(a, []byte("0123456789")) {
		t.Fatalf("unexpected a.txt contents: %q", a)
	}

	if !bytes.Equal(b, []byte("0123456789")) {
		t.Fatalf("unexpected b.txt contents: %q", b)
	}
}

func TestStorage_WritePiece_CrossesFileBoundary(t *testing.T) {
	root := t.TempDir()

	info := &types.Info{
		Name: "torrent",
		Files: []types.File{
			{
				Length: 10,
				Path:   []string{"a.txt"},
			},
			{
				Length: 10,
				Path:   []string{"b.txt"},
			},
		},
		PieceLength: 10,
	}

	storage, err := NewStorage(root, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Piece 0 occupies the first file completely.
	piece0 := makeVerifiedPiece(
		t,
		0,
		[]byte("AAAAAAAAAA"),
		info.PieceLength,
	)

	if err := storage.WritePiece(piece0); err != nil {
		t.Fatalf("failed to write piece 0: %v", err)
	}

	// Piece 1 occupies the second file completely.
	piece1 := makeVerifiedPiece(
		t,
		1,
		[]byte("BBBBBBBBBB"),
		info.PieceLength,
	)

	if err := storage.WritePiece(piece1); err != nil {
		t.Fatalf("failed to write piece 1: %v", err)
	}

	a, err := os.ReadFile(filepath.Join(root, "torrent", "a.txt"))
	if err != nil {
		t.Fatalf("failed to read a.txt: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(root, "torrent", "b.txt"))
	if err != nil {
		t.Fatalf("failed to read b.txt: %v", err)
	}

	if !bytes.Equal(a, []byte("AAAAAAAAAA")) {
		t.Fatalf("unexpected a.txt contents: %q", a)
	}

	if !bytes.Equal(b, []byte("BBBBBBBBBB")) {
		t.Fatalf("unexpected b.txt contents: %q", b)
	}
}

func TestStorage_WritePiece_IncompletePiece(t *testing.T) {
	root := t.TempDir()

	info := &types.Info{
		Name:        "test.txt",
		Length:      5,
		PieceLength: 5,
	}

	storage, err := NewStorage(root, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hash := sha1.Sum([]byte("hello"))

	piece, err := NewPiece(0, 5, hash)
	if err != nil {
		t.Fatalf("failed to create piece: %v", err)
	}

	err = storage.WritePiece(piece)

	if err == nil {
		t.Fatal("expected error for incomplete piece")
	}
}

func TestStorage_WritePiece_InvalidHash(t *testing.T) {
	root := t.TempDir()

	info := &types.Info{
		Name:        "test.txt",
		Length:      5,
		PieceLength: 5,
	}

	storage, err := NewStorage(root, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Hash is deliberately wrong.
	wrongHash := sha1.Sum([]byte("wrong"))

	piece, err := NewPiece(0, 5, wrongHash)
	if err != nil {
		t.Fatalf("failed to create piece: %v", err)
	}

	if err := piece.SetBlock(0, []byte("hello")); err != nil {
		t.Fatalf("failed to set block: %v", err)
	}

	err = storage.WritePiece(piece)

	if err == nil {
		t.Fatal("expected error for invalid piece hash")
	}
}

func TestNewStorage_NilInfo(t *testing.T) {
	root := t.TempDir()

	_, err := NewStorage(root, nil)

	if err == nil {
		t.Fatal("expected error for nil info")
	}
}

func TestNewStorage_EmptyRoot(t *testing.T) {
	info := &types.Info{
		Name:        "test.txt",
		Length:      5,
		PieceLength: 5,
	}

	_, err := NewStorage("", info)

	if err == nil {
		t.Fatal("expected error for empty storage root")
	}
}
