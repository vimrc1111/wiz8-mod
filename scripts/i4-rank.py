"""I4 rank v1 templates and summary (goals/stage16.md I4).

  python3 -I scripts/i4-rank.py screen <ITEMS.DBS> <out>    screening templates
  python3 -I scripts/i4-rank.py main <ITEMS.DBS> <out>      main/reference templates (needs screen TSVs)
  python3 -I scripts/i4-rank.py summary <out>               markdown tables to stdout

Driven by scripts/i4-rank.sh. Writes item ids only; names stay in the local DBS.
"""
import collections
import json
import os
import struct
import sys

R = 269
MONSTERS = (564, 411)
CLASSES = range(10)  # Fighter..Bard (casters have no melee build here)
RACES = list(range(11))
TRAIN_ALL = {str(s): 100 for s in range(41)}
# MG four (goals/stage16.md) and MG party (notes/calc-qa/mg-party-l25.md), by class.
MG = {5: [29, 70], 7: [62, 5], 4: [41, 51], 0: [501, 25, 70], 1: [772, 25], 6: [81], 3: [778], 9: [59, 780, 511], 8: [62, 3]}
MG_PAIRS = {5: (29, 70), 7: (62, 5), 4: (41, 51), 0: (501, 70)}  # Fighter: MG 2026-10-12 Ivory Blade + Rod of Sprites
TOP = 4
# Offhand pool: top TOP_OFF legal offhands per class by single-hand screening value (attack, speed,
# effects all priced in). A full 30-offhand run found every best pair's offhand at rank <= 7.
TOP_OFF = 8
# name: race, class, level, attrs (STR INT PIE VIT DEX SPD SEN as shown), raw skills, (main, off) pairs.
# Skills: weapon 0 sword/3 mace/4 dagger/5 staff/7 modern, 16 close, 18 DW, 19 crit, 34 Power Strike.
PARTY = {
    'samurai': (0, 4, 25, [100, 60, 45, 62, 100, 97, 65], {0: 97, 4: 5, 16: 91, 18: 92, 19: 96, 34: 33}, [(41, 51)]),
    'lord': (9, 1, 25, [100, 35, 60, 65, 105, 91, 113], {3: 111, 0: 11, 4: 13, 7: 77, 16: 87, 18: 92, 34: 49}, [(772, 25)]),
    'monk': (0, 6, 25, [91, 65, 50, 50, 100, 100, 78], {5: 48, 16: 93, 18: 0, 19: 97}, [(81, None)]),
    'ranger': (10, 3, 25, [84, 55, 25, 62, 100, 93, 100], {0: 76, 7: 10, 16: 74, 18: 5}, [(778, None)]),
    'bard': (1, 9, 27, [113, 55, 50, 50, 105, 98, 110], {0: 78, 4: 75, 16: 90, 18: 76, 19: 20}, [(59, 511), (780, 511)]),
    'gadgeteer': (1, 8, 27, [85, 60, 50, 50, 100, 110, 110], {7: 97, 0: 58, 4: 47, 16: 75, 18: 43}, [(62, 3)]),
}


def items(path):
    d = open(path, 'rb').read()
    n = struct.unpack('<I', d[:4])[0]
    return [d[4 + i * R:4 + (i + 1) * R] for i in range(n)]


def melee(recs):
    out = []
    for i, r in enumerate(recs):
        _, n, s = struct.unpack('<hBB', r[0x4a:0x4e])
        mask = struct.unpack('<H', r[0x4e:0x50])[0]
        # Weapon skills 0..5, range enum 0/1 (melee, extended), real dice, some attack mode.
        if r[0x46] <= 5 and r[0x47] <= 1 and n * s > 0 and mask:
            out.append(i)
    return out


def w(i):
    return {"Item": i, "Skill": 0}


def tmpl(classes, loadouts, mon, **kw):
    t = {"Attrs": [0], "Levels": [1, 10, 28], "SkillPrio": [], "Train": TRAIN_ALL, "AttrMin": 100, "Haste": True,
         "Loadouts": loadouts, "Commands": ["attack", "berserk"], "Monster": mon, "Difficulty": 1,
         "Races": RACES, "Classes": classes}
    t.update(kw)
    return {k: v for k, v in t.items() if v is not None}  # None drops a field (one SkillPrio source)


def dump(out, name, t):
    with open(os.path.join(out, name), 'w') as f:
        json.dump(t, f)


def rows(path):
    for line in open(path):
        if line.startswith('# pareto'):
            return  # the pareto block repeats rows above
        f = line.rstrip('\n').split('\t')
        if line.startswith(('#', 'race')) or len(f) < 7:
            continue
        yield f


def num(s):
    try:
        return float(s)
    except ValueError:
        return None  # NA


def main():
    cmd = sys.argv[1]
    if cmd == 'screen':
        m = melee(items(sys.argv[2]))
        for mon in MONSTERS:
            dump(sys.argv[3], f'screen-{mon}.json',
                 tmpl(list(CLASSES), [{"Name": f"i{i}", "Main": w(i)} for i in m], mon, Levels=[28]))
    elif cmd == 'main':
        recs, out = items(sys.argv[2]), sys.argv[3]
        m = melee(recs)
        offs = [i for i in m if recs[i][0x41] & 8 and not recs[i][0x41] & 4]
        best = collections.defaultdict(dict)
        for mon in MONSTERS:
            for f in rows(os.path.join(out, f'screen-{mon}.tsv')):
                v = num(f[6])
                if v is not None:
                    c, i = int(f[1]), int(f[2][1:])
                    best[c][i] = max(best[c].get(i, 0), v)
        for c in CLASSES:
            top = [i for i, _ in sorted(best[c].items(), key=lambda x: (-x[1], x[0]))[:TOP]]
            mains = sorted(set(top) | {i for i in MG.get(c, []) if i in m})
            lo = [{"Name": "unarmed"}] + [{"Name": f"i{a}", "Main": w(a)} for a in mains]  # I7: no Main = unarmed, skill 14
            pool = sorted((i for i in offs if i in best[c]), key=lambda i: (-best[c][i], i))[:TOP_OFF]
            pool = sorted(set(pool) | {i for i in MG.get(c, []) if i in offs})
            lo += [{"Name": f"i{a}+i{b}", "Main": w(a), "Off": w(b)} for a in mains for b in pool if a != b]
            for mon in MONSTERS:
                dump(out, f'main-c{c}-{mon}.json', tmpl([c], lo, mon))
        # Reference: points only (no Train, AttrMin, Haste); STR-first vs DEX-first, skill orders enumerated.
        for c, (a, b) in MG_PAIRS.items():
            for mon in MONSTERS:
                dump(out, f'ref-c{c}-{mon}.json', tmpl([c], [{"Name": f"i{a}+i{b}", "Main": w(a), "Off": w(b)}], mon,
                                                      Attrs=[0, 4], Train={}, AttrMin=0, Haste=False,
                                                      SkillPrio=None, SkillSet=[-1, -2, 16, 18]))
                dump(out, f'ref1-c{c}-{mon}.json', tmpl([c], [{"Name": f"i{a}", "Main": w(a)}], mon,
                                                       Attrs=[0, 4], Train={}, AttrMin=0, Haste=False,
                                                       SkillPrio=None, SkillSet=[-1, 16, 34]))
    elif cmd == 'party':
        # Secondary: MG party panel (notes/calc-qa/mg-party-l25.md) through calc fight, plus Haste.
        # Panel primary skill is effective (raw + raw/4); raw is the inverse. Other skills as shown.
        for name, (race, cls, lv, attrs, skills, pairs) in PARTY.items():
            for main_i, off_i in pairs:
                for off in ([off_i, None] if off_i else [None]):
                    for mon in MONSTERS:
                        sc = {"Race": race, "Class": cls, "Level": lv, "Prio": [0], "Attrs": attrs, "Haste": True,
                              "Skills": {str(k): v for k, v in skills.items()}, "Main": w(main_i),
                              "Off": w(off) if off else None, "Mode": 0, "OffMode": 0, "Monster": mon, "Difficulty": 1}
                        tag = f'party-{name}-i{main_i}' + (f'+i{off}' if off else '') + f'-{mon}'
                        dump(sys.argv[2], tag + '.json', sc)
    elif cmd == 'summary':
        summary(sys.argv[2])


def summary(out):
    """Per class/monster/level: best single, best dual, and the MG pair."""
    lv = [1, 10, 28]
    print('| 職業 | 怪 | L | 單持最佳 | 雙持最佳 | 雙/單 | MG 組 | 空手 |')
    print('|---|---|---|---|---|---|---|---|')
    for c in CLASSES:
        for mon in MONSTERS:
            single, dual, mg, bare = {}, {}, {}, {}
            for f in rows(os.path.join(out, f'main-c{c}-{mon}.tsv')):
                for k, l in enumerate(lv):
                    v = num(f[6 + k])
                    if v is None:
                        continue
                    tag = (v, f'{f[2]} {f[3]} r{f[0]}')
                    d = bare if f[2] == 'unarmed' else dual if '+' in f[2] else single
                    if l not in d or v > d[l][0]:
                        d[l] = tag
                    if c in MG_PAIRS and f[2] == 'i%d+i%d' % MG_PAIRS[c] and (l not in mg or v > mg[l][0]):
                        mg[l] = tag
            for l in lv:
                s, d = single.get(l), dual.get(l)
                ratio = f'{d[0] / s[0]:.2f}' if s and d and s[0] > 0 else 'NA'
                fmt = lambda x: f'{x[0]:.0f}（{x[1]}）' if x else 'NA'
                print(f'| {c} | {mon} | {l} | {fmt(s)} | {fmt(d)} | {ratio} | {fmt(mg.get(l))} | {fmt(bare.get(l))} |')
    # Cross-class Pareto over (L1, L10, L28); races with identical rows collapse into one line.
    for mon in MONSTERS:
        pts = collections.defaultdict(list)
        for c in CLASSES:
            for f in rows(os.path.join(out, f'main-c{c}-{mon}.tsv')):
                v = tuple(num(x) if num(x) is not None else float('-inf') for x in f[6:9])
                pts[(c, f[2], f[3], v)].append(f[0])
        keys = list(pts)
        front = [k for k in keys if not any(all(a >= b for a, b in zip(o[3], k[3])) and o[3] != k[3] for o in keys)]
        print(f'\n怪 {mon} 跨職業 Pareto（L1／L10／L28）：\n')
        for c, lo, cmd, v in sorted(front, key=lambda k: -k[3][2]):
            print(f'- c{c} {lo} {cmd} r{",".join(pts[(c, lo, cmd, v)])}：' + '／'.join(f'{x:.0f}' for x in v))


if __name__ == "__main__":
    main()
