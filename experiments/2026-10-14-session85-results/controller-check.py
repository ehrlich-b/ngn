import ast,json,runpy,tempfile,unittest
from pathlib import Path
from unittest.mock import patch

r=Path('/home/ehrli/ngn-data/night-20261007')
module=runpy.run_path(str(r/'session84-training174/train174.py'))
main=module['main']; g=main.__globals__
class Child:
    pid=123
    def wait(self,**kwargs):return 1

with tempfile.TemporaryDirectory() as name:
    d=Path(name); (d/'trained').mkdir()
    original={'argv':['taskset','-c','12,14','python','-m','trainer.train','--epochs','174']}
    (d/'manifest.json').write_text(json.dumps(original))
    (d/'trained/history.json').write_text(json.dumps([{'epoch':92}]))
    (d/'trained/latest.pt').write_bytes(b'synthetic-checkpoint')
    g['D']=d
    with patch.object(g['sys'],'argv',['train174.py','--resume']), patch.object(g['os'],'statvfs',return_value=type('Disk',(),dict(f_bavail=30*1024**3,f_frsize=1))()), patch.dict(g,{'status':lambda *a:None}), patch.object(g['subprocess'],'Popen',return_value=Child()) as popen:
        try:main()
        except AssertionError as e:assert str(e)=='trainer exited1'
        else:raise AssertionError('child failure not propagated')
    assert json.loads((d/'manifest.json').read_text())==original
    receipt=json.loads((d/'resume-manifest.json').read_text())
    assert receipt['completed_epoch']==92
    assert popen.call_args.args[0]==[*original['argv'],'--resume',str(d/'trained/latest.pt')]

# Evaluate the actual final-history assertion against both epoch sequences.
tree=ast.parse((r/'session84-training174/train174.py').read_text())
condition=next(n.test for n in ast.walk(tree) if isinstance(n,ast.Assert) and isinstance(n.test,ast.Compare) and isinstance(n.test.left,ast.ListComp))
expression=compile(ast.Expression(condition),'<actual-final-history-check>','eval')
assert eval(expression,{'history':[{'epoch':i} for i in range(1,175)]})
assert not eval(expression,{'history':[{'epoch':i} for i in range(1,116)]})
print('PASS resume argv, immutable manifest, checkpoint receipt, child failure, final174 acceptance/final115 rejection')
