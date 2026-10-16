"""Descriptive matched-root summary; no policy or Elo acceptance."""
from collections import defaultdict
from pathlib import Path
import datetime,json,sys
D=Path(sys.argv[1]); rows=[json.loads(x) for x in (D/'history.jsonl').read_text().splitlines()]
pairs=defaultdict(dict)
for row in rows: pairs[(row['fixture'],row['hash_mb'],row['ply'])][row['arm']]=row
assert all(set(p)=={'warm','cold'} for p in pairs.values())
summary=[]
for size in (64,256):
    panel=[p for (_,h,ply),p in pairs.items() if h==size and ply>=2]
    counts=defaultdict(int)
    for p in panel:
        for key,value in p['warm']['pressure'].items():
            if key.startswith('Max'): counts[key]=max(counts[key],value)
            else: counts[key]+=value
    summary.append(dict(hash_mb=size,matched_roots=len(panel),warm_counts=dict(counts),
        depth_warm_minus_cold=sum(p['warm']['depth']-p['cold']['depth'] for p in panel),
        warm_deeper=sum(p['warm']['depth']>p['cold']['depth'] for p in panel),
        cold_deeper=sum(p['warm']['depth']<p['cold']['depth'] for p in panel),
        same_depth=sum(p['warm']['depth']==p['cold']['depth'] for p in panel),
        best_move_disagreements=sum(p['warm']['best']!=p['cold']['best'] for p in panel),
        mean_abs_score_difference=sum(abs(p['warm']['score']-p['cold']['score']) for p in panel)/len(panel)))
result=dict(state='DIAGNOSED',roots=len(rows),matched_roots=len(pairs),summary=summary,
    identity=json.loads((D/'identity.json').read_text()),
    interpretation='Persistent current-owned TT versus empty TT, identical root/history/node budget; no correctness, speed, Elo or policy verdict. Payload age is diagnostic metadata, not a proposed retention rule.',
    finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
(D/'result.json').write_text(json.dumps(result,indent=2)+'\n')
with (D.parent.parent/'status.jsonl').open('a') as f:
    f.write(json.dumps(dict(utc=result['finished_utc'],stage='session89-tt-history',event='complete',detail=result))+'\n')
