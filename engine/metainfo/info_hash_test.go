package metainfo

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestGetInfoHash(t *testing.T) {
	path := filepath.Join("..", "sample", "test.torrent")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read torrent: %v", err)
	}

	hash, err := GetInfoHash(data)
	if err != nil {
		t.Fatalf("failed to get info hash: %v", err)
	}

	expected := "3c49c1da36ef82853221cb26ad4448e3d2ff2b13"
	got := hex.EncodeToString(hash[:])

	t.Logf("Info hash: %s", got)

	if got != expected {
		t.Fatalf(
			"unexpected info hash: expected %s, got %s",
			expected,
			got,
		)
	}
}
