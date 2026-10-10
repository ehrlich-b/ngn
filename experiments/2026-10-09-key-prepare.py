"""Prepare two diagnostic-only builds from the same accepted playing source."""
from pathlib import Path
import shutil

r = Path.cwd()
for kind in ("oldkey", "newkey"):
    target = r / kind
    shutil.copytree(r / "baseline", target)
    shutil.copyfile(r / "key-diagnostics-test.go", target / "engine/campaign_key_test.go")
    path = target / "engine/cache.go"
    text = path.read_text()
    text = text.replace("type ttRecord struct", "var ngnKeyStats struct { Probes, Hits, IndexCollisions, Evictions uint64 }\n\ntype ttRecord struct", 1)
    text = text.replace("func (c *Cache) Get(hash uint64) (Move, int16, int8, NodeType, bool, bool) {", "func (c *Cache) Get(hash uint64) (Move, int16, int8, NodeType, bool, bool) {\n\tngnKeyStats.Probes++", 1)
    text = text.replace("\t\tif record.payload&ttOccupied != 0 && record.hash == hash {", "\t\tif record.payload&ttOccupied != 0 && record.hash != hash { ngnKeyStats.IndexCollisions++ }\n\t\tif record.payload&ttOccupied != 0 && record.hash == hash {\n\t\t\tngnKeyStats.Hits++", 1)
    text = text.replace("\t\tc.items[victim] = ttRecord{hash, incoming}", "\t\tif c.items[victim].payload&ttOccupied != 0 && c.items[victim].hash != hash { ngnKeyStats.Evictions++ }\n\t\tc.items[victim] = ttRecord{hash, incoming}", 1)
    assert text.count("ngnKeyStats.") == 4
    path.write_text(text)

# Only init's seed/draw order differs. Preserve all actual-base hash operations.
path = r / "newkey/engine/hash.go"
text = path.read_text()
start = text.index("func init() {")
end = text.index("\nfunc epTargetKey", start)
init = (r / "new-key-init.txt").read_text()
path.write_text(text[:start] + init + "\n" + text[end:])
