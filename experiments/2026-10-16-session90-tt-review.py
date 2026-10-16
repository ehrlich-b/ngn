"""Review the completed persistent-TT panel without selecting an engine policy."""
from collections import defaultdict
from pathlib import Path
import datetime
import json
import statistics
import sys

D = Path(sys.argv[1])
pairs = defaultdict(dict)
for line in (D / 'history.jsonl').read_text().splitlines():
    row = json.loads(line)
    pairs[row['fixture'], row['hash_mb'], row['ply']][row['arm']] = row
assert len(pairs) == 282 and all(set(p) == {'warm', 'cold'} for p in pairs.values())
result = json.loads((D / 'result.json').read_text())
summary = []
for cell in result['summary']:
    size = cell['hash_mb']
    panel = [p for (_, h, ply), p in pairs.items() if h == size and ply >= 2]
    finite = [p for p in panel if max(abs(p[a]['score']) for a in ('warm', 'cold')) < 29000]
    c = cell['warm_counts']
    summary.append(dict(
        hash_mb=size, post_control_pairs=len(panel),
        older_payload_hit_fraction=c['OldPayloadHits'] / c['Hits'],
        older_payload_rejected_store_fraction=c['OldPayloadRejected'] / c['MatchingRejected'],
        warm_deeper=cell['warm_deeper'], cold_deeper=cell['cold_deeper'],
        same_depth=cell['same_depth'], move_disagreements=cell['best_move_disagreements'],
        nonmate_pairs=len(finite),
        nonmate_mean_abs_score_difference=statistics.mean(abs(p['warm']['score'] - p['cold']['score']) for p in finite),
        nonmate_median_abs_score_difference=statistics.median(abs(p['warm']['score'] - p['cold']['score']) for p in finite),
        mate_range_pairs=len(panel) - len(finite),
        mate_range_score_gap_over_100_fixtures=sorted({
            p['warm']['fixture'] for p in panel
            if max(abs(p[a]['score']) for a in ('warm', 'cold')) >= 29000
            and abs(p['warm']['score'] - p['cold']['score']) > 100
        }),
    ))
review = dict(
    reviewed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
    source='9ec5d31', net='81215229', vector='compiled final14',
    terminal='2026-10-16 04:22:16 EDT; controller exit0',
    search_rows=result['roots'], matched_roots=len(pairs),
    instrumentation_identity=result['identity'], summary=summary,
    finding='Persistent TT usually completes a greater depth at the matched node budget. Roughly half of hits use a payload from an earlier search. Large aggregate score gaps are dominated by mate-range results in the queen fixture. Neither old payload reuse nor a warm/cold disagreement proves incorrect retention.',
    limitations='Six selected trajectories, 2M nodes/root, up to24 plies. Descriptive, not randomized games, not 60+0.6, not an archived-TT comparison. Nominal depth is not strength or proof of correctness. The panel does not identify which replacement caused the certification loss.',
    decision='No engine policy nomination or adoption from this panel; no new provenance games or unchanged certificate retry. Keep owned replacements and certification debt. Advance existing width, Final174 and tuning order.',
    reopen='A concrete reproducible TT invariant violation, or a separately predeclared changed-policy experiment with mechanism evidence and the appropriate real-clock gate. Do not repeat cold/persistent panels simply to refresh custody.',
)
(D / 'review-session90.json').write_text(json.dumps(review, indent=2) + '\n')
print(json.dumps(review, indent=2))
