"""Compare ordinary/consuming gather byte for byte on an owned fixture."""
import hashlib,json,shutil,struct,subprocess,sys
from pathlib import Path
r=Path(sys.argv[1]);tool=Path(sys.argv[2]);r.mkdir()
knots=[[-1000.,-501.],[-100.,-50.5],[0.,0.],[100.,50.5],[1000.,501.]]
(r/'map.json').write_text(json.dumps({'knots':knots}))
# Packing fields are unchanged by gather. Cover both sentinels, signed ties,
# tails, multiple input scales, every result, and empty buckets.
for group in ('a','b'):
    d=r/group;d.mkdir();(d/'STATE').write_text('SCATTERED\n')
    counts=[]
    for bucket in range(3):
        records=[]
        for i,score in enumerate([] if bucket==2 else [-32768,-32002,-1001,-100,-1,0,1,100,1001,32002,32767]):
            rec=bytearray(32);struct.pack_into('<Q',rec,0,(1<<4)|(1<<60))
            rec[8]=0xd5;struct.pack_into('<h',rec,24,score);rec[26]=i%3;rec[27]=4;rec[28]=4
            records.append(rec)
        (d/f'bucket-{bucket:04}.bin').write_bytes(b''.join(records));counts.append(len(records))
    (d/'scatter.json').write_text(json.dumps({'schema':'ngn-k4-archive-scatter-v1','state':'COMPLETE','bucket_records':counts}))
for group in ('a','b'):shutil.copytree(r/group,r/(group+'-consume'))
for mode in ('normal','consume'):
    names=('a','b') if mode=='normal' else ('a-consume','b-consume')
    argv=[str(tool),'gather','--scatter',f'{r/names[0]}@1,{r/names[1]}@0.5','--map',str(r/'map.json'),'--output',str(r/f'{mode}.bf'),'--manifest',str(r/f'{mode}.json'),'--seed','2026100901']
    if mode=='consume':argv+=['--consume-buckets','true']
    subprocess.run(argv,check=True)
a,b=[(r/f'{x}.bf').read_bytes() for x in ('normal','consume')]
assert a==b and len(a)==28*32
for group in ('a','b'):
    assert len(list((r/group).glob('bucket-*.bin')))==3
    assert not list((r/(group+'-consume')).glob('bucket-*.bin'))
x,y=[json.loads((r/f'{m}.json').read_text()) for m in ('normal','consume')]
assert x['records']==y['records']==28 and x['dropped_sentinel']==y['dropped_sentinel']==16
print(json.dumps({'passed':True,'bytes_identical':True,'records':28,'dropped_sentinel':16,'sha256':hashlib.sha256(a).hexdigest()}))
