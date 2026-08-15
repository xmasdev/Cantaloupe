package bencode

import (
	"bytes"
	"testing"
)

func TestDecodeInteger(t *testing.T) {
	value, err := Decode([]byte("i42e"))
	if err != nil {
		t.Fatal(err)
	}

	if value != int64(42) {
		t.Fatalf("expected 42, got %v", value)
	}
}

func TestDecodeString(t *testing.T) {
	value, err := Decode([]byte("4:spam"))
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte("spam")

	if !bytes.Equal(value.([]byte), expected) {
		t.Fatalf("expected %q, got %q", expected, value)
	}
}

func TestDecodeList(t *testing.T) {
	value, err := Decode([]byte("l4:spam4:eggse"))
	if err != nil {
		t.Fatal(err)
	}

	list := value.([]any)

	if len(list) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(list))
	}

	if string(list[0].([]byte)) != "spam" {
		t.Fatalf("unexpected first element")
	}

	if string(list[1].([]byte)) != "eggs" {
		t.Fatalf("unexpected second element")
	}
}

func TestDecodeDictionary(t *testing.T) {
	value, err := Decode([]byte("d3:foo3:bare"))
	if err != nil {
		t.Fatal(err)
	}

	dict := value.(map[string]any)

	if string(dict["foo"].([]byte)) != "bar" {
		t.Fatalf("unexpected dictionary value")
	}
}

func TestDecodeNested(t *testing.T) {
	data := []byte(
		"d4:spaml1:a1:bee",
	)

	value, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}

	dict := value.(map[string]any)

	list := dict["spam"].([]any)

	if len(list) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(list))
	}

	if string(list[0].([]byte)) != "a" {
		t.Fatalf("unexpected first element")
	}

	if string(list[1].([]byte)) != "b" {
		t.Fatalf("unexpected second element")
	}
}

func TestDecodeTorrentLikeStructure(t *testing.T) {
	data := []byte(
		"d4:infod4:name10:Cantaloupe12:piece lengthi262144eee",
	)

	value, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}

	root := value.(map[string]any)
	info := root["info"].(map[string]any)

	if string(info["name"].([]byte)) != "Cantaloupe" {
		t.Fatalf("unexpected name")
	}

	if info["piece length"] != int64(262144) {
		t.Fatalf("unexpected piece length")
	}
}