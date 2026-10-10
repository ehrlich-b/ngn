//go:build amd64 && amd64.v3 && !purego && !race

package engine

import "testing"

func TestNGNN4VNNIState(t *testing.T) {
	const hardware = uint32(1<<16 | 1<<31)
	const vector = uint32(1 << 11)
	if !ngnn4VNNIState(hardware, vector, 0xe6) {
		t.Fatal("complete hardware and OS state rejected")
	}
	for _, bit := range []uint32{1 << 16, 1 << 31} {
		if ngnn4VNNIState(hardware&^bit, vector, 0xe6) {
			t.Fatal("missing hardware feature admitted")
		}
	}
	if ngnn4VNNIState(hardware, 0, 0xe6) {
		t.Fatal("missing dot instruction admitted")
	}
	for _, bit := range []uint64{2, 4, 32, 64, 128} {
		if ngnn4VNNIState(hardware, vector, 0xe6&^bit) {
			t.Fatal("missing OS vector state admitted")
		}
	}
}

func TestNGNN4VNNIDirectAndFallback(t *testing.T) {
	if !ngnn4UseVNNI {
		t.Skip("VNNI unavailable")
	}
	t.Log("VNNI512 hardware and OS state admitted")
	// Each full all-width guarded differential sweep uses the int64 oracle.
	// Force fallback separately, even though this rig selects VNNI normally.
	TestNGNN4RowAffineDifferential(t)
	prior := ngnn4UseVNNI
	ngnn4UseVNNI = false
	defer func() { ngnn4UseVNNI = prior }()
	TestNGNN4RowAffineDifferential(t)
}
