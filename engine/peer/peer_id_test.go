package peer

import "testing"

func TestGeneratePeerID(t *testing.T) {
	peerID, err := GeneratePeerID()
	if err != nil {
		t.Fatalf("failed to generate peer ID: %v", err)
	}

	expectedPrefix := "-CN0001-"

	if string(peerID[:8]) != expectedPrefix {
		t.Errorf(
			"unexpected peer ID prefix: expected %q, got %q",
			expectedPrefix,
			string(peerID[:8]),
		)
	}

	if len(peerID) != 20 {
		t.Errorf(
			"expected peer ID to be 20 bytes, got %d",
			len(peerID),
		)
	}
}

func TestGeneratePeerIDIsRandom(t *testing.T) {
	first, err := GeneratePeerID()
	if err != nil {
		t.Fatalf("failed to generate first peer ID: %v", err)
	}

	second, err := GeneratePeerID()
	if err != nil {
		t.Fatalf("failed to generate second peer ID: %v", err)
	}

	if first == second {
		t.Error("two generated peer IDs are identical")
	}
}
