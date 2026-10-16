"""Counters and true payload-write ages in a disposable single-thread TT copy."""
from pathlib import Path
import sys
path = Path(sys.argv[1]) / 'engine/cache.go'
s = path.read_text()
def replace(a,b):
    global s
    assert s.count(a)==1, a
    s = s.replace(a,b)
replace('type Cache struct {', 'type Cache struct {\n campaignWritten []uint16')
replace('clear(c.items); c.age = 0', 'clear(c.items); clear(c.campaignWritten); c.age = 0')
replace('current := c.generation()\n\tfirst := c.index(hash)',
        'campaignHistory.Stores++\n current := c.generation()\n if c.campaignWritten == nil { c.campaignWritten=make([]uint16,len(c.items)) }\n first := c.index(hash)')
replace('if ttRetention(incoming, current)+8 >= quality {',
        'if ttRetention(incoming, current)+8 >= quality {\n campaignHistory.MatchingAccepted++\n c.campaignWritten[i]=current')
replace('// A rejected shallow update still proves the entry is current.', '''campaignHistory.MatchingRejected++
                touchAge := (current-uint16(old.payload>>23&255))&255
                payloadAge := (current-c.campaignWritten[i])&255
                if touchAge>0 { campaignHistory.OldTouchRejected++ }
                if payloadAge>0 { campaignHistory.OldPayloadRejected++ }
                if payloadAge>1 { campaignHistory.OlderPayloadRejected++ }
                if payloadAge>0 && int16(old.payload)!=score { campaignHistory.ChangedOldScore++ }
                if payloadAge>campaignHistory.MaxRejectedPayloadAge { campaignHistory.MaxRejectedPayloadAge=payloadAge }
                // A rejected shallow update still proves the entry is current.''')
replace('if ttRetention(incoming, current) >= weakest {\n\t\tc.items[victim] = ttRecord{hash, incoming}\n\t}', '''if ttRetention(incoming, current) >= weakest {
        if c.items[victim].payload&ttOccupied != 0 { campaignHistory.Evictions++ }
        c.campaignWritten[victim]=current
        c.items[victim] = ttRecord{hash, incoming}
    } else { campaignHistory.CollisionRejected++ }''')
replace('move, score, depth, bound, _, pv := Unpack(record.payload)', '''campaignHistory.Hits++
            payloadAge := (c.age-c.campaignWritten[i])&255
            if payloadAge>0 { campaignHistory.OldPayloadHits++ }
            if payloadAge>1 { campaignHistory.OlderPayloadHits++ }
            if uint16(record.payload>>23&255)!=c.age { campaignHistory.OldTouchHits++ }
            if payloadAge>campaignHistory.MaxHitPayloadAge { campaignHistory.MaxHitPayloadAge=payloadAge }
            move, score, depth, bound, _, pv := Unpack(record.payload)''')
path.write_text(s)
