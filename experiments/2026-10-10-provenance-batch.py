"""Review the pinned harness's terminal combined-certification evidence."""
import collections
import json
import math
from pathlib import Path
import re

text = Path('gate.log').read_text()
assert not re.search(r'(?im)panic|fatal|illegal|no.move|watchdog|timeout|crash|forfeit|engine.error|voided', text)
summaries = re.findall(r'Games: (\d+)\s+W-D-L: (\d+)-(\d+)-(\d+)', text)
assert len(summaries) == 1
games, w, d, l = map(int, summaries[0])
assert 400 <= games <= 4000 and games == w+d+l
rows = re.findall(r'(?m)^G(\d+)\s+(\d+)W\s+(\d+)D\s+(\d+)L.*?\s+(\S+)\s*$', text)
assert len(rows) == games
previous = (0, 0, 0)
reasons = collections.Counter()
for index, (n, wi, di, li, reason) in enumerate(rows, 1):
    counts = tuple(map(int, (wi, di, li)))
    assert int(n) == sum(counts) == index
    assert sorted(c-p for c,p in zip(counts,previous)) == [0,0,1]
    assert reason in ('checkmate','draw-rule','max-moves','stalemate')
    reasons[reason] += 1
    previous = counts
assert previous == (w,d,l)
pentas = re.findall(r'Pentanomial \[LL (\d+)\s+LD (\d+)\s+\{LW,DD\} (\d+)\s+WD (\d+)\s+WW (\d+)\] over (\d+) pairs', text)
assert len(pentas) == 1
*penta, pairs = map(int,pentas[0])
assert sum(penta) == pairs and 0 <= games-2*pairs < 12
mu = sum(i*c/4 for i,c in enumerate(penta))/pairs
variance = max(sum((i/4)**2*c for i,c in enumerate(penta))/pairs-mu**2,.001)
error = 1.96*math.sqrt(variance/pairs)
def elo(score):
    if score <= 1e-4: return -800.
    if score >= 1-1e-4: return 800.
    return 400*math.log10(score/(1-score))
exact = [elo(mu),elo(mu-error),elo(mu+error)]
estimates = re.findall(r'Penta Elo: ([-+\d.]+)\s+95% CI \[([-+\d.]+),\s*([-+\d.]+)\]',text)
assert len(estimates)==1
reported = list(map(float,estimates[0]))
assert abs(exact[0]-reported[0]) <= .051
assert all(abs(a-b)<=.501 for a,b in zip(exact[1:],reported[1:]))
flags = re.findall(r'Flag-outs \(lost on time\): new (\d+), base (\d+)\s+of (\d+) games',text)
assert len(flags)==1 and tuple(map(int,flags[0]))==(0,0,games)
assert not re.search(r'PANIC|RECOVERED|FATAL',Path('ngn_crashes.log').read_text())
terminal = re.findall(r'^Verdict: (.+)$',text,re.M)
assert len(terminal)==1
if terminal[0].startswith('H1 ACCEPTED'):
    verdict = 'certification-eligible' if exact[1]>0 else 'requires-bisect'
elif terminal[0].startswith('H0 ACCEPTED'):
    verdict = 'requires-bisect'
else:
    assert terminal[0].startswith('INCONCLUSIVE') and games==4000
    verdict = 'requires-bisect'
s0=.5;s1=1/(1+10**(-6/400))
result=dict(games=games,pairs=pairs,unpaired_games=games-2*pairs,wins=w,draws=d,losses=l,penta=penta,exact_elo_ci95=exact,penta_llr=(s1-s0)/variance*(pairs*mu-pairs*(s0+s1)/2),flags=[0,0],reasons=dict(reasons),harness_verdict=terminal[0],verdict=verdict,automatic_promotion=False)
print(json.dumps(result,indent=2))
