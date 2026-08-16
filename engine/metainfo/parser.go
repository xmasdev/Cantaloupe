package metainfo

import (
	"errors"
	"fmt"

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

	if err := extractRequiredFields(root, &metadata); err != nil {
		return nil, err
	}

	// Optional root-level fields.
	if announce, exists, err := extractField[[]byte](root, "announce"); err != nil {
		return nil, err
	} else if exists {
		metadata.Announce = string(announce)
	}

	if announceList, exists, err := extractField[[]any](root, "announce-list"); err != nil {
		return nil, err
	} else if exists {
		parsed, err := parseAnnounceList(announceList)
		if err != nil {
			return nil, err
		}
		metadata.AnnounceList = parsed
	}

	if creationDate, exists, err := extractField[int64](root, "creation date"); err != nil {
		return nil, err
	} else if exists {
		metadata.CreationDate = creationDate
	}

	if comment, exists, err := extractField[[]byte](root, "comment"); err != nil {
		return nil, err
	} else if exists {
		metadata.Comment = string(comment)
	}

	if encoding, exists, err := extractField[[]byte](root, "encoding"); err != nil {
		return nil, err
	} else if exists {
		metadata.Encoding = string(encoding)
	}

	return &metadata, nil
}

func extractRequiredFields(
	root map[string]any,
	metadata *types.TorrentMetadata,
) error {
	info, exists, err := extractField[map[string]any](root, "info")
	if err != nil {
		return err
	}

	if !exists {
		return errors.New("info not found in root")
	}

	name, exists, err := extractField[[]byte](info, "name")
	if err != nil {
		return err
	}

	if !exists {
		return errors.New("info.name not found")
	}

	pieceLength, exists, err := extractField[int64](info, "piece length")
	if err != nil {
		return err
	}

	if !exists {
		return errors.New("info.piece length not found")
	}

	if pieceLength <= 0 {
		return errors.New("info.piece length must be positive")
	}

	pieces, exists, err := extractField[[]byte](info, "pieces")
	if err != nil {
		return err
	}

	if !exists {
		return errors.New("info.pieces not found")
	}

	if len(pieces)%20 != 0 {
		return errors.New("info.pieces length must be a multiple of 20")
	}

	_, hasLength, err := extractField[int64](info, "length")
	if err != nil {
		return err
	}

	_, hasFiles, err := extractField[[]any](info, "files")
	if err != nil {
		return err
	}

	if hasLength == hasFiles {
		return errors.New(
			"info must contain exactly one of length or files",
		)
	}

	metadata.Info.Name = string(name)
	metadata.Info.PieceLength = pieceLength
	metadata.Info.Pieces = pieces

	if hasLength {
		length, _, err := extractField[int64](info, "length")
		if err != nil {
			return err
		}

		if length < 0 {
			return errors.New("info.length cannot be negative")
		}

		metadata.Info.Length = length
	}

	if hasFiles {
		files, _, err := extractField[[]any](info, "files")
		if err != nil {
			return err
		}

		metadata.Info.Files = make([]types.File, 0, len(files))

		for _, file := range files {
			parsedFile, err := parseFile(file)
			if err != nil {
				return err
			}

			metadata.Info.Files = append(
				metadata.Info.Files,
				parsedFile,
			)
		}
	}

	return nil
}

func extractField[T any](
	dict map[string]any,
	field string,
) (T, bool, error) {
	var zero T

	raw, exists := dict[field]
	if !exists {
		return zero, false, nil
	}

	value, ok := raw.(T)
	if !ok {
		return zero, true, fmt.Errorf(
			"%q field has an invalid type",
			field,
		)
	}

	return value, true, nil
}

func parseFile(value any) (types.File, error) {
	fileDict, ok := value.(map[string]any)
	if !ok {
		return types.File{}, errors.New(
			"file is not a dictionary",
		)
	}

	length, exists, err := extractField[int64](fileDict, "length")
	if err != nil {
		return types.File{}, err
	}

	if !exists {
		return types.File{}, errors.New(
			"file.length not found",
		)
	}

	if length < 0 {
		return types.File{}, errors.New(
			"file.length cannot be negative",
		)
	}

	rawPath, exists, err := extractField[[]any](fileDict, "path")
	if err != nil {
		return types.File{}, err
	}

	if !exists {
		return types.File{}, errors.New(
			"file.path not found",
		)
	}

	if len(rawPath) == 0 {
		return types.File{}, errors.New(
			"file.path cannot be empty",
		)
	}

	path := make([]string, 0, len(rawPath))

	for _, component := range rawPath {
		componentBytes, ok := component.([]byte)
		if !ok {
			return types.File{}, errors.New(
				"file.path contains a non-string component",
			)
		}

		path = append(path, string(componentBytes))
	}

	return types.File{
		Length: length,
		Path:   path,
	}, nil
}

func parseAnnounceList(raw []any) ([][]string, error) {
	result := make([][]string, 0, len(raw))

	for _, rawTier := range raw {
		tierValues, ok := rawTier.([]any)
		if !ok {
			return nil, errors.New(
				"announce-list tier is not a list",
			)
		}

		tier := make([]string, 0, len(tierValues))

		for _, rawTracker := range tierValues {
			tracker, ok := rawTracker.([]byte)
			if !ok {
				return nil, errors.New(
					"announce-list contains a non-string tracker",
				)
			}

			tier = append(tier, string(tracker))
		}

		result = append(result, tier)
	}

	return result, nil
}
