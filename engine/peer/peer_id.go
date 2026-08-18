package peer

import "crypto/rand"

func GeneratePeerID() ([20]byte, error) {
	var peerID [20]byte

	copy(peerID[:8], "-CN0001-")

	_, err := rand.Read(peerID[8:])

	return peerID, err
}
