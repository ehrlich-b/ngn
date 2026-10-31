import array,datetime,hashlib,json,struct,sys
from pathlib import Path
p=Path('/work/lanes/ngn-s39-key-integration/base.nnue');b=p.read_bytes()
assert hashlib.sha256(b).hexdigest()=='be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a'
magic,v,h,kb=struct.unpack_from('<4sIII',b)
assert (magic,v,h,kb)==(b'NGN3',3,512,8)
w=array.array('h');w.frombytes(b[96:96+2*kb*768*h]);bias=array.array('h');bias.frombytes(b[96+2*len(w):96+2*len(w)+2*h])
if sys.byteorder!='little':w.byteswap();bias.byteswap()
rows=[]
for k in range(kb):
 lows=[];highs=[]
 for lane in range(h):
  low=high=bias[lane]
  for square in range(64):
   choices=[w[(k*768+piece*64+square)*h+lane] for piece in range(12)]
   low+=min(0,*choices);high+=max(0,*choices)
  lows.append(low);highs.append(high)
 rows.append(dict(bucket=k,lower=min(lows),upper=max(highs),unsafe_lanes=sum(a< -32768 or c>32767 for a,c in zip(lows,highs))))
j=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),net_sha256=hashlib.sha256(b).hexdigest(),method='bias plus per-square min/max over empty and all12 pieces, summed over64 squares; conservative bound for every board occupancy and both perspectives',buckets=rows,provably_int16=all(r['unsafe_lanes']==0 for r in rows),max_abs_weight=max(abs(x) for x in w),meaning='numerical eligibility only; no speed or Elo verdict')
print(json.dumps(j,indent=2))
