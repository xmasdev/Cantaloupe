package tracker

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/xmasdev/Cantaloupe/engine/types"
)

const (
	udpConnectMagic   uint64 = 0x41727101980
	udpActionConnect         = 0
	udpActionAnnounce        = 1
	udpActionError           = 3
)

func announceUDP(trackerURL string, req types.AnnounceRequest) (*types.AnnounceResponse, error) {
	parsed, err := url.Parse(trackerURL)
	if err != nil || parsed.Host == "" {
		return nil, fmt.Errorf("invalid UDP tracker URL %q", trackerURL)
	}

	address, err := net.ResolveUDPAddr("udp", parsed.Host)
	if err != nil {
		return nil, fmt.Errorf("resolve UDP tracker: %w", err)
	}
	conn, err := net.DialUDP("udp", nil, address)
	if err != nil {
		return nil, fmt.Errorf("connect to UDP tracker: %w", err)
	}
	defer conn.Close()

	transaction := randomUint32()
	connectPacket := make([]byte, 16)
	binary.BigEndian.PutUint64(connectPacket[0:8], udpConnectMagic)
	binary.BigEndian.PutUint32(connectPacket[8:12], udpActionConnect)
	binary.BigEndian.PutUint32(connectPacket[12:16], transaction)

	if response, err := udpRoundTrip(conn, connectPacket, transaction); err != nil {
		return nil, fmt.Errorf("UDP tracker connect: %w", err)
	} else if len(response) < 16 {
		return nil, errors.New("UDP tracker connect response is too short")
	} else {
		connectionID := binary.BigEndian.Uint64(response[8:16])
		transaction = randomUint32()
		announcePacket := make([]byte, 98)
		binary.BigEndian.PutUint64(announcePacket[0:8], connectionID)
		binary.BigEndian.PutUint32(announcePacket[8:12], udpActionAnnounce)
		binary.BigEndian.PutUint32(announcePacket[12:16], transaction)
		copy(announcePacket[16:36], req.InfoHash[:])
		copy(announcePacket[36:56], req.PeerID[:])
		binary.BigEndian.PutUint64(announcePacket[56:64], uint64(maxZero(req.Downloaded)))
		binary.BigEndian.PutUint64(announcePacket[64:72], uint64(maxZero(req.Left)))
		binary.BigEndian.PutUint64(announcePacket[72:80], uint64(maxZero(req.Uploaded)))
		binary.BigEndian.PutUint32(announcePacket[80:84], uint32(udpEvent(req.Event)))
		binary.BigEndian.PutUint32(announcePacket[84:88], 0) // IPv4 address: let tracker infer it.
		binary.BigEndian.PutUint32(announcePacket[88:92], randomUint32())
		numWant := int32(-1)
		binary.BigEndian.PutUint32(announcePacket[92:96], uint32(numWant))
		binary.BigEndian.PutUint16(announcePacket[96:98], req.Port)

		response, err = udpRoundTrip(conn, announcePacket, transaction)
		if err != nil {
			return nil, fmt.Errorf("UDP tracker announce: %w", err)
		}
		if len(response) < 20 {
			return nil, errors.New("UDP tracker announce response is too short")
		}
		result := &types.AnnounceResponse{Interval: int64(binary.BigEndian.Uint32(response[8:12])), Incomplete: int64(binary.BigEndian.Uint32(response[12:16])), Complete: int64(binary.BigEndian.Uint32(response[16:20]))}
		peers, err := parseCompactPeers(response[20:])
		if err != nil {
			return nil, fmt.Errorf("parse UDP tracker peers: %w", err)
		}
		result.Peers = peers
		return result, nil
	}
}

func udpRoundTrip(conn *net.UDPConn, packet []byte, transaction uint32) ([]byte, error) {
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return nil, err
	}
	if _, err := conn.Write(packet); err != nil {
		return nil, err
	}
	buffer := make([]byte, 64*1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return nil, err
		}
		if n < 8 || binary.BigEndian.Uint32(buffer[4:8]) != transaction {
			continue
		}
		if binary.BigEndian.Uint32(buffer[0:4]) == udpActionError {
			return nil, fmt.Errorf("tracker error: %s", string(buffer[8:n]))
		}
		if binary.BigEndian.Uint32(buffer[0:4]) != udpActionConnect && binary.BigEndian.Uint32(buffer[0:4]) != udpActionAnnounce {
			continue
		}
		return append([]byte(nil), buffer[:n]...), nil
	}
}

func randomUint32() uint32 {
	var value uint32
	if err := binary.Read(rand.Reader, binary.BigEndian, &value); err != nil {
		return uint32(time.Now().UnixNano())
	}
	return value
}
func maxZero(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
func udpEvent(event string) int {
	switch event {
	case "completed":
		return 1
	case "started":
		return 2
	case "stopped":
		return 3
	default:
		return 0
	}
}
