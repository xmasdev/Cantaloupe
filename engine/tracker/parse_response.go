package tracker

import (
	"errors"
	"fmt"
	"net"

	"github.com/xmasdev/Cantaloupe/engine/bencode"
	"github.com/xmasdev/Cantaloupe/engine/types"
)

func parseAnnounceResponse(data []byte) (*types.AnnounceResponse, error) {
	decoded, err := bencode.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode tracker response: %w", err)
	}

	root, ok := decoded.(map[string]any)
	if !ok {
		return nil, errors.New("tracker response is not a dictionary")
	}

	// Tracker can return a failure response instead of a normal response.
	if failure, exists := root["failure reason"]; exists {
		reason, ok := failure.([]byte)
		if !ok {
			return nil, errors.New(
				"tracker failure reason is not a string",
			)
		}

		return nil, fmt.Errorf(
			"tracker failure: %s",
			string(reason),
		)
	}

	interval, ok := root["interval"].(int64)
	if !ok {
		return nil, errors.New(
			"tracker response missing or invalid interval",
		)
	}

	rawPeers, ok := root["peers"].([]byte)
	if !ok {
		return nil, errors.New(
			"tracker response missing or invalid peers",
		)
	}

	peers, err := parseCompactPeers(rawPeers)
	if err != nil {
		return nil, err
	}

	response := &types.AnnounceResponse{
		Interval: interval,
		Peers:    peers,
	}

	if complete, ok := root["complete"].(int64); ok {
		response.Complete = complete
	}

	if incomplete, ok := root["incomplete"].(int64); ok {
		response.Incomplete = incomplete
	}

	return response, nil
}

func parseCompactPeers(data []byte) ([]types.Peer, error) {
	if len(data)%6 != 0 {
		return nil, fmt.Errorf(
			"invalid compact peer list length: %d",
			len(data),
		)
	}

	peers := make([]types.Peer, 0, len(data)/6)

	for i := 0; i < len(data); i += 6 {
		ip := net.IPv4(
			data[i],
			data[i+1],
			data[i+2],
			data[i+3],
		)

		port := uint16(data[i+4])<<8 | uint16(data[i+5])

		peers = append(peers, types.Peer{
			IP:   ip,
			Port: port,
		})
	}

	return peers, nil
}
