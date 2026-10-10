"""Read complete black-box cells and their timestamped traffic; no adoption."""
import json
import math
from pathlib import Path
import re
import statistics
import sys


def cell(path, expected_pairs, mode, budget):
    path=Path(path)
    summary=json.loads((path/'summary.json').read_text())
    assert summary['complete_sample'] and summary['pairs']==expected_pairs and summary['faulty_pairs']==0
    pairs={p['pair_id']:p for p in map(json.loads,(path/'pairs.jsonl').read_text().splitlines())}
    assert sorted(pairs)==list(range(expected_pairs))
    assert all(not p['fault'] and len(p['games'])==2 for p in pairs.values())
    moves=[]
    for name in sorted(path.glob('pair-*.jsonl')):
        if name.name=='pairs.jsonl':continue
        events=list(map(json.loads,name.read_text().splitlines()))
        assert sum(e['kind']=='exit' and e['text']=='0' for e in events)==1
        assert not any(e['kind'] in ['forced-kill','stdout-error','start-error'] or e['kind']=='exit' and e['text']!='0' for e in events)
        assert not any(re.search(r'(?i)panic|fatal|illegal|no.move|crash',e['text']) for e in events if e['kind'] in ['stdout','stderr'])
        match=re.fullmatch(r'pair-(\d+)-(ngn|opponent)',name.stem);assert match
        pair=pairs[int(match[1])];actor=match[2]
        game=None;position=None;search=None;searched_move=0;game_index=-1;counts=[]
        for event in events:
            text=event['text'];kind=event['kind']
            if kind=='game':
                game=text;searched_move=0;game_index+=1;counts.append(0)
            elif kind=='stdin' and text.startswith('position '):
                position=text
            elif kind=='stdin' and text.startswith('go ') and game is not None:
                assert search is None
                searched_move+=1
                search=dict(game=game,engine='ngn' if name.stem.endswith('-ngn') else 'opponent',searched_move=searched_move,position=position,go=text,start_ns=event['elapsed_ns'],reported_nodes=0,depth=0)
            elif kind=='stdout' and search is not None:
                fields=text.split()
                if text.startswith('info string ngn_clock '):
                    observation=json.loads(text.removeprefix('info string ngn_clock '))
                    assert observation['iteration_overflow']==0
                    assert 0<=observation['tail_ns']<=observation['wall_ns']
                    search['observer']=observation
                elif fields and fields[0]=='info':
                    for key in ['nodes','depth']:
                        if key in fields:
                            i=fields.index(key)
                            if i+1<len(fields) and fields[i+1].isdigit():
                                target='reported_nodes' if key=='nodes' else key
                                search[target]=max(search[target],int(fields[i+1]))
                elif text.startswith('bestmove '):
                    search['bestmove']=fields[1]
                    search['wall_seconds']=(event['elapsed_ns']-search.pop('start_ns'))/1e9
                    assert search['reported_nodes']>0
                    if mode=='nodes':
                        assert search['go']==f'go nodes {budget}'
                        assert search['reported_nodes']<=budget+max(4096,budget//100)
                    elif mode=='movetime':
                        assert search['go']==f'go movetime {budget}'
                        assert search['wall_seconds']<=budget/1000+.2,search
                    elif mode=='clock':
                        assert search['go'].startswith('go wtime ')
                        if search['engine']=='ngn':assert 'observer' in search, 'missing internal NGN observer'
                        if 'observer' in search:search['reported_nodes']=search['observer']['actual_nodes']
                        args=search['go'].split()[1:];search['clocks_before_ms']={args[i]:int(args[i+1]) for i in range(0,len(args),2)}
                    else:raise ValueError(mode)
                    moves.append(search);counts[game_index]+=1;search=None
        assert search is None and game_index==1
        opening=pair['opening'];open_len=0 if '/' in opening else len(opening.split())
        initial_white=opening.split()[1]=='w' if '/' in opening else True
        next_white=initial_white if open_len%2==0 else not initial_white
        for index,g in enumerate(pair['games']):
            plies=len(g['moves'])-open_len;assert plies>=0
            actor_white=g['ngn_white'] if actor=='ngn' else not g['ngn_white']
            expected=(plies+1)//2 if actor_white==next_white else plies//2
            assert counts[index]==expected,(name,index,counts[index],expected)
    with (path/'moves.jsonl').open('w') as f:
        for move in moves:f.write(json.dumps(move)+'\n')
    return summary,pairs,moves


def compare(nodes_path,time_path,budget):
    n,npairs,_=cell(nodes_path,100,'nodes',budget)
    t,tpairs,_=cell(time_path,100,'movetime',1000)
    def score(pair):return sum(g['result']['Res']/2 for g in pair['games'])/2
    delta=[]
    for i in range(100):
        assert npairs[i]['opening']==tpairs[i]['opening']
        delta.append(score(npairs[i])-score(tpairs[i]))
    mean=statistics.mean(delta)
    margin=1.9842169515086827*statistics.stdev(delta)/math.sqrt(100)
    elo_difference=n['elo']-t['elo']
    return dict(equal_nodes=n,equal_time=t,paired_opening_score_difference=mean,paired_difference_ci95=[mean-margin,mean+margin],elo_difference=elo_difference,decision='throughput-contribution-supported' if elo_difference>=10 and mean-margin>0 else 'unresolved',predeclared_materiality_elo=10,limitation='Reported nodes are engine-defined; equal nodes does not separate evaluation from search. No Elo decomposition, adoption, or absolute rating.')


def node_budget(path):
    rows=list(map(json.loads,(Path(path)/'probes.jsonl').read_text().splitlines()))
    samples=[r['nodes'] for r in rows if r.get('engine')=='ngn' and r.get('movetime_ms')==1000]
    assert len(samples)==96 and all(n>0 for n in samples)
    assert len(rows)==384 and all(r['natural_budget_compliance'] for r in rows)
    return int(statistics.median(samples))


if __name__=='__main__':
    if sys.argv[1]=='budget':print(json.dumps(dict(common_node_budget=node_budget(sys.argv[2]),selection='integer median of96 NGN1s reported node counts, frozen before any scored games')))
    elif sys.argv[1]=='compare':print(json.dumps(compare(sys.argv[2],sys.argv[3],int(sys.argv[4])),indent=2))
    elif sys.argv[1]=='aa':
        summary,_,_=cell(sys.argv[2],20,sys.argv[3],int(sys.argv[4]))
        assert summary['elo_lower95']<=0<=summary['elo_upper95']
        print(json.dumps(dict(verdict='aa-pass',summary=summary)))
    elif sys.argv[1]=='clock':
        summary,_,moves=cell(sys.argv[2],12,'clock',120000)
        checkpoints={str(n):[m for m in moves if m['searched_move']==n] for n in [10,20,30,40,60]}
        print(json.dumps(dict(summary=summary,checkpoints=checkpoints,ngn_tail_fraction=sum(m['observer']['tail_ns'] for m in moves if m['engine']=='ngn')/sum(m['observer']['wall_ns'] for m in moves if m['engine']=='ngn'),observer_build='ngnclocktrace diagnostic only; ordinary search path with existing observer attached'),indent=2))
    else:raise ValueError(sys.argv[1])
