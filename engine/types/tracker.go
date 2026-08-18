package types

type AnnounceRequest struct {
	InfoHash   [20]byte
	PeerID     [20]byte
	Port       uint16
	Uploaded   int64
	Downloaded int64
	Left       int64
	Event      string
}

type AnnounceResponse struct {
	Interval   int64
	Peers      []Peer
	Complete   int64
	Incomplete int64
}
