#!/usr/bin/env python3
"""Draw training and held-out mean squared error as a portable PNG."""
import csv
import struct
import sys
import zlib

FONT = {
 '0':'111101101101111','1':'010110010010111','2':'111001111100111','3':'111001111001111','4':'101101111001001','5':'111100111001111','6':'111100111101111','7':'111001001001001','8':'111101111101111','9':'111101111001111',
 'G':'111100101101111','A':'010101111101101','C':'111100100100111','D':'110101101101110','E':'111100110100111','H':'101101111101101','I':'111010010010111','L':'100100100100111','M':'101111111101101','N':'101111111111101','O':'111101101101111','P':'110101110100100','R':'110101110101101','S':'111100111001111','T':'111010010010010','V':'101101101101010',
 '.':'000000000000010','-':'000000111000000',' ':'000000000000000',':':'000010000010000'
}
def plot(csv_path, png_path):
    with open(csv_path) as f: rows=list(csv.DictReader(f))
    curve=[(float(r['train']),float(r['validation'])) for r in rows]
    w,h=900,520; pixels=bytearray([255]*(w*h*3))
    def pixel(x,y,c):
        if 0<=x<w and 0<=y<h: pixels[(y*w+x)*3:(y*w+x)*3+3]=bytes(c)
    def line(x0,y0,x1,y1,c):
        n=max(abs(x1-x0),abs(y1-y0),1)
        for i in range(n+1): pixel(round(x0+(x1-x0)*i/n),round(y0+(y1-y0)*i/n),c)
    def text(x,y,s,color=(30,30,30),scale=2):
        for char in s:
            glyph=FONT[char]
            for i,on in enumerate(glyph):
                if on=='1':
                    for dx in range(scale):
                        for dy in range(scale): pixel(x+(i%3)*scale+dx,y+(i//3)*scale+dy,color)
            x+=4*scale
    text(80,15,'TRAINING AND VALIDATION LOSS',scale=3)
    line(90,70,90,445,(30,30,30)); line(90,445,850,445,(30,30,30))
    lo=0; hi=max(max(v) for v in curve)*1.05
    for i in range(5):
        y=445-round(i*375/4); value=hi*i/4
        line(90,y,850,y,(225,225,225)); text(8,y-5,f'{value:.3f}',scale=2)
    for j,col in [(0,(35,100,220)),(1,(220,70,50))]:
        pts=[(90+round(i*760/max(1,len(curve)-1)),445-round((v[j]-lo)*375/(hi-lo))) for i,v in enumerate(curve)]
        for p,q in zip(pts,pts[1:]): line(*p,*q,col)
        for x,y in pts:
            for dx in range(-2,3):
                for dy in range(-2,3): pixel(x+dx,y+dy,col)
    text(95,475,'EPOCH 1',scale=2); text(755,475,f'EPOCH {len(curve)}',scale=2)
    text(510,45,'TRAIN', (35,100,220)); text(645,45,'VALIDATION',(220,70,50)); text(8,50,'MSE')
    def chunk(kind,data): return struct.pack('>I',len(data))+kind+data+struct.pack('>I',zlib.crc32(kind+data)&0xffffffff)
    raw=b''.join(b'\0'+pixels[i*w*3:(i+1)*w*3] for i in range(h))
    with open(png_path,'wb') as f: f.write(b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR',struct.pack('>IIBBBBB',w,h,8,2,0,0,0))+chunk(b'IDAT',zlib.compress(raw))+chunk(b'IEND',b''))
if __name__ == '__main__': plot(sys.argv[1],sys.argv[2])
