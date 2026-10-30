#!/usr/bin/env python3
"""Render paired Elo results and 95% confidence intervals to a PNG."""
import struct
import zlib
from pathlib import Path
from plot_loss import FONT
from check_stage4 import match

FONT.update({'B':'110101110101110','F':'111100110100100','U':'101101101101111','W':'101101111111101','X':'101101010101101','Y':'101101010010010','+':'000010111010000','[':'110100100100110',']':'011001001001011',',':'000000000010100'})

def render(out):
    rows = [('STAGE 2 V HCE','stage2-ngn'),('STAGE 3 V HCE','stage3-ngn'),('STAGE 3 V STAGE 2','stage3-stage2'),('STAGE 4 V HCE','stage4-ngn'),('STAGE 4 V STAGE 3','stage4-stage3')]
    results = [(label,match(key)) for label,key in rows]
    w,h=1200,440; pixels=bytearray([255]*(w*h*3))
    def pixel(x,y,c):
        if 0<=x<w and 0<=y<h: pixels[(y*w+x)*3:(y*w+x)*3+3]=bytes(c)
    def line(x0,y0,x1,y1,c):
        n=max(abs(x1-x0),abs(y1-y0),1)
        for i in range(n+1): pixel(round(x0+(x1-x0)*i/n),round(y0+(y1-y0)*i/n),c)
    def text(x,y,s,color=(30,30,30),scale=3):
        for char in s:
            for i,on in enumerate(FONT[char]):
                if on=='1':
                    for dx in range(scale):
                        for dy in range(scale): pixel(x+(i%3)*scale+dx,y+(i//3)*scale+dy,color)
            x+=4*scale
    low=min(-200,*(r[1] for _,r in results)); high=max(100,*(r[2] for _,r in results)); low-=20; high+=20
    def x(v): return 360+round((v-low)/(high-low)*510)
    text(40,25,'PAIRED ELO AT 1000 NODES',scale=4)
    text(40,65,'500 OPENINGS  95 PERCENT INTERVAL',scale=2)
    line(x(0),105,x(0),365,(190,190,190))
    for i,(label,(value,lo,hi)) in enumerate(results):
        y=125+i*52; color=(35,100,220) if i<3 else (20,145,80)
        text(40,y-7,label,scale=3)
        line(x(lo),y,x(hi),y,color); line(x(lo),y-7,x(lo),y+7,color); line(x(hi),y-7,x(hi),y+7,color)
        for dx in range(-4,5):
            for dy in range(-4,5): pixel(x(value)+dx,y+dy,color)
        text(900,y-7,f'{value:+.1f} [{int(lo):+d},{int(hi):+d}]',scale=2)
    text(x(0)-4,380,'0',scale=2)
    def chunk(kind,data): return struct.pack('>I',len(data))+kind+data+struct.pack('>I',zlib.crc32(kind+data)&0xffffffff)
    raw=b''.join(b'\0'+pixels[i*w*3:(i+1)*w*3] for i in range(h))
    Path(out).write_bytes(b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR',struct.pack('>IIBBBBB',w,h,8,2,0,0,0))+chunk(b'IDAT',zlib.compress(raw))+chunk(b'IEND',b''))

if __name__ == '__main__': render('../evidence/comparison.png')
