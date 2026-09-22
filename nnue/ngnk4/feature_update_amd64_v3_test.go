//go:build amd64 && amd64.v3

package ngnk4

import "testing"

func TestK4FeatureUpdatesUseAVX2OnAMD64V3(t *testing.T) {
	if !k4FeatureUpdateUsesAVX2 {
		t.Fatal("amd64.v3 did not select the K4 AVX2 update kernels")
	}
}
