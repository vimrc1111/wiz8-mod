# monster-names.py <src.json> <monsters.json> <glossary.tsv>：怪物名機檢（記憶 monster-names-by-family）。
# src.json＝MONSTERS.DBS 4×記錄＋欄位的扁平原文；欄 0＝單數名（# 之後是編輯器註記）、欄 2／3＝種族單／複數。
# 1 tail：英文單數以種族英文結尾 → 中文（展開 {{詞}}、去 # 註記）以種族中文結尾。
# 2 same：同一英文單數名 → 同一中文；同一種族英文 → 同一中文。
import sys, json, re
src = json.load(open(sys.argv[1], encoding='utf-8'))
tr = {e['id']: e['zh'] for e in json.load(open(sys.argv[2], encoding='utf-8'))}
gl = {l.split('\t')[0]: l.split('\t')[1] for l in open(sys.argv[3], encoding='utf-8') if '\t' in l and not l.startswith('#')}
exp = lambda z: re.sub(r'\{\{(.+?)\}\}', lambda m: gl.get(m.group(1), m.group(0)), z).split('#')[0].rstrip(' ')
bad, seen = 0, {}
def fail(rule, i, msg):
    global bad; bad += 1; print(f'FAIL {rule} id={i} {msg}')
for r in range(len(src) // 4):
    en, fam = src[4*r].split('#')[0].rstrip(' '), src[4*r+2]
    zh, fzh = tr.get(4*r), tr.get(4*r+2)
    for k, i, z in ((('n', en), 4*r, zh), (('f', fam), 4*r+2, fzh)):
        if z is None: continue
        if seen.setdefault(k, (i, exp(z)))[1] != exp(z): fail('same', i, f'與 id={seen[k][0]} 同原文不同譯')
    if zh and fzh and (en == fam or en.endswith(' ' + fam)) and not exp(zh).endswith(exp(fzh)):
        fail('tail', 4*r, f'譯名不以種族譯名「{exp(fzh)}」結尾')
print(f'{"ok" if not bad else "FAIL"} monster-names：{len(src)//4} 筆，{bad} 條違規')
sys.exit(1 if bad else 0)
