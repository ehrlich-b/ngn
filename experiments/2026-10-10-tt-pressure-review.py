"""Summarize the frozen diagnostic cells; no playing-strength inference."""
import datetime
import json
from pathlib import Path
import statistics
import sys

p = Path(sys.argv[1]).resolve()
assert (p/'pressure.exit').read_text().strip() == '0'
results = []
for design in ('legacy', 'own'):
    for keys in ('old', 'new'):
        rows = [json.loads(line) for line in (p/f'{design}-{keys}.jsonl').read_text().splitlines()]
        assert len(rows) == 16 and {row['depth'] for row in rows} == {16, 20}
        assert len({row['fen'] for row in rows}) == 8
        final = [row for row in rows if row['depth'] == 20]
        assert len(final) == 8
        assert all(row['tt_design'] == design and row['keys'] == keys for row in rows)
        for row in rows:
            c = row['counters']
            assert c['Probes'] >= c['Hits']
            assert c['MatchingStores'] + c['MissStores'] == c['Stores']
            assert sum(c['StoreOccupancy']) == c['Stores']
            assert sum(c[name] for name in ('Inserts','Evictions','SameKeyWrites','UnchangedMatchingStores','RejectedMissStores')) == c['Stores']
            assert sum(row['bucket_occupancy_histogram'])*row['ways'] == row['allocated_records']
        counters = {name:sum(row['counters'][name] for row in final) for name in final[0]['counters'] if name != 'StoreOccupancy'}
        occupancy = [sum(i*n for i,n in enumerate(row['bucket_occupancy_histogram']))/row['allocated_records'] for row in final]
        full = [row['bucket_occupancy_histogram'][row['ways']]/sum(row['bucket_occupancy_histogram']) for row in final]
        results.append(dict(tt=design,keys=keys,depth=20,positions=8,nodes=sum(row['nodes'] for row in final),counters=counters,
                            hit_rate=counters['Hits']/counters['Probes'],miss_store_refusal_rate=counters['RejectedMissStores']/counters['MissStores'],
                            evictions_per_store=counters['Evictions']/counters['Stores'],mean_record_occupancy=statistics.mean(occupancy),mean_full_bucket_fraction=statistics.mean(full)))
report = dict(reviewed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),complete=True,raw_counted_depth10_identity=True,cells=results,
              interpretation='Fixed cold positions only. TT designs change searched nodes; hit rates and churn are mechanism diagnostics, never Elo or speed verdicts. Key assignment effects are also position-specific.',playing_adoption=False)
(p/'pressure-review.json').write_text(json.dumps(report,indent=2)+'\n')
print(json.dumps(report,indent=2))
