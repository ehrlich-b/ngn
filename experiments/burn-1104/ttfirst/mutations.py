#!/usr/bin/env python3
"""Seed concrete source faults in separate private copies; require test rejection."""
import os
import pathlib
import shutil
import subprocess
import sys
source, evidence = map(pathlib.Path,sys.argv[1:])
mutations = [
 ('eager-main-list', 'search.go', 'fastTT := ttHit && ttMoveLegal(pos, ttMove)', 'fastTT := false', '^TestTTFirstCounters$'),
 ('legal-counter', 'search.go', 'legalTried++', 'legalTried += 2', '^TestTTFirstCounters$'),
 ('live-history', 'ttfirst_validator.go', 's.quiet, s.capture, s.killers, s.counter = *c.quiet, *c.capture, c.killers, c.counter', 's.quiet, s.capture, s.killers, s.counter = [13][64]int{}, *c.capture, c.killers, c.counter', '^TestTTFirstOrder$'),
 ('packed-bit', 'ttfirst_validator.go', 'uint32(m)>>27 != 0', 'false', '^TestTTFirstMutation$'),
]
for name, file, old, new, test in mutations:
 target=source.parent / ('mutation-'+name)
 # Files remain available for diagnosis. Never alter the tested candidate.
 shutil.copytree(source,target,ignore=shutil.ignore_patterns('.git'))
 p=target/'engine'/file
 text=p.read_text()
 assert text.count(old)==1,(name,text.count(old))
 p.write_text(text.replace(old,new))
 argv=['timeout','120s','nice','-n','10','go','test','-p','2','-short','-count=1','-failfast','./engine','-run',test,'-v']
 with (evidence/('mutation-'+name+'.log')).open('w') as log:
  log.write('seed=1104; '+repr(argv)+'\n'); log.flush()
  result=subprocess.run(argv,cwd=target,stdout=log,stderr=subprocess.STDOUT)
 output=(evidence/('mutation-'+name+'.log')).read_text()
 assert result.returncode==1 and '--- FAIL: TestTTFirst' in output and '[build failed]' not in output,(name,result.returncode)
 print('PASS: seed=1104 source mutation rejected:',name)
