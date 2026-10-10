import json
from pathlib import Path
import tempfile
import unittest
import review

class TrafficReview(unittest.TestCase):
    def fixture(self,p):
        pair=dict(pair_id=0,opening='e2e4',fault=False,games=[dict(ngn_white=white,moves=['e2e4','e7e5','g1f3'],result={'Res':1,'Reason':'max-moves'}) for white in [True,False]])
        (p/'summary.json').write_text(json.dumps(dict(complete_sample=True,pairs=1,faulty_pairs=0)))
        (p/'pairs.jsonl').write_text(json.dumps(pair)+'\n')
        for actor in ['ngn','opponent']:
            events=[]
            def add(kind,text):events.append(dict(kind=kind,text=text,elapsed_ns=len(events)*10_000_000))
            for white in [True,False]:
                add('game',f'pair=0 ngn_white={str(white).lower()}')
                add('stdin','position startpos moves e2e4')
                add('stdin','go wtime 120000 btime 120000 winc 1000 binc 1000')
                add('stdout','info depth 4 nodes 400 time 30 pv e7e5')
                if actor=='ngn':add('stdout','info string ngn_clock '+json.dumps(dict(actual_nodes=800,iteration_overflow=0,tail_ns=30_000_000,wall_ns=80_000_000)))
                add('stdout','bestmove e7e5')
            add('exit','0')
            (p/f'pair-000-{actor}.jsonl').write_text(''.join(json.dumps(e)+'\n' for e in events))
    def test_clock_includes_partial_iteration_nodes_and_observer(self):
        with tempfile.TemporaryDirectory() as td:
            p=Path(td);self.fixture(p)
            _,_,moves=review.cell(p,1,'clock',120000)
            self.assertEqual([m['reported_nodes'] for m in moves if m['engine']=='ngn'],[800,800])
    def test_missing_game_traffic_rejected(self):
        with tempfile.TemporaryDirectory() as td:
            p=Path(td);self.fixture(p)
            f=p/'pair-000-opponent.jsonl';rows=list(map(json.loads,f.read_text().splitlines()))
            f.write_text(''.join(json.dumps(e)+'\n' for e in rows[6:]))
            with self.assertRaises(AssertionError):review.cell(p,1,'clock',120000)
    def test_missing_ngn_internal_observer_rejected(self):
        with tempfile.TemporaryDirectory() as td:
            p=Path(td);self.fixture(p)
            f=p/'pair-000-ngn.jsonl';rows=list(map(json.loads,f.read_text().splitlines()))
            f.write_text(''.join(json.dumps(e)+'\n' for e in rows if not e['text'].startswith('info string ngn_clock ')))
            with self.assertRaises(AssertionError):review.cell(p,1,'clock',120000)

if __name__=='__main__':unittest.main()
