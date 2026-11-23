# 近戰武器靜態表（notes/party.md「近戰武器 30 名」）：python3 -I scripts/weapon-static.py <wiz8-mod>
# 輸出：首行筆數，之後每行 平均 id 中名 英名 技能 單雙 骰 命中 攻速 模式數 特效 種族註 排除註，依平均傷害排序。
import json, struct, sys, os, csv
W = sys.argv[1]
d = open(os.path.join(W, 'build/data-x/DATABASES/ITEMS.DBS'), 'rb').read()
n = struct.unpack('<I', d[:4])[0]
recs = [d[4 + i * 0x10d:4 + (i + 1) * 0x10d] for i in range(n)]
gl = {}
for r in csv.reader(open(os.path.join(W, 'translations/glossary.tsv')), delimiter='\t'):
    if len(r) >= 2: gl[r[0]] = r[1]
import re
zh = {e['id']: re.sub(r'\{\{(.+?)\}\}', lambda m: gl.get(m.group(1), m.group(1)), e['zh']) for e in json.load(open(os.path.join(W, 'translations/items.json')))}
en = lambda r: r[:0x40].decode('utf-16le').split('\0')[0]
SK = ['劍', '斧', '長柄', '錘／連枷', '匕首', '杖']
EF = {0: '睡', 1: '麻', 2: '毒', 3: 'Hex', 4: '病', 5: '殺', 6: 'KO', 7: 'k7', 8: 'k8', 12: 'k12', 14: 'k14', 15: 'k15'}
EXCL = {85, 93, 96, 465, 489, 748, 749}
rows = []
for i, r in enumerate(recs):
    z, cnt, sides = struct.unpack('<hBB', r[0x4a:0x4e]); mask = struct.unpack('<H', r[0x4e:0x50])[0]
    if not (r[0x46] <= 5 and r[0x47] <= 1 and cnt * sides > 0 and mask): continue
    avg = z + cnt * (sides + 1) / 2
    eff = ' '.join(f'{EF.get(k, k)}{v}' for k, v in enumerate(r[0x50:0x60]) if v)
    hum = struct.unpack('<I', r[0x78:0x7c])[0] & 1
    rows.append((avg, i, zh.get(i, ''), en(r), SK[r[0x46]], '雙' if r[0x41] & 4 else '單', f'{z:+d}+{cnt}d{sides}' if z else f'{cnt}d{sides}',
                 struct.unpack('b', r[0x49:0x4a])[0], struct.unpack('b', r[0x48:0x49])[0], bin(mask).count('1'), eff, '' if hum else '非人類', '*' if i in EXCL else ''))
rows.sort(key=lambda t: (-t[0], t[1]))
print(len(rows))
for t in rows: print('\t'.join(map(str, t)))
