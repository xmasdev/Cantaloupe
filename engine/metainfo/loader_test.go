package metainfo

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	path := filepath.Join("..", "sample", "test.torrent")

	metadata, err := Load(path)
	if err != nil {
		t.Fatalf("failed to load torrent: %v", err)
	}

	if metadata == nil {
		t.Fatal("expected metadata, got nil")
	}

	t.Logf("Metadata: %+v", metadata)
	t.Logf("Info: %+v", metadata.Info)

	// Announce
	expectedAnnounce := "https://tracker.example.com/announce"

	if metadata.Announce != expectedAnnounce {
		t.Errorf(
			"expected announce %q, got %q",
			expectedAnnounce,
			metadata.Announce,
		)
	}

	// Announce list
	expectedAnnounceList := [][]string{
		{
			"https://tracker1.example.com/announce",
		},
		{
			"https://tracker2.example.com/announce",
			"https://tracker3.example.com/announce",
		},
	}

	if len(metadata.AnnounceList) != len(expectedAnnounceList) {
		t.Fatalf(
			"expected %d announce-list tiers, got %d",
			len(expectedAnnounceList),
			len(metadata.AnnounceList),
		)
	}

	for i, expectedTier := range expectedAnnounceList {
		gotTier := metadata.AnnounceList[i]

		if len(gotTier) != len(expectedTier) {
			t.Fatalf(
				"tier %d: expected %d trackers, got %d",
				i,
				len(expectedTier),
				len(gotTier),
			)
		}

		for j, expectedTracker := range expectedTier {
			if gotTier[j] != expectedTracker {
				t.Errorf(
					"tier %d tracker %d: expected %q, got %q",
					i,
					j,
					expectedTracker,
					gotTier[j],
				)
			}
		}
	}

	// Info
	info := metadata.Info

	if info.Name != "hello.txt" {
		t.Errorf("expected name %q, got %q", "hello.txt", info.Name)
	}

	if info.PieceLength != 16384 {
		t.Errorf(
			"expected piece length %d, got %d",
			16384,
			info.PieceLength,
		)
	}

	expectedPieceHash := "89cc388db3ecef162d14f5b82cf0021c099972a9"

	if len(info.Pieces) != 20 {
		t.Fatalf(
			"expected 20 bytes of piece hashes, got %d",
			len(info.Pieces),
		)
	}

	gotPieceHash := fmt.Sprintf("%x", info.Pieces)

	if gotPieceHash != expectedPieceHash {
		t.Errorf(
			"unexpected piece hash: expected %s, got %s",
			expectedPieceHash,
			gotPieceHash,
		)
	}

	if info.Length != 23 {
		t.Errorf(
			"expected length %d, got %d",
			23,
			info.Length,
		)
	}

	if len(info.Files) != 0 {
		t.Errorf(
			"expected no files for single-file torrent, got %d",
			len(info.Files),
		)
	}
}
