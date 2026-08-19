package messages

import "encoding/binary"

type Message struct {
	ID        byte
	Payload   []byte
	KeepAlive bool
}

const (
	Choke         byte = 0
	Unchoke       byte = 1
	Interested    byte = 2
	NotInterested byte = 3
	Have          byte = 4
	Bitfield      byte = 5
	Request       byte = 6
	Piece         byte = 7
	Cancel        byte = 8
	Port          byte = 9
)

func KeepAliveMessage() Message {
	return Message{
		KeepAlive: true,
	}
}

func ChokeMessage() Message {
	return Message{
		ID: Choke,
	}
}

func UnchokeMessage() Message {
	return Message{
		ID: Unchoke,
	}
}

func InterestedMessage() Message {
	return Message{
		ID: Interested,
	}
}

func NotInterestedMessage() Message {
	return Message{
		ID: NotInterested,
	}
}

func HaveMessage(pieceIndex uint32) Message {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, pieceIndex)

	return Message{
		ID:      Have,
		Payload: payload,
	}
}

func BitfieldMessage(bitfield []byte) Message {
	return Message{
		ID:      Bitfield,
		Payload: bitfield,
	}
}

func RequestMessage(
	pieceIndex uint32,
	begin uint32,
	length uint32,
) Message {
	payload := make([]byte, 12)

	binary.BigEndian.PutUint32(payload[0:4], pieceIndex)
	binary.BigEndian.PutUint32(payload[4:8], begin)
	binary.BigEndian.PutUint32(payload[8:12], length)

	return Message{
		ID:      Request,
		Payload: payload,
	}
}

func CancelMessage(
	pieceIndex uint32,
	begin uint32,
	length uint32,
) Message {
	payload := make([]byte, 12)

	binary.BigEndian.PutUint32(payload[0:4], pieceIndex)
	binary.BigEndian.PutUint32(payload[4:8], begin)
	binary.BigEndian.PutUint32(payload[8:12], length)

	return Message{
		ID:      Cancel,
		Payload: payload,
	}
}

func PortMessage(port uint16) Message {
	payload := make([]byte, 2)
	binary.BigEndian.PutUint16(payload, port)

	return Message{
		ID:      Port,
		Payload: payload,
	}
}

func PieceMessage(
	pieceIndex uint32,
	begin uint32,
	block []byte,
) Message {
	payload := make([]byte, 8+len(block))

	binary.BigEndian.PutUint32(payload[0:4], pieceIndex)
	binary.BigEndian.PutUint32(payload[4:8], begin)
	copy(payload[8:], block)

	return Message{
		ID:      Piece,
		Payload: payload,
	}
}
