"""Summarize the completed throughput stage and failed node compliance."""
import datetime
import json
from pathlib import Path
import statistics
import sys

p = Path(sys.argv[1])
rows = [json.loads(line) for line in (p/'probe/probes.jsonl').read_text().splitlines()]
assert len(rows) == 384 and all(r['natural_budget_compliance'] for r in rows)
assert len({(r['repetition'],r['position'],r['engine'],r['movetime_ms']) for r in rows}) == 384
cells = {}
for budget in (1000, 5000):
    paired = {(r['repetition'],r['position']):{} for r in rows if r['movetime_ms']==budget}
    for r in rows:
        if r['movetime_ms']==budget:
            paired[(r['repetition'],r['position'])][r['engine']] = r
    assert len(paired)==96 and all(set(v)=={'ngn','opponent'} for v in paired.values())
    ratios = [(v['ngn']['nodes']/v['ngn']['wall_seconds'])/(v['opponent']['nodes']/v['opponent']['wall_seconds']) for v in paired.values()]
    cells[str(budget)] = dict(paired_probes=96,median_ngn_over_rodent_reported_nodes_per_wall_second=statistics.median(ratios),
        median_ngn_reported_nodes=statistics.median([v['ngn']['nodes'] for v in paired.values()]),
        median_rodent_reported_nodes=statistics.median([v['opponent']['nodes'] for v in paired.values()]),
        median_ngn_wall_seconds=statistics.median([v['ngn']['wall_seconds'] for v in paired.values()]),
        median_rodent_wall_seconds=statistics.median([v['opponent']['wall_seconds'] for v in paired.values()]))
compliance = [json.loads(line) for line in (p/'compliance/probes.jsonl').read_text().splitlines()]
assert len(compliance)==2 and compliance[1]['engine']=='opponent'
assert compliance[1]['nodes']>compliance[1]['node_budget']+max(4096,compliance[1]['node_budget']//100)
events = [json.loads(line) for line in (p/'compliance/probe-0-00-opponent.jsonl').read_text().splitlines()]
quit_event = next(e for e in events if e['kind']=='stdin' and e['text']=='quit')
bestmove = next(e for e in events if e['kind']=='stdout' and e['text'].startswith('bestmove ') and e['elapsed_ns']>quit_event['elapsed_ns'])
assert bestmove['elapsed_ns']>quit_event['elapsed_ns']
assert (p/'controller.exit').read_text().strip()=='1'
assert all(line.endswith(': OK') for line in (p/'input-check-terminal.txt').read_text().splitlines())
review = dict(reviewed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),terminal=(p/'controller.done').read_text().strip(),controller_exit=1,
    playing_source='9bfbae4',playing_net='a4d90b41',completed_movetime_probes=384,throughput=cells,
    failed_stage='node-compliance first position/opponent',compliance_rows=compliance,
    opponent_bestmove_after_quit=bestmove,opponent_quit=quit_event,
    inputs_unchanged=True,scored_games=0,verdict='operational failure; no comparison/clock/adoption verdict; no retry or extension',
    limitation='These are reported completed-iteration nodes divided by wall time. Engines count work differently; NGN also has unreported unfinished iteration work. The node/time causal comparison and120+1 clock profiles did not run.')
(p/'review.json').write_text(json.dumps(review,indent=2)+'\n')
print(json.dumps(review,indent=2))
