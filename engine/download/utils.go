package download

import (
	"errors"

	"github.com/xmasdev/Cantaloupe/engine/types"
)

func totalLength(info *types.Info) (int64, error) {
	if len(info.Files) == 0 {
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
