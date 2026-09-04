package tracker

import (
	"encoding/binary"
	"fmt"
	"net"
	"testing"

	"github.com/xmasdev/Cantaloupe/engine/types"
)

func TestAnnounceUDP(t *testing.T) {
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	finished := make(chan error, 1)
	go func() {
		buffer := make([]byte, 2048)
		if n, address, err := server.ReadFromUDP(buffer); err != nil {
			finished <- err
			return
		} else {
			if n != 16 || binary.BigEndian.Uint64(buffer[:8]) != udpConnectMagic || binary.BigEndian.Uint32(buffer[8:12]) != udpActionConnect {
				finished <- fmt.Errorf("invalid connect packet")
				return
			}
			response := make([]byte, 16)
			binary.BigEndian.PutUint32(response[0:4], udpActionConnect)
			binary.BigEndian.PutUint32(response[4:8], binary.BigEndian.Uint32(buffer[12:16]))
			binary.BigEndian.PutUint64(response[8:16], 0x0102030405060708)
			if _, err := server.WriteToUDP(response, address); err != nil {
				finished <- err
				return
			}
		}

		n, address, err := server.ReadFromUDP(buffer)
		if err != nil {
			finished <- err
			return
		}
		if n != 98 || binary.BigEndian.Uint32(buffer[8:12]) != udpActionAnnounce {
			finished <- fmt.Errorf("invalid announce packet")
			return
		}
		response := make([]byte, 26)
		binary.BigEndian.PutUint32(response[0:4], udpActionAnnounce)
		binary.BigEndian.PutUint32(response[4:8], binary.BigEndian.Uint32(buffer[12:16]))
		binary.BigEndian.PutUint32(response[8:12], 1800)
		binary.BigEndian.PutUint32(response[12:16], 2)
		binary.BigEndian.PutUint32(response[16:20], 3)
		copy(response[20:], []byte{127, 0, 0, 1, 0x1A, 0xE1})
		_, err = server.WriteToUDP(response, address)
		finished <- err
	}()

	response, err := announceUDP("udp://"+server.LocalAddr().String(), types.AnnounceRequest{Port: 6881})
	if err != nil {
		t.Fatal(err)
	}
	if response.Interval != 1800 || response.Complete != 3 || response.Incomplete != 2 {
		t.Fatalf("unexpected response: %+v", response)
	}
	if len(response.Peers) != 1 || response.Peers[0].Port != 6881 {
		t.Fatalf("unexpected peers: %+v", response.Peers)
	}
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
}
