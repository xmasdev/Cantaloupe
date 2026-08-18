package types

import "net"

type Peer struct {
	IP   net.IP
	Port uint16
}
