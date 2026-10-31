"""Add counters only to a disposable accepted-source copy for H0 diagnosis."""
from pathlib import Path
import sys

root = Path(sys.argv[1])
path = root / 'engine/cache.go'
source = path.read_text()

def replace(old, new):
    global source
    assert source.count(old) == 1, old
    source = source.replace(old, new)

replace('current := c.generation()\n\tfirst := c.index(hash)',
        'campaignPressure.Stores++\n\tcurrent := c.generation()\n\tfirst := c.index(hash)')
replace('if old.hash == hash {', 'if old.hash == hash {\n\t\t\tcampaignPressure.Matching++')
replace('if ttRetention(incoming, current)+8 >= quality {',
        'if ttRetention(incoming, current)+8 >= quality {\n\t\t\t\tcampaignPressure.MatchingAccepted++')
replace('// A rejected shallow update still proves the entry is current.',
        '''campaignPressure.MatchingRejected++
                if uint16(old.payload>>23&255) != current { campaignPressure.OldMatchingRejected++ }
                if int16(old.payload) != score { campaignPressure.RejectedDifferentScore++ }
                if Move(old.payload>>32&ttMoveMask) != move { campaignPressure.RejectedDifferentMove++ }
                // A rejected shallow update still proves the entry is current.''')
replace('if ttRetention(incoming, current) >= weakest {\n\t\tc.items[victim] = ttRecord{hash, incoming}\n\t}',
        '''if ttRetention(incoming, current) >= weakest {
        if c.items[victim].payload&ttOccupied != 0 { campaignPressure.Evictions++ } else { campaignPressure.EmptyInserts++ }
        c.items[victim] = ttRecord{hash, incoming}
    } else { campaignPressure.CollisionRejected++ }''')
replace('record := c.items[i]\n\t\tif record.payload&ttOccupied != 0 && record.hash == hash {',
        '''record := c.items[i]
        campaignPressure.ProbeSlots++
        if record.payload&ttOccupied != 0 && record.hash != hash { campaignPressure.OccupiedMismatches++ }
        if record.payload&ttOccupied != 0 && record.hash == hash {''')
path.write_text(source)
