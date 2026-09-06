/*
https://github.com/amanjpro/zahak/?tab=MIT-1-ov-file#readme
MIT License

Copyright (c) 2021 Amanj Sherwany

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package engine

import (
	"unsafe"
	// 	"fmt"
)

type CachedEval struct {
	Key  uint64 // 8
	Data uint64 // 8
	// Plain uint64, NOT atomic: NGN's search is single-threaded (the Threads UCI
	// option is a no-op stub and the TT is touched only by the lone `go uci.search`
	// goroutine), so atomic loads/stores were pure overhead (~4.5% of search CPU,
	// measured) buying nothing. No cache-line padding either: a 16-byte entry
	// quadruples slot count at the same MB (128MB: 2.1M -> 8.4M), which the endgame
	// audit showed cuts the middlegame tree materially. The lockless XOR-verify
	// (Key=Data^hash) is kept as a cheap collision checksum — correct single-threaded.
}

type NodeType uint8

const (
	Exact      NodeType = 1 << iota // PV-Node
	UpperBound                      // All-Node
	LowerBound                      // Cut-Node
)

type Cache struct {
	items  []CachedEval
	size   int
	length uint64
	mask   uint
	age    uint16
}

const OldAge = uint16(5)
const DEFAULT_CACHE_SIZE = 128
const MAX_CACHE_SIZE = 120_000

var CACHE_ENTRY_SIZE = int(unsafe.Sizeof(CachedEval{}))
var TranspositionTable = NewCache(DEFAULT_CACHE_SIZE)

func (c *CachedEval) LoadKey() uint64 {
	return c.Key
}

func (c *CachedEval) LoadData() uint64 {
	return c.Data
}

const MOVE_MASK uint64 = 0b1111111111111111111111111111 // move << 0, 28 bits
const EVAL_MASK uint64 = 0b1111111111111111             // eval << 28, 16 bits
const DEPTH_MASK uint64 = 0b1111111                     // depth << 44, 7 bits
const TYPE_MASK uint64 = 0b111                          // type << 51, 3 bits
const AGE_MASK uint64 = 0b111111111                     // age << 54, 9 bits (T19: 1 bit ceded to ttPv)
const TTPV_MASK uint64 = 0b1                            // ttPv << 63, 1 bit

func Pack(hashmove Move, eval int16, depth int8, nodeType NodeType, age uint16, ttPv bool) uint64 {
	var pv uint64
	if ttPv {
		pv = 1
	}
	return (uint64(hashmove) & MOVE_MASK) |
		((uint64(eval) & EVAL_MASK) << 28) |
		((uint64(depth) & DEPTH_MASK) << 44) |
		((uint64(nodeType) & TYPE_MASK) << 51) |
		((uint64(age) & AGE_MASK) << 54) |
		((pv & TTPV_MASK) << 63)
}

func Unpack(data uint64) (hashmove Move, eval int16, depth int8, nodeType NodeType, age uint16, ttPv bool) {
	hashmove = Move(data & MOVE_MASK)
	eval = int16((data >> 28) & EVAL_MASK)
	depth = int8((data >> 44) & DEPTH_MASK)
	nodeType = NodeType((data >> 51) & TYPE_MASK)
	age = uint16((data >> 54) & AGE_MASK)
	ttPv = ((data >> 63) & TTPV_MASK) != 0
	return
}

func (c *CachedEval) Update(key uint64, data uint64) {
	c.Key = key
	c.Data = data
}

func (c *Cache) AdvanceAge() {
	c.age += 1
	if c.age > uint16(0b1111111111) {
		c.age = 0
	}
}

func (c *Cache) Consumed() int {
	used := 0
	samples := 1000

	for i := 0; i < samples; i++ {
		if c.items[i].LoadData() != 0 {
			used++
		}
	}

	return used / (samples / 1000)
}

func (c *Cache) index(hash uint64) uint {
	return uint(hash) & c.mask
}

func (c *Cache) Set(hash uint64, hashmove Move, eval int16, depth int8, nodeType NodeType, ttPv bool) {
	index := c.index(hash)
	age := c.age

	oldValue := &c.items[index]
	oldKey := oldValue.LoadKey()
	oldData := oldValue.LoadData()
	oldHash := oldKey ^ oldData

	// very good for debugging hash issues
	// newHashmove, newEval, newDepth, newNodeType, newAge := Unpack(newData)
	// if hashmove != newHashmove || eval != newEval || depth != newDepth || nodeType != newNodeType || age != newAge {
	// 	panic(fmt.Sprintf(
	// 		"Culprits are: %d %d %d %d %d\nSomehow became: %d %d %d %d %d\n", hashmove, eval, depth, nodeType, age, newHashmove, newEval, newDepth, newNodeType, newAge))
	// }

	_, _, oldDepth, _, oldAge, _ := Unpack(oldData)
	var replace bool
	if oldData == 0 {
		replace = true
	} else if oldHash == hash {
		replace = depth >= oldDepth-3 || nodeType == Exact
	} else {
		replace = oldAge != age || depth >= oldDepth
	}
	if replace {

		newData := Pack(hashmove, eval, depth, nodeType, age, ttPv)
		newKey := newData ^ hash
		c.items[index].Update(newKey, newData)
	}
}

func (c *Cache) Size() int {
	return c.size
}

// Clear invalidates every entry while preserving the configured allocation.
// Callers must ensure no search is concurrently reading or writing this table.
func (c *Cache) Clear() {
	clear(c.items)
	c.age = 0
}

func (c *Cache) Get(hash uint64) (Move, int16, int8, NodeType, bool, bool) {
	index := c.index(hash)
	value := &c.items[index]
	data := value.LoadData()
	key := value.LoadKey()
	ok := hash == (key ^ data)
	if ok {
		move, eval, depth, nType, _, ttPv := Unpack(data)
		return move, eval, depth, nType, true, ttPv
	}
	return 0, 0, 0, 0, false, false
}

func NewCache(megabytes int) *Cache {
	if megabytes < 1 {
		return nil
	}
	size := int((megabytes * 1024 * 1024) / CACHE_ENTRY_SIZE)
	length := RoundPowerOfTwo(size)

	return &Cache{make([]CachedEval, length), megabytes, uint64(length), uint(length - 1), 0}
}

func RoundPowerOfTwo(size int) int {
	var x = 1
	for (x << 1) <= size {
		x <<= 1
	}
	return x
}
