package types

import "net"

type Peer struct {
	IP   net.IP
	Port uint16
}

type Bitfield []byte

func (b *Bitfield) SetPiece(index int) {
	if index < 0 {
		return
	}

	byteIndex := index / 8

	if byteIndex >= len(*b) {
		*b = append(*b, make([]byte, byteIndex-len(*b)+1)...)
	}

	bitIndex := 7 - (index % 8)

	(*b)[byteIndex] |= 1 << bitIndex
}

func (b Bitfield) HasPiece(index int) bool {
	if index < 0 {
		return false
	}

	byteIndex := index / 8

	if byteIndex >= len(b) {
		return false
	}

	bitIndex := 7 - (index % 8)

	return (b[byteIndex] & (1 << bitIndex)) != 0
}
