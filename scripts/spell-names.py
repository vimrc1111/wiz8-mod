# spell-names.py <spells-src.json> <spells.json> <glossary.tsv> <exempt.tsv> <label> <src.json> <tr.json>：法術名機檢。
# 原文以獨立詞（區分大小寫）提到法術英文名 → 譯文（展開 {{詞}}）要含該法術在 spells.json 的譯名。
# 不是指法術的同形詞（Slow down、屬性 Stamina）登記在 exempt.tsv：label<TAB>id<TAB>英文<TAB>理由；登記了卻沒命中＝過期，也紅。
import sys, json, re
ssrc = json.load(open(sys.argv[1], encoding='utf-8'))
gl = {l.split('\t')[0]: l.split('\t')[1] for l in open(sys.argv[3], encoding='utf-8') if '\t' in l and not l.startswith('#')}
exp = lambda z: re.sub(r'\{\{(.+?)\}\}', lambda m: gl.get(m.group(1), m.group(0)), z)
sp = {ssrc[e['id']]: exp(e['zh']) for e in json.load(open(sys.argv[2], encoding='utf-8'))}
label = sys.argv[5]
ex = {(int(c[1]), c[2]) for c in (l.rstrip('\n').split('\t') for l in open(sys.argv[4], encoding='utf-8') if not l.startswith('#') and l.strip()) if c[0] == label}
src = json.load(open(sys.argv[6], encoding='utf-8'))
tr = {e['id']: exp(e['zh']) for e in json.load(open(sys.argv[7], encoding='utf-8'))}
bad, used = 0, set()
for i, zh in sorted(tr.items()):
    for en, szh in sp.items():
        if not re.search(r'(?<![A-Za-z])' + re.escape(en) + r'(?![A-Za-z])', src[i]) or szh in zh: continue
        if (i, en) in ex: used.add((i, en)); continue
        bad += 1; print(f'FAIL spell id={i} 原文提到 {en}，譯文沒有「{szh}」')
for i, en in sorted(ex - used):
    bad += 1; print(f'FAIL spell-exempt id={i} {label} 的 {en} 登記了但沒命中（過期）')
print(f'{"FAIL" if bad else "ok"} spell-names {label}：{len(tr)} 筆，{bad} 條違規，例外 {len(used)}')
sys.exit(1 if bad else 0)
