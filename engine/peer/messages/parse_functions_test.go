package messages

import (
	"bytes"
	"testing"
)

func TestParseHave(t *testing.T) {
	message := HaveMessage(42)

	got, err := ParseHave(message)
	if err != nil {
		t.Fatalf("ParseHave failed: %v", err)
	}

	if got.PieceIndex != 42 {
		t.Errorf(
			"expected piece index 42, got %d",
			got.PieceIndex,
		)
	}
}

func TestParseHaveWrongMessageID(t *testing.T) {
	message := InterestedMessage()

	_, err := ParseHave(message)
	if err == nil {
		t.Fatal("expected error for wrong message ID")
	}
}

func TestParseHaveInvalidPayload(t *testing.T) {
	message := Message{
		ID:      Have,
		Payload: []byte{1, 2, 3},
	}

	_, err := ParseHave(message)
	if err == nil {
		t.Fatal("expected error for invalid payload length")
	}
}

func TestParseBitfield(t *testing.T) {
	expected := []byte{
		0b10110000,
		0b01010000,
	}

	message := BitfieldMessage(expected)

	got, err := ParseBitfield(message)
	if err != nil {
		t.Fatalf("ParseBitfield failed: %v", err)
	}

	if !bytes.Equal(got.Bitfield, expected) {
		t.Errorf(
			"unexpected bitfield: expected %08b, got %08b",
			expected,
			got.Bitfield,
		)
	}
}

func TestParseBitfieldWrongMessageID(t *testing.T) {
	message := InterestedMessage()

	_, err := ParseBitfield(message)
	if err == nil {
		t.Fatal("expected error for wrong message ID")
	}
}

func TestParseBitfieldEmptyPayload(t *testing.T) {
	message := Message{
		ID:      Bitfield,
		Payload: []byte{},
	}

	_, err := ParseBitfield(message)
	if err == nil {
		t.Fatal("expected error for empty bitfield")
	}
}

func TestParseRequest(t *testing.T) {
	message := RequestMessage(
		5,
		16384,
		16384,
	)

	got, err := ParseRequest(message)
	if err != nil {
		t.Fatalf("ParseRequest failed: %v", err)
	}

	if got.PieceIndex != 5 {
		t.Errorf(
			"expected piece index 5, got %d",
			got.PieceIndex,
		)
	}

	if got.Begin != 16384 {
		t.Errorf(
			"expected begin 16384, got %d",
			got.Begin,
		)
	}

	if got.Length != 16384 {
		t.Errorf(
			"expected length 16384, got %d",
			got.Length,
		)
	}
}

func TestParseRequestWrongMessageID(t *testing.T) {
	message := InterestedMessage()

	_, err := ParseRequest(message)
	if err == nil {
		t.Fatal("expected error for wrong message ID")
	}
}

func TestParseRequestInvalidPayload(t *testing.T) {
	message := Message{
		ID:      Request,
		Payload: make([]byte, 11),
	}

	_, err := ParseRequest(message)
	if err == nil {
		t.Fatal("expected error for invalid payload length")
	}
}

func TestParsePiece(t *testing.T) {
	message := PieceMessage(
		5,
		16384,
		[]byte("hello torrent"),
	)

	got, err := ParsePiece(message)
	if err != nil {
		t.Fatalf("ParsePiece failed: %v", err)
	}

	if got.PieceIndex != 5 {
		t.Errorf(
			"expected piece index 5, got %d",
			got.PieceIndex,
		)
	}

	if got.Begin != 16384 {
		t.Errorf(
			"expected begin 16384, got %d",
			got.Begin,
		)
	}

	expectedBlock := []byte("hello torrent")

	if !bytes.Equal(got.Block, expectedBlock) {
		t.Errorf(
			"unexpected block: expected %q, got %q",
			expectedBlock,
			got.Block,
		)
	}
}

func TestParsePieceWrongMessageID(t *testing.T) {
	message := InterestedMessage()

	_, err := ParsePiece(message)
	if err == nil {
		t.Fatal("expected error for wrong message ID")
	}
}

func TestParsePieceTooShort(t *testing.T) {
	message := Message{
		ID:      Piece,
		Payload: make([]byte, 7),
	}

	_, err := ParsePiece(message)
	if err == nil {
		t.Fatal("expected error for short piece payload")
	}
}

func TestParseCancel(t *testing.T) {
	message := CancelMessage(
		10,
		32768,
		16384,
	)

	got, err := ParseCancel(message)
	if err != nil {
		t.Fatalf("ParseCancel failed: %v", err)
	}

	if got.PieceIndex != 10 {
		t.Errorf(
			"expected piece index 10, got %d",
			got.PieceIndex,
		)
	}

	if got.Begin != 32768 {
		t.Errorf(
			"expected begin 32768, got %d",
			got.Begin,
		)
	}

	if got.Length != 16384 {
		t.Errorf(
			"expected length 16384, got %d",
			got.Length,
		)
	}
}

func TestParseCancelWrongMessageID(t *testing.T) {
	message := InterestedMessage()

	_, err := ParseCancel(message)
	if err == nil {
		t.Fatal("expected error for wrong message ID")
	}
}

func TestParseCancelInvalidPayload(t *testing.T) {
	message := Message{
		ID:      Cancel,
		Payload: make([]byte, 10),
	}

	_, err := ParseCancel(message)
	if err == nil {
		t.Fatal("expected error for invalid payload length")
	}
}

func TestParsePort(t *testing.T) {
	message := PortMessage(6881)

	got, err := ParsePort(message)
	if err != nil {
		t.Fatalf("ParsePort failed: %v", err)
	}

	if got.Port != 6881 {
		t.Errorf(
			"expected port 6881, got %d",
			got.Port,
		)
	}
}

func TestParsePortWrongMessageID(t *testing.T) {
	message := InterestedMessage()

	_, err := ParsePort(message)
	if err == nil {
		t.Fatal("expected error for wrong message ID")
	}
}

func TestParsePortInvalidPayload(t *testing.T) {
	message := Message{
		ID:      Port,
		Payload: []byte{1},
	}

	_, err := ParsePort(message)
	if err == nil {
		t.Fatal("expected error for invalid payload length")
	}
}
