package bencode

import "errors"

func GetInfoBytes(data []byte) ([]byte, error) {
	pos := 0

	// Root must be a dictionary.
	if data[pos] != 'd' {
		return nil, errors.New("root is not a dictionary")
	}

	pos++

	for data[pos] != 'e' {
		// Decode the dictionary key.
		key, err := decode(data, &pos)
		if err != nil {
			return nil, err
		}

		keyBytes, ok := key.([]byte)
		if !ok {
			return nil, errors.New("dictionary key is not a string")
		}

		if string(keyBytes) == "info" {
			// pos is currently exactly at
			// the beginning of the info value.
			start := pos

			_, err := decode(data, &pos)
			if err != nil {
				return nil, err
			}

			end := pos

			return data[start:end], nil
		}

		// Not info, so decode value just to advance pos.
		if _, err := decode(data, &pos); err != nil {
			return nil, err
		}
	}

	return nil, errors.New("info dictionary not found")
}
