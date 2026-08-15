package types

type TorrentMetadata struct {
	Announce     string
	AnnounceList []string
	CreationDate int64
	Comment      string
	Encoding     string
	Info         Info
}

type Info struct {
	Name        string
	PieceLength int64
	Pieces      []byte

	// Single-file torrent
	Length int64

	// Multi-file torrent
	Files []File
}

type File struct {
	Length int64
	Path   []string
}
