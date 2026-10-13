package engine

import (
	"reflect"
	"testing"
)

func TestSMPHelperDepthSchedules(t *testing.T) {
	for worker, want := range [][]int{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, {1, 2, 3, 4, 6, 8, 10}, {1, 2, 3, 4, 5, 7, 9, 10}, {1, 2, 3, 4, 5, 6, 7, 8, 9, 10}} {
		var got []int
		for d := 1; d <= 10; d = nextSMPHelperDepth(d, 10, worker) {
			got = append(got, d)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("worker%d depths%v want%v", worker, got, want)
		}
	}
}
func TestSMPHelperDepthCapsAndWarmup(t *testing.T) {
	for cap := 1; cap <= MaximumDepth; cap++ {
		for worker := 0; worker < 9; worker++ {
			last := 0
			count := 0
			for d := 1; d <= cap; d = nextSMPHelperDepth(d, cap, worker) {
				if d <= last || d > cap {
					t.Fatal("nonprogress/cap")
				}
				if last < 4 && d != last+1 {
					t.Fatal("warmup skipped")
				}
				last = d
				count++
				if count > cap {
					t.Fatal("loop did not terminate")
				}
			}
			if last != cap {
				t.Fatal("requested cap missed")
			}
		}
	}
}
