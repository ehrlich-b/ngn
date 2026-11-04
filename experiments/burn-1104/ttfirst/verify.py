#!/usr/bin/env python3
"""Inspect provenance, pinned references, experiment boundaries and qualification."""
import hashlib
import json
import os
import pathlib
import socket
import subprocess
import sys
root=pathlib.Path(__file__).resolve().parents[3]
folder=root/'experiments/burn-1104/ttfirst'
base='faaf3741eaf0e2add133ffe6574a3c24db415907'
previous='fbe5458b270d8e6cd384234728ceb4bc1073a025'
# Integrated product tree before this experiment's isolation-check adjustment.
default_baseline='5aebe1ed1ae35e317a830afdf2ceebeb849539c4'
patch=folder/'candidate.patch'
mode=sys.argv[1]
def require(ok,message):
    if not ok: sys.exit(message)
def git(*args,cwd=root):
    return subprocess.check_output(['git',*args],cwd=cwd,text=True).strip()
def body(text,start,end):
    return text[text.index(start):text.index(end,text.index(start))].strip()
if mode=='previous':
    source=root.parent/'sources/previous'
    require(git('rev-parse','HEAD',cwd=source)==previous,'wrong previous checkout')
    original=git('show',previous+':experiments/burn-1104/ttfirst/candidate.patch',cwd=source)+'\n'
    require(hashlib.sha256(original.encode()).hexdigest()=='137134fea6bec5f92824f54094ccc2956374127fd8984a6bd97c536251f43ce6','wrong prior patch')
    require('ttMoveGeneratedLegal' in patch.read_text(),'prior oracle not retained')
    print('previous checkout='+previous+'; prior generated oracle retained as test-only reference')
elif mode=='host':
    require(os.getuid()!=0,'must run unprivileged')
    print('hostname='+socket.gethostname()+' uid='+str(os.getuid())+'; ordinary Go gates only; no kernel/device/root operations')
elif mode in ('integration','blockers','patch','focused','corpus'):
    text=patch.read_text()
    if mode=='patch': subprocess.run(['git','apply','--check','--whitespace=error-all',str(patch)],cwd=root,check=True)
    elif mode in ('integration','blockers'):
        for marker in ('diff --git a/engine/search.go b/engine/search.go','fastTT := ttHit && ttMoveLegal(pos, ttMove)','if fastTT && oi == 1','ttSnapshot.context().score','ordered[0], ordered[ttIndex] = ordered[ttIndex], ordered[0]','s.quiet, s.capture, s.killers, s.counter = *c.quiet, *c.capture, c.killers, c.counter','TestTTFirstCounters'):
            require(marker in text,'missing integration: '+marker)
        require((folder/'mutations.py').is_file() and (folder/'compare.py').is_file(),'review blocker fixtures missing')
    elif mode=='focused':
        for marker in ('TestTTFirstValidatorOracle','TestTTFirstCounters','TestTTFirstMutation','TestTTFirstOrder','ttReferenceAlphaBetaPV','ttDecisionState','reflect.DeepEqual(a.history, b.history)'):
            require(marker in text,'missing focused fixture: '+marker)
    elif mode=='corpus':
        fens=json.loads((folder/'corpus.json').read_text())
        require(len(fens)==len(set(fens))==12,'expected twelve unique positions')
        require(any(' d6 ' in f for f in fens) and any(' d3 ' in f for f in fens) and any(' KQkq ' in f for f in fens),'special fixtures missing')
elif mode=='reference':
    source,candidate=map(pathlib.Path,sys.argv[2:])
    pinned=git('show',base+':engine/search.go',cwd=source)
    order=git('show',base+':engine/moveorder.go',cwd=source)
    expected=body(pinned,'func alphaBetaPV(','\n// depthBand').replace('alphaBetaPV(','ttReferenceAlphaBetaPV(').replace('.scoreMovesIntoBuffer(','.ttReferenceScoreMovesIntoBuffer(')
    expected+='\n\n'+body(order,'func (h *workerHistory) scoreMovesIntoBuffer(','\n}\n')+'\n}'
    expected=expected.replace('func (h *workerHistory) scoreMovesIntoBuffer(','func (h *workerHistory) ttReferenceScoreMovesIntoBuffer(')
    actual=(candidate/'engine/ttfirst_reference_test.go').read_text().split('package engine\n\n',1)[1].strip()
    require(actual==expected,'reference is not exact pinned baseline functions')
    print('pinned alphaBetaPV and eager scorer bodies match exactly')
elif mode=='default':
    changed=git('diff','--name-only',default_baseline,'--','.',':(exclude)experiments/burn-1104/ttfirst')
    require(not changed,'default adoption detected: '+changed)
    require(not git('status','--porcelain','--','engine','cmd'),'default engine dirty')
    print('default tracked tree unchanged outside experiments/burn-1104/ttfirst')
elif mode=='manifest':
    m=json.loads((folder/'manifest.json').read_text())
    require(m['base']==base and m['previous']==previous,'provenance missing')
    require(m['patch_sha256']==hashlib.sha256(patch.read_bytes()).hexdigest(),'patch hash mismatch')
    prior=json.loads((root.parent/'evidence/previous-manifest.json').read_text())
    require(m['qualification']==prior['qualification'],'original qualification changed')
    require(m['closes_original_card'] is False and m['engine_improvement_claim'] is False,'invalid adoption claim')
elif mode=='authors':
    result=git('log','--format=%an <%ae>|%cn <%ce>',base+'..HEAD').splitlines()
    require(result and all(x=='Bryan Ehrlich <ehrlich.bryan@gmail.com>|Bryan Ehrlich <ehrlich.bryan@gmail.com>' for x in result),'unexpected commit identity')
else: sys.exit('unknown check')
print(mode+': OK')
