#!/usr/bin/env python3
"""Plot ablation validation curves without external image dependencies."""
import csv
import struct
import sys
import zlib
from pathlib import Path
from plot_loss import FONT
FONT['Z']='111001010100111'

COLORS=[(35,100,220),(220,70,50),(20,145,80),(145,75,180)]
def render(paths,out):
    curves=[]
    for path in paths:
        with open(path) as f: curves.append([float(r['validation']) for r in csv.DictReader(f)])
    w,h=900,520; pixels=bytearray([255]*(w*h*3))
    def pixel(x,y,c):
        if 0<=x<w and 0<=y<h: pixels[(y*w+x)*3:(y*w+x)*3+3]=bytes(c)
    def line(x0,y0,x1,y1,c):
        n=max(abs(x1-x0),abs(y1-y0),1)
        for i in range(n+1): pixel(round(x0+(x1-x0)*i/n),round(y0+(y1-y0)*i/n),c)
    def text(x,y,s,color=(30,30,30),scale=2):
        for char in s:
            for i,on in enumerate(FONT[char]):
                if on=='1':
                    for dx in range(scale):
                        for dy in range(scale): pixel(x+(i%3)*scale+dx,y+(i//3)*scale+dy,color)
            x+=4*scale
    lo=min(min(c) for c in curves); hi=max(max(c) for c in curves); pad=max((hi-lo)*.08,.0001); lo-=pad; hi+=pad
    text(100,15,'VALIDATION MSE',scale=3); line(90,70,90,445,(30,30,30)); line(90,445,850,445,(30,30,30))
    for i in range(5):
        y=445-round(i*375/4); line(90,y,850,y,(220,220,220)); text(8,y-5,f'{lo+(hi-lo)*i/4:.3f}')
    labels={'lr002':'LR 0.002','lr0005':'LR 0.0005','hidden256':'SIZE 256','search':'SEARCH'}
    for j,(path,curve) in enumerate(zip(paths,curves)):
        color=COLORS[j%len(COLORS)]; text(105+(j%2)*350,53+(j//2)*16,labels[Path(path).stem],color)
        points=[(90+round(i*760/max(1,len(curve)-1)),445-round((v-lo)*375/(hi-lo))) for i,v in enumerate(curve)]
        for a,b in zip(points,points[1:]): line(*a,*b,color)
        for x,y in points:
            for dx in range(-2,3):
                for dy in range(-2,3): pixel(x+dx,y+dy,color)
    text(95,475,'EPOCH 1'); text(755,475,f'EPOCH {max(map(len,curves))}')
    def chunk(kind,data): return struct.pack('>I',len(data))+kind+data+struct.pack('>I',zlib.crc32(kind+data)&0xffffffff)
    raw=b''.join(b'\0'+pixels[i*w*3:(i+1)*w*3] for i in range(h))
    with open(out,'xb') as f: f.write(b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR',struct.pack('>IIBBBBB',w,h,8,2,0,0,0))+chunk(b'IDAT',zlib.compress(raw))+chunk(b'IEND',b''))
if __name__=='__main__': render(sys.argv[2:],sys.argv[1])
