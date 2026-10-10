"""Instrument archived NGN snapshots without copying either TT implementation."""
from pathlib import Path
import sys

s = Path(sys.argv[1])
for design in ('legacy', 'own'):
    root = s / design
    p = root / 'engine/cache.go'
    code = p.read_text()
    assert code.count('func (c *Cache) Get(') == code.count('func (c *Cache) Set(') == 1
    code = code.replace('func (c *Cache) Get(', 'func (c *Cache) campaignPressureGetRaw(')
    code = code.replace('func (c *Cache) Set(', 'func (c *Cache) campaignPressureSetRaw(')
    p.write_text(code)
    (root / 'engine/campaign_pressure_test.go').write_bytes((s / 'pressure-test.go.txt').read_bytes())
    if design == 'own':
        support = '''package engine
const campaignPressureWays = 4
func (c *Cache) campaignPressureBucket(hash uint64) (rows [4]campaignPressureEntry) {
    first:=c.index(hash)
    for i:=0;i<4;i++ {r:=c.items[first+uint(i)];rows[i]=campaignPressureEntry{r.hash,r.payload,r.payload&ttOccupied!=0}}
    return
}
func (c *Cache) campaignPressureOccupancy() (histogram [5]uint64) {
    for first:=0;first<len(c.items);first+=4 {n:=0;for i:=0;i<4;i++{if c.items[first+i].payload&ttOccupied!=0{n++}};histogram[n]++}
    return
}
'''
    else:
        support = '''package engine
const campaignPressureWays = 1
func (c *Cache) campaignPressureBucket(hash uint64) (rows [4]campaignPressureEntry) {
    r:=c.items[c.index(hash)];rows[0]=campaignPressureEntry{r.Key^r.Data,r.Data,r.Data!=0};return
}
func (c *Cache) campaignPressureOccupancy() (histogram [5]uint64) {
    for _,r:=range c.items{if r.Data!=0{histogram[1]++}else{histogram[0]++}};return
}
'''
    (root / 'engine/campaign_pressure_support_test.go').write_text(support)
