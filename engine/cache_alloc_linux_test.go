//go:build linux

package engine

import (
	"runtime"
	"testing"
	"unsafe"
)

func TestTTAllocationAlignmentAndLifetime(t *testing.T) {
	if unsafe.Sizeof(ttRecord{}) != ttRecordBytes {
		t.Fatal("TT byte allocation no longer matches the record layout")
	}
	for _, megabytes := range []int{1, 2, 3, 5, 64, 256} {
		func() {
			c := NewCache(megabytes)
			want := RoundPowerOfTwo(megabytes * 1024 * 1024 / ttRecordBytes)
			if len(c.items) != want || cap(c.items) != want {
				t.Fatalf("Hash%d: len/cap %d/%d, want %d", megabytes, len(c.items), cap(c.items), want)
			}
			address := uintptr(unsafe.Pointer(&c.items[0]))
			if want*ttRecordBytes >= ttHugePageBytes && address%ttHugePageBytes != 0 {
				t.Fatalf("Hash%d: unaligned TT at %#x", megabytes, address)
			}
			for index, item := range c.items {
				if item != (ttRecord{}) {
					t.Fatalf("Hash%d: nonzero initial record %d", megabytes, index)
				}
			}
			c.items[0] = ttRecord{1, 2}
			c.items[len(c.items)-1] = ttRecord{3, 4}
			runtime.GC()
			if c.items[0] != (ttRecord{1, 2}) || c.items[len(c.items)-1] != (ttRecord{3, 4}) {
				t.Fatal("TT backing allocation did not survive GC")
			}
			c.Clear()
			if c.items[0] != (ttRecord{}) || c.items[len(c.items)-1] != (ttRecord{}) {
				t.Fatal("TT clear missed the allocation endpoints")
			}
		}()
	}
}
