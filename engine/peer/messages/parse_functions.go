package messages

import (
	"encoding/binary"
	"errors"
)

type HaveData struct {
	PieceIndex uint32
}

type BitfieldData struct {
	Bitfield []byte
}

type RequestData struct {
	PieceIndex uint32
	Begin      uint32
	Length     uint32
}

type PieceData struct {
	PieceIndex uint32
	Begin      uint32
	Block      []byte
}

type CancelData struct {
	PieceIndex uint32
	Begin      uint32
	Length     uint32
}

type PortData struct {
	Port uint16
}

func ParseHave(message Message) (HaveData, error) {
	if message.ID != Have {
		return HaveData{}, errors.New("message is not a Have message")
	}

	if len(message.Payload) != 4 {
		return HaveData{}, errors.New("invalid Have payload length")
	}

	return HaveData{
		PieceIndex: binary.BigEndian.Uint32(message.Payload),
	}, nil
}

func ParseBitfield(message Message) (BitfieldData, error) {
	if message.ID != Bitfield {
		return BitfieldData{}, errors.New("message is not a Bitfield message")
	}

	if len(message.Payload) == 0 {
		return BitfieldData{}, errors.New("bitfield payload cannot be empty")
	}

	return BitfieldData{
		Bitfield: message.Payload,
	}, nil
}

func ParseRequest(message Message) (RequestData, error) {
	if message.ID != Request {
		return RequestData{}, errors.New("message is not a Request message")
	}

	if len(message.Payload) != 12 {
		return RequestData{}, errors.New("invalid Request payload length")
	}

	return RequestData{
		PieceIndex: binary.BigEndian.Uint32(message.Payload[0:4]),
		Begin:      binary.BigEndian.Uint32(message.Payload[4:8]),
		Length:     binary.BigEndian.Uint32(message.Payload[8:12]),
	}, nil
}

func ParsePiece(message Message) (PieceData, error) {
	if message.ID != Piece {
		return PieceData{}, errors.New("message is not a Piece message")
	}

	if len(message.Payload) < 8 {
		return PieceData{}, errors.New("Piece payload is too short")
	}

	return PieceData{
		PieceIndex: binary.BigEndian.Uint32(message.Payload[0:4]),
		Begin:      binary.BigEndian.Uint32(message.Payload[4:8]),
		Block:      message.Payload[8:],
	}, nil
}

func ParseCancel(message Message) (CancelData, error) {
	if message.ID != Cancel {
		return CancelData{}, errors.New("message is not a Cancel message")
	}

	if len(message.Payload) != 12 {
		return CancelData{}, errors.New("invalid Cancel payload length")
	}

	return CancelData{
		PieceIndex: binary.BigEndian.Uint32(message.Payload[0:4]),
		Begin:      binary.BigEndian.Uint32(message.Payload[4:8]),
		Length:     binary.BigEndian.Uint32(message.Payload[8:12]),
	}, nil
}

func ParsePort(message Message) (PortData, error) {
	if message.ID != Port {
		return PortData{}, errors.New("message is not a Port message")
	}

	if len(message.Payload) != 2 {
		return PortData{}, errors.New("invalid Port payload length")
	}

	return PortData{
		Port: binary.BigEndian.Uint16(message.Payload),
	}, nil
}
