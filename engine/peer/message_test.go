package peer

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"
)

func TestWriteReadMessage(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	writer := &Connection{
		conn: client,
	}

	reader := &Connection{
		conn: server,
	}

	expected := Message{
		ID:      Interested,
		Payload: []byte{},
	}

	errCh := make(chan error, 1)

	go func() {
		errCh <- writer.WriteMessage(expected)
	}()

	actual, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	if actual.KeepAlive {
		t.Fatal("expected normal message, got keep-alive")
	}

	if actual.ID != expected.ID {
		t.Errorf(
			"unexpected message ID: expected %d, got %d",
			expected.ID,
			actual.ID,
		)
	}

	if !bytes.Equal(actual.Payload, expected.Payload) {
		t.Errorf(
			"unexpected payload: expected %v, got %v",
			expected.Payload,
			actual.Payload,
		)
	}
}

func TestWriteReadMessageWithPayload(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	writer := &Connection{
		conn: client,
	}

	reader := &Connection{
		conn: server,
	}

	expected := Message{
		ID: Piece,
		Payload: []byte{
			0x01, 0x02, 0x03, 0x04,
			0x05, 0x06, 0x07, 0x08,
		},
	}

	errCh := make(chan error, 1)

	go func() {
		errCh <- writer.WriteMessage(expected)
	}()

	actual, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	if actual.KeepAlive {
		t.Fatal("expected normal message, got keep-alive")
	}

	if actual.ID != expected.ID {
		t.Errorf(
			"unexpected message ID: expected %d, got %d",
			expected.ID,
			actual.ID,
		)
	}

	if !bytes.Equal(actual.Payload, expected.Payload) {
		t.Errorf(
			"unexpected payload: expected %v, got %v",
			expected.Payload,
			actual.Payload,
		)
	}
}

func TestWriteReadKeepAlive(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	writer := &Connection{
		conn: client,
	}

	reader := &Connection{
		conn: server,
	}

	expected := Message{
		KeepAlive: true,
	}

	errCh := make(chan error, 1)

	go func() {
		errCh <- writer.WriteMessage(expected)
	}()

	actual, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	if !actual.KeepAlive {
		t.Fatal("expected keep-alive message")
	}

	if actual.ID != 0 {
		t.Errorf(
			"expected keep-alive ID to be 0, got %d",
			actual.ID,
		)
	}

	if len(actual.Payload) != 0 {
		t.Errorf(
			"expected empty keep-alive payload, got %v",
			actual.Payload,
		)
	}
}
func TestWriteReadMultipleMessages(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	writer := &Connection{
		conn: client,
	}

	reader := &Connection{
		conn: server,
	}

	messages := []Message{
		{ID: Interested},
		{ID: Have, Payload: []byte{0, 0, 0, 5}},
		{KeepAlive: true},
		{ID: Request, Payload: []byte{1, 2, 3}},
	}

	errCh := make(chan error, 1)

	go func() {
		for _, message := range messages {
			if err := writer.WriteMessage(message); err != nil {
				errCh <- err
				return
			}
		}

		errCh <- nil
	}()

	for i, expected := range messages {
		actual, err := reader.ReadMessage()
		if err != nil {
			t.Fatalf(
				"ReadMessage failed for message %d: %v",
				i,
				err,
			)
		}

		if actual.KeepAlive != expected.KeepAlive {
			t.Errorf(
				"message %d: keep-alive mismatch",
				i,
			)
		}

		if actual.ID != expected.ID {
			t.Errorf(
				"message %d: expected ID %d, got %d",
				i,
				expected.ID,
				actual.ID,
			)
		}

		if !bytes.Equal(actual.Payload, expected.Payload) {
			t.Errorf(
				"message %d: payload mismatch",
				i,
			)
		}
	}

	if err := <-errCh; err != nil {
		t.Fatalf("writer failed: %v", err)
	}
}
func TestReadMessageTooLarge(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	reader := &Connection{
		conn: server,
	}

	errCh := make(chan error, 1)

	go func() {
		// Write a length larger than maxMessageLength.
		length := uint32(maxMessageLength + 1)

		buffer := []byte{
			byte(length >> 24),
			byte(length >> 16),
			byte(length >> 8),
			byte(length),
		}

		_, err := client.Write(buffer)
		errCh <- err
	}()

	_, err := reader.ReadMessage()
	if err == nil {
		t.Fatal("expected error for oversized message")
	}

	t.Logf("got expected error: %v", err)

	if errors.Is(err, io.EOF) {
		t.Errorf("unexpected EOF: %v", err)
	}
}

func TestReadMessageTruncated(t *testing.T) {
	client, server := net.Pipe()

	reader := &Connection{
		conn: server,
	}

	errCh := make(chan error, 1)

	go func() {
		// Length says 5 bytes follow, but only send 2.
		data := []byte{
			0, 0, 0, 5,
			Interested,
			1,
		}

		_, err := client.Write(data)
		client.Close()

		errCh <- err
	}()

	_, err := reader.ReadMessage()

	if err == nil {
		t.Fatal("expected error for truncated message")
	}

	t.Logf("got expected error: %v", err)
	server.Close()

	<-errCh
}
