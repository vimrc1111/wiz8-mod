"""I9 six-member party optimizer (goals/stage16.md I9).

  python3 -I scripts/i9-party.py templates <DATABASES> <out>   per-class rank templates (L28, kill metric, 56 L20+ monsters)
  python3 -I scripts/i9-party.py expert <in> <out>             I9b: Expert 8-monster copies of the <in> templates
  python3 -I scripts/i9-party.py party <out>                   best party per class combination, markdown to stdout
  python3 -I scripts/i9-party.py screen <DATABASES> <out>      I9c: Expert single-hand screening templates
  python3 -I scripts/i9-party.py pool <DATABASES> <out> <c,..> <top> <top_off>   I9c: Expert party templates from screen TSVs
  python3 -I scripts/i9-party.py compare <old> <new>           I9c: shared cells identical, pool and party diffs

Pools come from the I4 screening TSVs (build/i4/screen-*.tsv). Writes item ids only.
"""
import importlib.util
import glob
import itertools
import json
import os
import struct
import sys

spec = importlib.util.spec_from_file_location('i4', os.path.join(os.path.dirname(__file__), 'i4-rank.py'))
i4 = importlib.util.module_from_spec(spec)
spec.loader.exec_module(i4)

MULTI = {3}      # 錐刺短劍: obtainable in multiples
EXCLUDED = {85, 93, 96, 465, 489, 748, 749}  # MG 2026-10-13 never saw these in play (no loot-table source either). The Avenger (Marten Chest) is in since I9c.
TOP, TOP_OFF = 6, 8  # ponytail: pool cut at screening rank; deepen if a best party sits on a pool edge
CORE_A, CORE_B, FLEX = (0, 7, 5), (9, 8, 0, 7, 5), (6, 2, 1, 4, 3)  # MG 2026-10-12
NAME = {0: '戰', 1: '領', 2: '女', 3: '遊', 4: '侍', 5: '忍', 6: '僧', 7: '賊', 8: '工', 9: '詩'}


def monsters(path):
    d = open(path, 'rb').read()
    n = struct.unpack('<I', d[:4])[0]
    out = [i for i in range(n) if d[4 + i * 0x297 + 0x24f] >= 20]
    assert len(out) == 56, len(out)  # 2026-10-08 kill-rate set (notes/research/ranking-history.md:184)
    return out


def weapons(dbs):
    recs = i4.items(os.path.join(dbs, 'ITEMS.DBS'))
    m = [i for i in i4.melee(recs) if i not in EXCLUDED]
    return m, [i for i in m if recs[i][0x41] & 8 and not recs[i][0x41] & 4]


def loadouts(m, offs, best, c, top=TOP, top_off=TOP_OFF, extra=frozenset()):
    """Pool for class c from screening values best[c][item]: top mains/offhands plus MG gear and Stiletto."""
    rank = lambda pool, k: [i for i in sorted((i for i in pool if i in best[c]), key=lambda i: (-best[c][i], i))[:k]]
    # MG gear (I4) joins the pools: screening is damage only, instakill-heavy picks (村正) can miss it.
    legal = set(best[c]) - EXCLUDED
    mg_main = {v[0] for v in i4.MG.values()} | {a for a, _ in i4.PARTY['bard'][5]}
    mains = sorted(set(rank(m, top)) | ({3} | mg_main) & legal | extra & set(m))
    pool = sorted(set(rank(offs, top_off)) | ({3} | {i for v in i4.MG.values() for i in v}) & legal & set(offs) | extra & set(offs))
    lo = [{"Name": "unarmed"}] + [{"Name": f"i{a}", "Main": i4.w(a)} for a in mains]
    lo += [{"Name": f"i{a}+i{b}", "Main": i4.w(a), "Off": i4.w(b)} for a in mains for b in pool if a != b or a in MULTI]
    return lo, rank(m, top), rank(offs, top_off)


def templates(dbs, out):
    m, offs = weapons(dbs)
    best = {}
    for mon in i4.MONSTERS:
        for f in i4.rows(f'build/i4/screen-{mon}.tsv'):
            v = i4.num(f[6])
            if v is not None:
                b = best.setdefault(int(f[1]), {})
                b[int(f[2][1:])] = max(b.get(int(f[2][1:]), 0), v)
    mons = monsters(os.path.join(dbs, 'MONSTERS.DBS'))
    for c in i4.CLASSES:
        t = i4.tmpl([c], loadouts(m, offs, best, c)[0], 0, Levels=[28], Monsters=mons, Metric="kill")
        i4.dump(out, f'party-c{c}.json', t)


# I9b (goals/stage16.md): baseline-align part 7 settings, Expert, MG's 8 monsters.
EXPERT = {"Attrs": [0, 4, 5, 3], "Levels": [28], "SkillPrio": [], "AttrMin": 0, "Haste": False, "Commands": ["attack"],
          "Train": {**{str(s): 100 for s in (0, 1, 2, 3, 4, 5, 14, 16, 18, 19)}, "34": 58},
          "Monsters": [564, 114, 184, 171, 129, 158, 120, 192], "Metric": "kill", "Difficulty": 2}


def expert(src, out):
    for c in i4.CLASSES:
        t = json.load(open(os.path.join(src, f'party-c{c}.json')))
        i4.dump(out, f'party-c{c}.json', expert_tmpl(c, t["Races"], t["Loadouts"]))


def expert_tmpl(c, races, lo):
    e = {**EXPERT, "Races": races, "Classes": [c], "Loadouts": lo}
    if c == 0:
        e["Commands"] = ["attack", "berserk"]  # Fighter berserk (baseline-align:12)
    return e


# I9c (goals/stage16.md): screen single-hand mains for the Expert pool, human, prio 0,4,5,3 only.
SCREEN_PRIO = '0,4,5,3'


def screen(dbs, out):
    m, _ = weapons(dbs)
    for c in i4.CLASSES:
        i4.dump(out, f'screen-c{c}.json', expert_tmpl(c, [0], [{"Name": f"i{i}", "Main": i4.w(i)} for i in m]))


def screened(out):
    """best[c][item] from screen-c*.tsv: max over commands at SCREEN_PRIO (NA rows = not legal)."""
    best = {}
    for c in i4.CLASSES:
        b = best.setdefault(c, {})
        # screen-c{c}-<tag>.tsv: supplements for items added after the main screen (I9c: The Avenger)
        for f in (f for t in sorted(glob.glob(os.path.join(out, f'screen-c{c}[.-]*tsv'))) for f in i4.rows(t)):
            v = i4.num(f[6])
            if v is not None and f[4] == SCREEN_PRIO:
                b[int(f[2][1:])] = max(b.get(int(f[2][1:]), 0), v)
    return best


def pool(dbs, out, classes, top, top_off):
    m, offs = weapons(dbs)
    best = screened(out)
    recs = i4.items(os.path.join(dbs, 'ITEMS.DBS'))
    # Screening is human-only, so race-restricted weapons (CoC, Giant's Sword, ...) never rank: always pool them (MG 2026-10-13).
    no_human = {i for i in m if not struct.unpack_from('<I', recs[i], 0x78)[0] & 1}
    for c in classes:
        extra = {i for i in no_human if struct.unpack_from('<H', recs[i], 0x76)[0] >> c & 1}
        lo, mt, ot = loadouts(m, offs, best, c, top, top_off, extra)
        print(f'c{c} main {mt} off {ot}')
        i4.dump(out, f'party-c{c}.json', expert_tmpl(c, i4.RACES, lo))


def items_of(out, c):
    t = json.load(open(os.path.join(out, f'party-c{c}.json')))
    mains = {l["Main"]["Item"] for l in t["Loadouts"] if "Main" in l}
    return mains, {l["Off"]["Item"] for l in t["Loadouts"] if "Off" in l}


def cells(out, c):
    return {tuple(f[:6]): f[6] for f in i4.rows(os.path.join(out, f'party-c{c}.tsv'))}


def best_parties(out):
    opts = {c: options(out, c) for c in i4.CLASSES}
    res = {}
    for core, k in ((CORE_A, 3), (CORE_B, 1)):
        for flex in itertools.combinations(FLEX, k):
            cls = list(core + flex)
            res[tuple(cls)] = solve(cls, opts)
    return res


def compare(old, new):
    """I9c (4)(5): shared cells must be byte-identical; pool and party differences."""
    n = bad = 0
    for c in i4.CLASSES:
        a, b = cells(old, c), cells(new, c)
        for k in a.keys() & b.keys():
            n += 1
            if a[k] != b[k]:
                bad += 1
                print('DIFF', k, a[k], b[k])
        (om, oo), (nm, no) = items_of(old, c), items_of(new, c)
        print(f'pool {NAME[c]} main +{sorted(nm - om)} -{sorted(om - nm)} off +{sorted(no - oo)} -{sorted(oo - no)}')
    print(f'cells compared {n} differ {bad}')
    po, pn = best_parties(old), best_parties(new)
    for cls in po:
        (vo, ko), (vn, kn) = po[cls], pn[cls]
        ch = [f'{NAME[c]} {x[2]} {x[0]:.3f}→{y[2]} {y[0]:.3f}' for c, x, y in zip(cls, ko, kn) if x[2] != y[2]]
        print(f'party {"".join(NAME[c] for c in cls)} {vo:.3f}→{vn:.3f} ({vn - vo:+.3f}) ' + ('；'.join(ch) or '不變'))
    assert bad == 0, bad


def options(out, c, race=None):
    """Best kill rate per item set for class c (race given: that race only): [(value, items, label)], best first."""
    best = {}
    for f in i4.rows(os.path.join(out, f'party-c{c}.tsv')):
        v = i4.num(f[6])
        if v is None or race is not None and f[0] != str(race):
            continue
        ids = [] if f[2] == 'unarmed' else [int(x[1:]) for x in f[2].split('+')]
        key = tuple(sorted(i for i in ids if i not in MULTI))
        if key not in best or v > best[key][0]:
            best[key] = (v, set(key), f'{f[2]} {f[3]} r{f[0]}')
    return sorted(best.values(), key=lambda o: -o[0])


def solve(classes, opts):
    """Max sum of member values with no item shared (branch and bound)."""
    top = [opts[c][0][0] if opts[c] else 0 for c in classes]
    bound = [sum(top[k:]) for k in range(len(classes) + 1)]
    best = [-1, None]

    def dfs(k, used, acc, pick):
        if acc + bound[k] <= best[0]:
            return
        if k == len(classes):
            best[:] = [acc, list(pick)]
            return
        for o in opts[classes[k]]:
            if not (o[1] & used):
                dfs(k + 1, used | o[1], acc + o[0], pick + [o])
    dfs(0, set(), 0.0, [])
    return best


def check(out, c):
    """I9b (2): every race x loadout of the template is a row group or a counted skip (rank.go skip accounting)."""
    t = json.load(open(os.path.join(out, f'party-c{c}.json')))
    perms = len(list(itertools.permutations(t["Attrs"])))
    groups, skips, na_unarmed, pareto = {}, {}, set(), False
    for line in open(os.path.join(out, f'party-c{c}.tsv')):
        f = line.rstrip('\n').split('\t')
        if line.startswith('# skipped'):
            _, _, reason, n = line.split()
            skips[reason] = int(n)
        pareto |= line.startswith('# pareto')  # the pareto block repeats rows above
        if pareto or line.startswith(('#', 'race')) or len(f) < 7:
            continue
        k = (f[0], f[2], f[3])
        groups[k] = groups.get(k, 0) + 1
        if f[2] == 'unarmed' and f[6] == 'NA':
            na_unarmed.add((f[0], f[2]))  # counted as a skip and still printed
    pairs = {(r, l) for r, l, _ in groups}
    want = len(t["Races"]) * len(t["Loadouts"])
    got = len(pairs) + sum(v for k, v in skips.items() if k != 'berserk') - len(na_unarmed)
    assert got == want, (c, got, want, skips)
    assert all(n == perms for n in groups.values()), (c, set(groups.values()))
    assert len(groups) == len(pairs) * len(t["Commands"]) - skips.get('berserk', 0), (c, len(groups), skips)


def party(out):
    for c in i4.CLASSES:
        check(out, c)
    res = [(v, list(cls), pick) for cls, (v, pick) in best_parties(out).items()]
    print('| 隊 | 合計 | 成員（職業 值 組 指令 種族） |\n|---|---|---|')
    for v, cls, pick in sorted(res, key=lambda r: (len(r[1]) != 6 or r[1][:3] != list(CORE_A), -r[0])):
        tag = ''.join(NAME[c] for c in cls)
        print(f'| {tag} | {v:.3f} | ' + '；'.join(f'{NAME[c]} {o[0]:.3f} {o[2]}' for c, o in zip(cls, pick)) + ' |')


# N3 (GOAL-wiz8calc.md): I9c frozen pool (build/i9e/party-c*.json loadouts), SEN attrs as the
# party-constraints basis (notes/research/ranking-history.md:921 感知段), metric kill / kill-hi.
SEN = [0, 4, 5, 6]
# notes/party.md:206 屬性判定: heal = 領女遊忍僧, ranged = 遊工忍, Infinity Helm + Robes = 戰領女遊侍
HEAL, RANGED, GEAR = {1, 2, 3, 5, 6}, {3, 5, 8}, {0, 1, 2, 3, 4}


def n3tmpl(src, out, metric):
    for c in i4.CLASSES:
        t = json.load(open(os.path.join(src, f'party-c{c}.json')))
        i4.dump(out, f'party-c{c}.json', {**t, "Attrs": SEN, "Metric": metric})


def ok_mg(cls):
    s = set(cls)
    return len(s & HEAL) >= 4 and len(s & RANGED) >= 2 and len(s & GEAR) >= 3


def rank6(out):
    """All C(10,6) parties: best unique-item assignment, MG-constraint flag; TSV, best first."""
    for c in i4.CLASSES:
        check(out, c)
    opts = {c: options(out, c) for c in i4.CLASSES}
    rows = []
    for cls in itertools.combinations(i4.CLASSES, 6):
        v, pick = solve(list(cls), opts)
        rows.append((round(v, 6), ''.join(NAME[c] for c in cls), int(ok_mg(cls)), int(len({0, 7} & set(cls)) <= 1),
                     '；'.join(f'{NAME[c]} {o[0]:.6f} {o[2]}' for c, o in zip(cls, pick))))
    print('party\tsum\tmg\tfr_le1\tpicks')
    for v, tag, mg, fr, picks in sorted(rows, key=lambda r: (-r[0], r[1])):
        print(f'{tag}\t{v:.6f}\t{mg}\t{fr}\t{picks}')


# N4: B 隊（notes/party.md:206 MG 裁定）定稿種族 戰哈比 賊貓人 忍妖精 僧人類 領狼人 遊貓人
B_FIXED = {0: 4, 7: 8, 5: 5, 6: 0, 1: 9, 3: 8}


def fixed(out):
    """B party at its fixed races: best unique-item assignment, one row per member."""
    cls = list(B_FIXED)
    v, pick = solve(cls, {c: options(out, c, r) for c, r in B_FIXED.items()})
    print('class\tvalue\tpick')
    for c, o in zip(cls, pick):
        print(f'{NAME[c]}\t{o[0]:.6f}\t{o[2]}')
    print(f'合計\t{v:.6f}\t')


def selftest():
    import random
    rnd = random.Random(1)
    for _ in range(200):
        cls = list(range(4))
        opts = {c: sorted([(rnd.random(), set(rnd.sample(range(6), rnd.randint(0, 2))), '') for _ in range(4)],
                          key=lambda o: -o[0]) for c in cls}
        brute = max((sum(o[0] for o in p) for p in itertools.product(*(opts[c] for c in cls))
                     if sum(len(o[1]) for o in p) == len(set().union(*(o[1] for o in p)))), default=-1)
        assert abs(solve(cls, opts)[0] - brute) < 1e-12
    # MG constraint flags (party-constraints.md 結果表): B 戰賊忍僧領遊 passes, MG's own 侍領僧遊詩工 has heal 3.
    code = {v: k for k, v in NAME.items()}
    assert ok_mg([code[x] for x in '戰賊忍僧領遊']) and ok_mg([code[x] for x in '忍僧領侍女遊'])
    assert not ok_mg([code[x] for x in '侍領僧遊詩工']) and not ok_mg([code[x] for x in '戰賊忍僧領侍'])
    print('selftest PASS')


if __name__ == '__main__':
    {'selftest': selftest, 'templates': lambda: templates(sys.argv[2], sys.argv[3]), 'expert': lambda: expert(sys.argv[2], sys.argv[3]), 'party': lambda: party(sys.argv[2]),
     'screen': lambda: screen(sys.argv[2], sys.argv[3]),
     'pool': lambda: pool(sys.argv[2], sys.argv[3], [int(c) for c in sys.argv[4].split(',')], int(sys.argv[5]), int(sys.argv[6])),
     'compare': lambda: compare(sys.argv[2], sys.argv[3]),
     'n3tmpl': lambda: n3tmpl(sys.argv[2], sys.argv[3], sys.argv[4]), 'rank6': lambda: rank6(sys.argv[2]), 'fixed': lambda: fixed(sys.argv[2])}[sys.argv[1]]()
