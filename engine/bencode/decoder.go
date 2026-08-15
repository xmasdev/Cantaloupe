package bencode

import (
	"errors"
	"fmt"
	"strconv"
)

func Decode(data []byte) (any, error) {
	pos := 0

	value, err := decode(data, &pos)
	if err != nil {
		return nil, err
	}

	// A valid bencoded document must contain exactly one value.
	if pos != len(data) {
		return nil, fmt.Errorf("trailing data at position %d", pos)
	}

	return value, nil
}

// decode parses exactly one bencode value starting at *pos.
// On success, *pos points to the first byte after the value.
func decode(data []byte, pos *int) (any, error) {
	if *pos >= len(data) {
		return nil, errors.New("unexpected end of input")
	}

	switch data[*pos] {
	case 'i':
		return decodeInteger(data, pos)

	case 'l':
		return decodeList(data, pos)

	case 'd':
		return decodeDictionary(data, pos)

	default:
		if isDigit(data[*pos]) {
			return decodeString(data, pos)
		}

		return nil, fmt.Errorf(
			"invalid bencode prefix %q at position %d",
			data[*pos],
			*pos,
		)
	}
}

func decodeInteger(data []byte, pos *int) (int64, error) {
	// Skip 'i'.
	*pos++

	start := *pos

	for *pos < len(data) && data[*pos] != 'e' {
		*pos++
	}

	if *pos >= len(data) {
		return 0, errors.New("unterminated integer")
	}

	if start == *pos {
		return 0, errors.New("empty integer")
	}

	numberBytes := data[start:*pos]

	// Move past 'e'.
	*pos++

	// Bencode has stricter integer rules than strconv.ParseInt:
	// - no leading zeroes except "0"
	// - "-0" is invalid
	if !validInteger(numberBytes) {
		return 0, fmt.Errorf("invalid integer %q", numberBytes)
	}

	value, err := strconv.ParseInt(string(numberBytes), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q: %w", numberBytes, err)
	}

	return value, nil
}

func decodeString(data []byte, pos *int) ([]byte, error) {
	start := *pos

	// Find ':'.
	for *pos < len(data) && data[*pos] != ':' {
		if !isDigit(data[*pos]) {
			return nil, fmt.Errorf(
				"invalid string length at position %d",
				*pos,
			)
		}

		*pos++
	}

	if *pos >= len(data) {
		return nil, errors.New("unterminated string length")
	}

	lengthBytes := data[start:*pos]

	length, err := strconv.ParseUint(string(lengthBytes), 10, 64)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid string length %q: %w",
			lengthBytes,
			err,
		)
	}

	// Skip ':'.
	*pos++

	// Make sure the claimed string fits inside the input.
	if length > uint64(len(data)-*pos) {
		return nil, errors.New("string extends beyond input")
	}

	end := *pos + int(length)

	value := data[*pos:end]

	*pos = end

	return value, nil
}

func decodeList(data []byte, pos *int) ([]any, error) {
	// Skip 'l'.
	*pos++

	var list []any

	for {
		if *pos >= len(data) {
			return nil, errors.New("unterminated list")
		}

		// End of list.
		if data[*pos] == 'e' {
			*pos++
			return list, nil
		}

		value, err := decode(data, pos)
		if err != nil {
			return nil, err
		}

		list = append(list, value)
	}
}

func decodeDictionary(data []byte, pos *int) (map[string]any, error) {
	// Skip 'd'.
	*pos++

	dictionary := make(map[string]any)

	for {
		if *pos >= len(data) {
			return nil, errors.New("unterminated dictionary")
		}

		// End of dictionary.
		if data[*pos] == 'e' {
			*pos++
			return dictionary, nil
		}

		// Dictionary keys must be byte strings.
		keyValue, err := decodeString(data, pos)
		if err != nil {
			return nil, fmt.Errorf("invalid dictionary key: %w", err)
		}

		key := string(keyValue)

		value, err := decode(data, pos)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid value for key %q: %w",
				key,
				err,
			)
		}

		dictionary[key] = value
	}
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func validInteger(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// "-0" is invalid.
	if len(data) == 2 && data[0] == '-' && data[1] == '0' {
		return false
	}

	// Leading zeroes are not allowed.
	if len(data) > 1 && data[0] == '0' {
		return false
	}

	// Negative numbers must have at least one digit.
	if data[0] == '-' {
		if len(data) == 1 {
			return false
		}

		if len(data) > 2 && data[1] == '0' {
			return false
		}
	}

	return true
}
