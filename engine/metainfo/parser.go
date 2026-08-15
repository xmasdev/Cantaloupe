package metainfo

import (
	"errors"

	"github.com/xmasdev/Cantaloupe/engine/bencode"
	"github.com/xmasdev/Cantaloupe/engine/types"
)

func Parse(data []byte) (*types.TorrentMetadata, error) {
	decoded, err := bencode.Decode(data)
	if err != nil {
		return nil, err
	}

	root, ok := decoded.(map[string]any)
	if !ok {
		return nil, errors.New("root is not a dictionary")
	}

	var metadata types.TorrentMetadata
	err = extractRequiredFields(root, &metadata)
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}

func extractRequiredFields(root map[string]any, metadata *types.TorrentMetadata) error {
	extractedInfo, ok := root["info"]
	if !ok {
		return errors.New("info not found in root")
	}
	info, ok := extractedInfo.(map[string]any)
	if !ok {
		return errors.New("info is not a dictionary")
	}

	name, ok := info["name"].([]byte)
	if !ok {
		return errors.New("info.name is missing or not a string")
	}

	pieceLength, ok := info["piece length"].(int64)
	if !ok {
		return errors.New("info.piece_length is missing or not an integer")
	}

	pieces, ok := info["pieces"].([]byte)
	if !ok {
		return errors.New("info.pieces is missing or not a string")
	}

	_, hasLength := info["length"]
	_, hasFiles := info["files"]

	if hasLength == hasFiles {
		// both true OR both false
		return errors.New("info must contain exactly one of length or files")
	}

	// set all metadata
	metadata.Info.Name = string(name)
	metadata.Info.PieceLength = pieceLength
	metadata.Info.Pieces = pieces

	if hasLength {
		length, ok := info["length"].(int64)
		if !ok {
			return errors.New("info.length is not an integer")
		}
		metadata.Info.Length = length
	}
	if hasFiles {
		files, ok := info["files"].([]any)
		if !ok {
			return errors.New("info.files is not a list")
		}
		for _, file := range files {
			parsedFile, err := parseFile(file)
			if err != nil {
				return err
			}
			metadata.Info.Files = append(metadata.Info.Files, parsedFile)
		}
	}

	return nil
}

func parseFile(value any) (types.File, error) {
	fileDict, ok := value.(map[string]any)
	if !ok {
		return types.File{}, errors.New("file is not a dictionary")
	}

	length, ok := fileDict["length"].(int64)
	if !ok {
		return types.File{}, errors.New("file.length is missing or not an integer")
	}

	rawPath, ok := fileDict["path"].([]any)
	if !ok {
		return types.File{}, errors.New("file.path is missing or not a list")
	}

	path := make([]string, 0, len(rawPath))

	for _, component := range rawPath {
		componentBytes, ok := component.([]byte)
		if !ok {
			return types.File{}, errors.New("file.path contains a non-string component")
		}

		path = append(path, string(componentBytes))
	}

	return types.File{
		Length: length,
		Path:   path,
	}, nil
}
