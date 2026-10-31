"""Reuse the fixed-gate parser for the predeclared 1000-game diagnosis."""
import importlib.util
import json
from pathlib import Path
import re
import sys

spec=importlib.util.spec_from_file_location('fixed_gate',Path(__file__).with_name('gate-parser.py'))
parser=importlib.util.module_from_spec(spec)
spec.loader.exec_module(parser)
mode,path=sys.argv[1:]
assert mode in ('aa','gate')
summary=parser.gate(Path(path).read_text(),mode)
assert not re.search(r'PANIC|RECOVERED|FATAL',Path('ngn_crashes.log').read_text())
if mode=='gate':
 lower,upper=summary['ci95']
 _,exact_lower,exact_upper=summary['exact_elo_ci95']
 summary['verdict']=('SEE-regression-implicated' if upper<0 and exact_upper<0 else
                     'SEE-positive' if lower>0 and exact_lower>0 else 'SEE-unresolved')
 summary['playing_adoption']=False
print(json.dumps(summary,indent=2))
