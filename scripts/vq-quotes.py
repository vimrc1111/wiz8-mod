#!/usr/bin/env python3
# vq.py <md> <pagesdir>: 對 md 每個 "## " 來源的 URL 抓頁（快取到 pagesdir），檢查每條 "…" 引文是否為頁面文字的子字串（正規化空白、引號、大小寫）
import sys, re, html, hashlib, os, subprocess
from html.parser import HTMLParser
class T(HTMLParser):
    def __init__(s): super().__init__(); s.o=[]; s.skip=0
    def handle_starttag(s,t,a): s.skip+= t in('script','style')
    def handle_endtag(s,t): s.skip-= t in('script','style') and s.skip>0
    def handle_data(s,d):
        if not s.skip: s.o.append(d)
def norm(x):
    x=html.unescape(x).replace('’',"'").replace('‘',"'").replace('“','"').replace('”','"').replace('…','...')
    return re.sub(r'\s+',' ',x).strip().lower()
md, pd = sys.argv[1], sys.argv[2]; os.makedirs(pd, exist_ok=True)
url=None; stats={}
for line in open(md, encoding='utf-8'):
    if line.startswith('URL:'):
        url=line.split(':',1)[1].strip(); f=os.path.join(pd, hashlib.sha1(url.encode()).hexdigest()+'.html')
        if not os.path.exists(f):
            subprocess.run(['curl','-sL','--max-time','30','-A','wiz8-mod personal research (non-commercial)','-o',f,url])
        try: p=T(); p.feed(open(f,encoding='utf-8',errors='replace').read()); page=norm(' '.join(p.o))
        except Exception: page=''
        stats[url]=[0,0,len(page)]
        continue
    if url and line.lstrip().startswith(('- [', '>')):
        for q in [x for x in line.split('"')[1:-1] if len(x.strip())>=12 and not x.rstrip().endswith(':')]:
            ok = norm(q).rstrip('.') in page
            stats[url][0]+=ok; stats[url][1]+=1
            if not ok: print('MISS', url, '|', q[:90])
for u,(a,b,n) in stats.items(): print(f'{a}/{b} pagelen={n} {u}')
