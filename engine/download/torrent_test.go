package download

import (
	"testing"

	"github.com/xmasdev/Cantaloupe/engine/peer/messages"
	"github.com/xmasdev/Cantaloupe/engine/types"
)

func TestNewTorrentDownload_SingleFile(t *testing.T) {
	info := &types.Info{
		Length:      50000,
		PieceLength: 16384,
		Pieces:      make([]byte, 80), // 4 hashes
	}

	download, err := NewTorrentDownload(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(download.Pieces) != 4 {
		t.Fatalf("expected 4 pieces, got %d", len(download.Pieces))
	}

	expectedLengths := []int{16384, 16384, 16384, 848}

	for i, expected := range expectedLengths {
		if download.Pieces[i].Length != expected {
			t.Errorf(
				"piece %d: expected length %d, got %d",
				i,
				expected,
				download.Pieces[i].Length,
			)
		}

		if download.Pieces[i].Index != i {
			t.Errorf(
				"piece %d: expected index %d, got %d",
				i,
				i,
				download.Pieces[i].Index,
			)
		}
	}
}

func TestNewTorrentDownload_ExactDivision(t *testing.T) {
	info := &types.Info{
		Length:      32768,
		PieceLength: 16384,
		Pieces:      make([]byte, 40), // 2 hashes
	}

	download, err := NewTorrentDownload(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(download.Pieces) != 2 {
		t.Fatalf("expected 2 pieces, got %d", len(download.Pieces))
	}

	for i, piece := range download.Pieces {
		if piece.Length != 16384 {
			t.Errorf(
				"piece %d: expected length 16384, got %d",
				i,
				piece.Length,
			)
		}
	}
}

func TestNewTorrentDownload_MultiFile(t *testing.T) {
	info := &types.Info{
		PieceLength: 16384,
		Pieces:      make([]byte, 80), // 4 hashes
		Files: []types.File{
			{Length: 20000},
			{Length: 20000},
			{Length: 10000},
		},
	}

	download, err := NewTorrentDownload(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Total = 50,000 bytes.
	// Pieces = 16,384 + 16,384 + 16,384 + 848.
	expectedLengths := []int{16384, 16384, 16384, 848}

	if len(download.Pieces) != len(expectedLengths) {
		t.Fatalf(
			"expected %d pieces, got %d",
			len(expectedLengths),
			len(download.Pieces),
		)
	}

	for i, expected := range expectedLengths {
		if download.Pieces[i].Length != expected {
			t.Errorf(
				"piece %d: expected length %d, got %d",
				i,
				expected,
				download.Pieces[i].Length,
			)
		}
	}
}

func TestNewTorrentDownload_Hashes(t *testing.T) {
	pieces := make([]byte, 40)

	// Give each piece a recognizable hash.
	for i := 0; i < 20; i++ {
		pieces[i] = byte(i)
		pieces[20+i] = byte(i + 20)
	}

	info := &types.Info{
		Length:      20000,
		PieceLength: 10000,
		Pieces:      pieces,
	}

	download, err := NewTorrentDownload(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i := 0; i < 20; i++ {
		if download.Pieces[0].Hash[i] != byte(i) {
			t.Errorf(
				"piece 0 hash byte %d: expected %d, got %d",
				i,
				i,
				download.Pieces[0].Hash[i],
			)
		}

		if download.Pieces[1].Hash[i] != byte(i+20) {
			t.Errorf(
				"piece 1 hash byte %d: expected %d, got %d",
				i,
				i+20,
				download.Pieces[1].Hash[i],
			)
		}
	}
}

func TestNewTorrentDownload_InvalidPieceLength(t *testing.T) {
	info := &types.Info{
		Length:      10000,
		PieceLength: 0,
		Pieces:      make([]byte, 20),
	}

	_, err := NewTorrentDownload(info)

	if err == nil {
		t.Fatal("expected error for zero piece length")
	}
}

func TestNewTorrentDownload_NegativeLength(t *testing.T) {
	info := &types.Info{
		Length:      -1,
		PieceLength: 16384,
		Pieces:      make([]byte, 20),
	}

	_, err := NewTorrentDownload(info)

	if err == nil {
		t.Fatal("expected error for negative torrent length")
	}
}

func TestNewTorrentDownload_InvalidPiecesLength(t *testing.T) {
	info := &types.Info{
		Length:      10000,
		PieceLength: 16384,
		Pieces:      make([]byte, 21),
	}

	_, err := NewTorrentDownload(info)

	if err == nil {
		t.Fatal("expected error when pieces length is not a multiple of 20")
	}
}

func TestNewTorrentDownload_NoPieces(t *testing.T) {
	info := &types.Info{
		Length:      10000,
		PieceLength: 16384,
		Pieces:      nil,
	}

	_, err := NewTorrentDownload(info)

	if err == nil {
		t.Fatal("expected error when torrent contains no pieces")
	}
}

func TestNewTorrentDownload_PieceCountMismatch(t *testing.T) {
	info := &types.Info{
		Length:      50000,
		PieceLength: 16384,
		Pieces:      make([]byte, 60), // 3 hashes, but 4 expected
	}

	_, err := NewTorrentDownload(info)

	if err == nil {
		t.Fatal("expected error for piece count mismatch")
	}
}

func TestNewTorrentDownload_CreatesBlocks(t *testing.T) {
	info := &types.Info{
		Length:      20000,
		PieceLength: 20000,
		Pieces:      make([]byte, 20),
	}

	download, err := NewTorrentDownload(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	piece := download.Pieces[0]

	// 20,000-byte piece with 16 KiB blocks:
	// block 0 = 16,384
	// block 1 = 3,616
	if len(piece.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(piece.Blocks))
	}

	if piece.Blocks[0].Begin != 0 {
		t.Errorf("expected first block begin 0, got %d", piece.Blocks[0].Begin)
	}

	if piece.Blocks[0].Length != 16384 {
		t.Errorf(
			"expected first block length 16384, got %d",
			piece.Blocks[0].Length,
		)
	}

	if piece.Blocks[1].Begin != 16384 {
		t.Errorf(
			"expected second block begin 16384, got %d",
			piece.Blocks[1].Begin,
		)
	}

	if piece.Blocks[1].Length != 3616 {
		t.Errorf(
			"expected second block length 3616, got %d",
			piece.Blocks[1].Length,
		)
	}
}

func TestTorrentDownload_HandlePiece(t *testing.T) {
	info := &types.Info{
		Length:      20000,
		PieceLength: 20000,
		Pieces:      make([]byte, 20),
	}

	download, err := NewTorrentDownload(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	block := make([]byte, 16384)
	for i := range block {
		block[i] = byte(i % 256)
	}

	data := messages.PieceData{
		PieceIndex: 0,
		Begin:      0,
		Block:      block,
	}

	err = download.HandlePiece(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	piece := download.Pieces[0]

	if len(piece.Blocks[0].Data) != 16384 {
		t.Fatalf(
			"expected block length 16384, got %d",
			len(piece.Blocks[0].Data),
		)
	}

	for i := range block {
		if piece.Blocks[0].Data[i] != block[i] {
			t.Fatalf("block data differs at byte %d", i)
		}
	}

	if piece.Complete() {
		t.Fatal("piece should not be complete yet")
	}
}
