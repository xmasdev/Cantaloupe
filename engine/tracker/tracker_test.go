package tracker

import (
	"bytes"
	"encoding/hex"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/xmasdev/Cantaloupe/engine/types"
)

func TestPercentEncodeBytes(t *testing.T) {
	input := []byte{
		0x00,
		0x01,
		0x20,
		0x7f,
		0x89,
		0xff,
		'a',
		'-',
		'_',
		'.',
		'~',
	}

	expected := "%00%01%20%7F%89%FFa-_.~"

	got := percentEncodeBytes(input)

	if got != expected {
		t.Fatalf(
			"unexpected encoding: expected %q, got %q",
			expected,
			got,
		)
	}
}

func TestBuildAnnounceURL(t *testing.T) {
	var infoHash [20]byte
	var peerID [20]byte

	for i := range infoHash {
		infoHash[i] = byte(i)
	}

	for i := range peerID {
		peerID[i] = byte(20 + i)
	}

	req := types.AnnounceRequest{
		InfoHash:   infoHash,
		PeerID:     peerID,
		Port:       6881,
		Uploaded:   100,
		Downloaded: 200,
		Left:       500,
		Event:      "started",
	}

	got, err := buildAnnounceURL(
		"http://tracker.example.com/announce",
		req,
	)
	if err != nil {
		t.Fatalf("buildAnnounceURL failed: %v", err)
	}

	t.Logf("announce URL: %s", got)

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("generated invalid URL: %v", err)
	}

	query := parsed.Query()

	if query.Get("port") != "6881" {
		t.Errorf("unexpected port: %q", query.Get("port"))
	}

	if query.Get("uploaded") != "100" {
		t.Errorf("unexpected uploaded: %q", query.Get("uploaded"))
	}

	if query.Get("downloaded") != "200" {
		t.Errorf("unexpected downloaded: %q", query.Get("downloaded"))
	}

	if query.Get("left") != "500" {
		t.Errorf("unexpected left: %q", query.Get("left"))
	}

	if query.Get("compact") != "1" {
		t.Errorf("unexpected compact: %q", query.Get("compact"))
	}

	if query.Get("event") != "started" {
		t.Errorf("unexpected event: %q", query.Get("event"))
	}

	// Query().Get() decodes percent encoding for us.
	decodedInfoHash, err := hex.DecodeString(
		"000102030405060708090a0b0c0d0e0f10111213",
	)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(
		[]byte(query.Get("info_hash")),
		decodedInfoHash,
	) {
		t.Errorf("info_hash was encoded incorrectly")
	}

	decodedPeerID, err := hex.DecodeString(
		"1415161718191a1b1c1d1e1f2021222324252627",
	)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(
		[]byte(query.Get("peer_id")),
		decodedPeerID,
	) {
		t.Errorf("peer_id was encoded incorrectly")
	}
}

func TestBuildAnnounceURLInvalidURL(t *testing.T) {
	req := types.AnnounceRequest{}

	_, err := buildAnnounceURL(
		"://definitely-not-a-url",
		req,
	)

	if err == nil {
		t.Fatal("expected invalid URL error, got nil")
	}
}

func TestParseCompactPeers(t *testing.T) {
	data := []byte{
		// 1.2.3.4:6881
		1, 2, 3, 4,
		0x1A, 0xE1,

		// 127.0.0.1:51413
		127, 0, 0, 1,
		0xC8, 0xD5,
	}

	peers, err := parseCompactPeers(data)
	if err != nil {
		t.Fatalf("parseCompactPeers failed: %v", err)
	}

	if len(peers) != 2 {
		t.Fatalf(
			"expected 2 peers, got %d",
			len(peers),
		)
	}

	if !peers[0].IP.Equal(net.IPv4(1, 2, 3, 4)) {
		t.Errorf(
			"unexpected first peer IP: %v",
			peers[0].IP,
		)
	}

	if peers[0].Port != 6881 {
		t.Errorf(
			"unexpected first peer port: %d",
			peers[0].Port,
		)
	}

	if !peers[1].IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Errorf(
			"unexpected second peer IP: %v",
			peers[1].IP,
		)
	}

	if peers[1].Port != 51413 {
		t.Errorf(
			"unexpected second peer port: %d",
			peers[1].Port,
		)
	}
}

func TestParseCompactPeersInvalidLength(t *testing.T) {
	data := []byte{
		1, 2, 3, 4, 5,
	}

	_, err := parseCompactPeers(data)

	if err == nil {
		t.Fatal("expected error for invalid compact peer list")
	}
}

func TestParseAnnounceResponse(t *testing.T) {
	peerData := []byte{
		1, 2, 3, 4,
		0x1A, 0xE1, // 6881

		5, 6, 7, 8,
		0xC8, 0xD5, // 51413
	}

	response := append(
		[]byte("d"),
		[]byte("8:completei42e10:incompletei17e8:intervali1800e5:peers12:")...,
	)

	response = append(response, peerData...)
	response = append(response, 'e')

	result, err := parseAnnounceResponse(response)
	if err != nil {
		t.Fatalf(
			"parseAnnounceResponse failed: %v",
			err,
		)
	}

	if result.Interval != 1800 {
		t.Errorf(
			"expected interval 1800, got %d",
			result.Interval,
		)
	}

	if result.Complete != 42 {
		t.Errorf(
			"expected complete 42, got %d",
			result.Complete,
		)
	}

	if result.Incomplete != 17 {
		t.Errorf(
			"expected incomplete 17, got %d",
			result.Incomplete,
		)
	}

	if len(result.Peers) != 2 {
		t.Fatalf(
			"expected 2 peers, got %d",
			len(result.Peers),
		)
	}

	t.Logf("Tracker response: %+v", result)
}

func TestParseAnnounceResponseFailure(t *testing.T) {
	data := []byte("d14:failure reason13:tracker failede")

	_, err := parseAnnounceResponse(data)

	if err == nil {
		t.Fatal("expected tracker failure error")
	}

	t.Logf("got expected error: %v", err)
}

func TestRequestTracker(t *testing.T) {
	expectedBody := []byte("test tracker response")

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf(
					"expected GET, got %s",
					r.Method,
				)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(expectedBody)
		}),
	)

	defer server.Close()

	body, err := requestTracker(server.URL)
	if err != nil {
		t.Fatalf(
			"requestTracker failed: %v",
			err,
		)
	}

	if !bytes.Equal(body, expectedBody) {
		t.Errorf(
			"unexpected response body: %q",
			body,
		)
	}
}

func TestRequestTrackerHTTPError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(
				w,
				"tracker unavailable",
				http.StatusServiceUnavailable,
			)
		}),
	)

	defer server.Close()

	_, err := requestTracker(server.URL)

	if err == nil {
		t.Fatal("expected HTTP error")
	}

	t.Logf("got expected error: %v", err)
}

func TestAnnounce(t *testing.T) {
	var infoHash [20]byte
	var peerID [20]byte

	for i := range infoHash {
		infoHash[i] = byte(i)
	}

	for i := range peerID {
		peerID[i] = byte(20 + i)
	}

	peerData := []byte{
		1, 2, 3, 4,
		0x1A, 0xE1, // 6881

		5, 6, 7, 8,
		0xC8, 0xD5, // 51413
	}

	trackerResponse := append(
		[]byte("d"),
		[]byte("8:completei42e10:incompletei17e8:intervali1800e5:peers12:")...,
	)

	trackerResponse = append(trackerResponse, peerData...)
	trackerResponse = append(trackerResponse, 'e')

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()

			if query.Get("port") != "6881" {
				t.Errorf(
					"unexpected port: %s",
					query.Get("port"),
				)
			}

			if query.Get("uploaded") != "0" {
				t.Errorf(
					"unexpected uploaded: %s",
					query.Get("uploaded"),
				)
			}

			if query.Get("downloaded") != "100" {
				t.Errorf(
					"unexpected downloaded: %s",
					query.Get("downloaded"),
				)
			}

			if query.Get("left") != "500" {
				t.Errorf(
					"unexpected left: %s",
					query.Get("left"),
				)
			}

			if query.Get("compact") != "1" {
				t.Errorf(
					"unexpected compact: %s",
					query.Get("compact"),
				)
			}

			if query.Get("event") != "started" {
				t.Errorf(
					"unexpected event: %s",
					query.Get("event"),
				)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(trackerResponse)
		}),
	)

	defer server.Close()

	req := types.AnnounceRequest{
		InfoHash:   infoHash,
		PeerID:     peerID,
		Port:       6881,
		Uploaded:   0,
		Downloaded: 100,
		Left:       500,
		Event:      "started",
	}

	response, err := Announce(server.URL, req)
	if err != nil {
		t.Fatalf(
			"Announce failed: %v",
			err,
		)
	}

	if response.Interval != 1800 {
		t.Errorf(
			"expected interval 1800, got %d",
			response.Interval,
		)
	}

	if response.Complete != 42 {
		t.Errorf(
			"expected complete 42, got %d",
			response.Complete,
		)
	}

	if response.Incomplete != 17 {
		t.Errorf(
			"expected incomplete 17, got %d",
			response.Incomplete,
		)
	}

	if len(response.Peers) != 2 {
		t.Fatalf(
			"expected 2 peers, got %d",
			len(response.Peers),
		)
	}

	t.Logf("Announce response: %+v", response)
}
