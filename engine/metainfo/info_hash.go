package metainfo

import (
	"crypto/sha1"

	"github.com/xmasdev/Cantaloupe/engine/bencode"
)

func GetInfoHash(data []byte) ([20]byte, error) {
	infoBytes, err := bencode.GetInfoBytes(data)
	if err != nil {
		return [20]byte{}, err
	}
	hash := sha1.Sum(infoBytes)
	return hash, nil

}
