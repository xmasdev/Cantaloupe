package engine

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/xmasdev/Cantaloupe/engine/types"
)

func createTestTorrent(t *testing.T) string {
	t.Helper()

	data := []byte("hello")
	hash := sha1.Sum(data)

	info := []byte(
		"d6:lengthi5e4:name8:test.txt12:piece lengthi16384e6:pieces20:",
	)
	info = append(info, hash[:]...)
	info = append(info, 'e')

	torrent := []byte("d4:info")
	torrent = append(torrent, info...)
	torrent = append(torrent, 'e')

	path := filepath.Join(t.TempDir(), "test.torrent")

	if err := os.WriteFile(path, torrent, 0644); err != nil {
		t.Fatal(err)
	}

	return path
}
func TestNewEngine(t *testing.T) {
	torrentPath := createTestTorrent(t)
	outputDir := t.TempDir()

	engine, err := NewEngine(torrentPath, outputDir)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}
	defer engine.Close()

	if engine.Metadata == nil {
		t.Fatal("Metadata is nil")
	}

	if engine.InfoHash == [20]byte{} {
		t.Fatal("InfoHash is zero")
	}

	if engine.PeerID == [20]byte{} {
		t.Fatal("PeerID is zero")
	}

	if engine.Port <= 0 {
		t.Fatalf("invalid port: %d", engine.Port)
	}

	if engine.Download == nil {
		t.Fatal("Download is nil")
	}

	if engine.Storage == nil {
		t.Fatal("Storage is nil")
	}

	if engine.Listener == nil {
		t.Fatal("Listener is nil")
	}
}

func TestNewEngineInvalidTorrentPath(t *testing.T) {
	_, err := NewEngine("/does/not/exist.torrent", t.TempDir())

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewEngineInvalidTorrent(t *testing.T) {
	dir := t.TempDir()
	torrentPath := filepath.Join(dir, "invalid.torrent")

	if err := os.WriteFile(torrentPath, []byte("not bencode"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := NewEngine(torrentPath, t.TempDir())

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewEngineCreatesListenerOnAvailablePort(t *testing.T) {
	torrentPath := createTestTorrent(t)

	engine, err := NewEngine(torrentPath, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	// The port should actually be occupied by our listener.
	conn, err := net.Dial("tcp", engine.Listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to connect to engine listener: %v", err)
	}
	conn.Close()
}

func TestEngineAnnounce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("compact") != "1" {
			t.Errorf("compact = %q, want 1", r.URL.Query().Get("compact"))
		}

		if r.URL.Query().Get("port") == "" {
			t.Error("missing port")
		}

		if r.URL.Query().Get("uploaded") != "0" {
			t.Errorf("uploaded = %q, want 0", r.URL.Query().Get("uploaded"))
		}

		if r.URL.Query().Get("downloaded") != "0" {
			t.Errorf("downloaded = %q, want 0", r.URL.Query().Get("downloaded"))
		}

		if r.URL.Query().Get("left") == "" {
			t.Error("missing left")
		}

		// One peer: 127.0.0.1:6881
		peer := []byte{
			127, 0, 0, 1,
			0x1A, 0xE1,
		}

		response := append(
			[]byte("d8:intervali1800e5:peers6:"),
			peer...,
		)
		response = append(response, []byte("e")...)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	}))
	defer server.Close()

	torrentPath := createTestTorrentWithAnnounce(t, server.URL)

	engine, err := NewEngine(torrentPath, t.TempDir())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}
	defer engine.Close()

	response, err := engine.Announce()
	if err != nil {
		t.Fatalf("Announce() error = %v", err)
	}

	if response == nil {
		t.Fatal("response is nil")
	}

	if response.Interval != 1800 {
		t.Fatalf(
			"Interval = %d, want 1800",
			response.Interval,
		)
	}

	if len(response.Peers) != 1 {
		t.Fatalf(
			"Peers = %d, want 1",
			len(response.Peers),
		)
	}

	if !response.Peers[0].IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Errorf(
			"peer IP = %v, want 127.0.0.1",
			response.Peers[0].IP,
		)
	}

	if response.Peers[0].Port != 6881 {
		t.Errorf(
			"peer port = %d, want 6881",
			response.Peers[0].Port,
		)
	}
}
func createTestTorrentWithAnnounce(t *testing.T, announce string) string {
	t.Helper()

	pieceHash := make([]byte, 20)

	info := append(
		[]byte("d6:lengthi5e4:name8:test.txt12:piece lengthi16384e6:pieces20:"),
		pieceHash...,
	)
	info = append(info, 'e')

	torrent := []byte("d8:announce")
	torrent = append(torrent, []byte(strconv.Itoa(len(announce)))...)
	torrent = append(torrent, ':')
	torrent = append(torrent, []byte(announce)...)
	torrent = append(torrent, []byte("4:info")...)

	// Dictionary key ordering matters for bencode.
	// announce comes before info, which is correct.
	torrent = append(torrent, info...)
	torrent = append(torrent, 'e')

	path := filepath.Join(t.TempDir(), "test.torrent")

	if err := os.WriteFile(path, torrent, 0644); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestEngineConnectToPeers(t *testing.T) {
	server := newFakePeerServer(t)
	defer server.Close()

	torrentPath := createTestTorrent(t)

	engine, err := NewEngine(torrentPath, t.TempDir())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}
	defer engine.Close()

	port := server.Addr().(*net.TCPAddr).Port

	peers := []*types.Peer{
		{
			IP:   net.ParseIP("127.0.0.1"),
			Port: uint16(port),
		},
	}

	err = engine.ConnectToPeers(peers)
	if err != nil {
		t.Fatalf("ConnectToPeers() error = %v", err)
	}

	if len(engine.Peers) != 1 {
		t.Fatalf(
			"connected peers = %d, want 1",
			len(engine.Peers),
		)
	}
}

func newFakePeerServer(t *testing.T) net.Listener {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// --------------------
		// Handshake
		// --------------------

		clientHandshake := make([]byte, 68)

		if _, err := io.ReadFull(conn, clientHandshake); err != nil {
			return
		}

		response := make([]byte, 68)

		response[0] = 19
		copy(response[1:20], []byte("BitTorrent protocol"))

		// Same torrent.
		copy(response[28:48], clientHandshake[28:48])

		// Exactly 20 bytes.
		peerID := []byte("-FAKE01-123456789012")
		if len(peerID) != 20 {
			return
		}
		copy(response[48:68], peerID)

		if _, err := conn.Write(response); err != nil {
			return
		}

		// --------------------
		// Bitfield
		// --------------------
		//
		// One piece, and we have it.
		// MSB of first byte = piece 0.
		//

		if err := writePeerMessage(conn, 5, []byte{0x80}); err != nil {
			return
		}

		// --------------------
		// Wait for Interested
		// --------------------

		messageID, payload, err := readPeerMessage(conn)
		if err != nil {
			return
		}

		if messageID != 2 || len(payload) != 0 {
			return
		}

		// --------------------
		// Send Unchoke
		// --------------------

		if err := writePeerMessage(conn, 1, nil); err != nil {
			return
		}

		// --------------------
		// Wait for Request
		// --------------------

		messageID, payload, err = readPeerMessage(conn)
		if err != nil {
			return
		}

		if messageID != 6 || len(payload) != 12 {
			return
		}

		pieceIndex := binary.BigEndian.Uint32(payload[0:4])
		begin := binary.BigEndian.Uint32(payload[4:8])
		length := binary.BigEndian.Uint32(payload[8:12])

		if pieceIndex != 0 || begin != 0 || length != 5 {
			return
		}

		// --------------------
		// Send Piece
		// --------------------

		piecePayload := make([]byte, 13)

		binary.BigEndian.PutUint32(piecePayload[0:4], 0)
		binary.BigEndian.PutUint32(piecePayload[4:8], 0)
		copy(piecePayload[8:], []byte("hello"))

		if err := writePeerMessage(conn, 7, piecePayload); err != nil {
			return
		}
	}()

	return listener
}
func writePeerMessage(
	conn net.Conn,
	id byte,
	payload []byte,
) error {
	length := uint32(1 + len(payload))

	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, length)

	if _, err := conn.Write(header); err != nil {
		return err
	}

	if _, err := conn.Write([]byte{id}); err != nil {
		return err
	}

	if len(payload) > 0 {
		if _, err := conn.Write(payload); err != nil {
			return err
		}
	}

	return nil
}

func readPeerMessage(conn net.Conn) (byte, []byte, error) {
	header := make([]byte, 4)

	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, nil, err
	}

	length := binary.BigEndian.Uint32(header)

	if length == 0 {
		return 0, nil, nil
	}

	data := make([]byte, length)

	if _, err := io.ReadFull(conn, data); err != nil {
		return 0, nil, err
	}

	return data[0], data[1:], nil
}

func TestEngineConnectToPeersSkipsFailedPeers(t *testing.T) {
	server := newFakePeerServer(t)
	defer server.Close()

	torrentPath := createTestTorrent(t)

	engine, err := NewEngine(torrentPath, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	port := server.Addr().(*net.TCPAddr).Port

	peers := []*types.Peer{
		{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 1, // almost certainly unavailable
		},
		{
			IP:   net.ParseIP("127.0.0.1"),
			Port: uint16(port),
		},
	}

	err = engine.ConnectToPeers(peers)
	if err != nil {
		t.Fatalf("ConnectToPeers() error = %v", err)
	}

	if len(engine.Peers) != 1 {
		t.Fatalf(
			"connected peers = %d, want 1",
			len(engine.Peers),
		)
	}
}

func TestEngineDownloadTorrent(t *testing.T) {
	server := newFakePeerServer(t)
	defer server.Close()

	torrentPath := createTestTorrent(t)
	outputDir := t.TempDir()

	engine, err := NewEngine(torrentPath, outputDir)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}
	defer engine.Close()

	port := server.Addr().(*net.TCPAddr).Port

	peers := []*types.Peer{
		{
			IP:   net.ParseIP("127.0.0.1"),
			Port: uint16(port),
		},
	}

	if err := engine.ConnectToPeers(peers); err != nil {
		t.Fatalf("ConnectToPeers() error = %v", err)
	}

	if err := engine.DownloadTorrent(); err != nil {
		t.Fatalf("DownloadTorrent() error = %v", err)
	}

	path := filepath.Join(outputDir, "test.txt")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}

	expected := []byte("hello")

	if !bytes.Equal(data, expected) {
		t.Fatalf(
			"downloaded data = %q, want %q",
			data,
			expected,
		)
	}

	if !engine.Download.Pieces[0].Complete() {
		t.Fatal("piece 0 should be complete")
	}
}
