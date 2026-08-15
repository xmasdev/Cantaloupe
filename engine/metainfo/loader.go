package metainfo

import (
	"os"

	"github.com/xmasdev/Cantaloupe/engine/types"
)

func Load(path string) (*types.TorrentMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return Parse(data)
}
