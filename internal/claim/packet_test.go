package claim

import "testing"

func TestOpeningPositionKeysTranspositions(t *testing.T) {
	keys, err := OpeningPositionKeys([]string{"g1f3 g8f6 b1c3 b8c6", "b1c3 b8c6 g1f3 g8f6", "e2e4 e7e5"})
	if err != nil {
		t.Fatal(err)
	}
	if keys[0] != keys[1] || keys[0] == keys[2] {
		t.Fatal("position equivalence failed", keys)
	}
	for _, bad := range []string{"", "e2e5", "g1f3 g8f6 f3g1 f6g8 e1g1"} {
		if _, err := OpeningPositionKeys([]string{bad}); err == nil {
			t.Fatalf("accepted illegal opening %q", bad)
		}
	}
}
