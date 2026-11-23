#!/usr/bin/env python3
"""Read-only DBS source lookup. See notes/calc.md:263 for limitations."""
import argparse
from pathlib import Path
import struct
import sys


class FormatError(ValueError):
    pass


class Reader:
    def __init__(self, path):
        self.label = str(path)
        self.data = path.read_bytes()

    def take(self, offset, size):
        if offset < 0 or size < 0 or offset + size > len(self.data):
            raise FormatError(f'{self.label}: truncated at offset {offset:#x}, need {size} bytes')
        return self.data[offset:offset + size]

    def unpack(self, fmt, offset):
        return struct.unpack(fmt, self.take(offset, struct.calcsize(fmt)))

    def eof(self, offset):
        if offset != len(self.data):
            raise FormatError(f'{self.label}: length/count mismatch: expected EOF {offset}, got {len(self.data)}')


def name(data, wide=False):
    if wide:
        end = next((i for i in range(0, len(data), 2) if data[i:i + 2] == b'\0\0'), None)
    else:
        end = data.find(b'\0')
        if end < 0:
            end = None
    if end is None:
        raise FormatError('name field lacks NUL terminator')
    return data[:end].decode('utf-16le' if wide else 'cp1252')


def dice(data):
    z, n, s = struct.unpack('<hBB', data)
    return f'{z}+{n}d{s}'


def fixed(path, stride):
    r = Reader(path)
    count, = r.unpack('<I', 0)
    r.eof(4 + count * stride)
    return [r.take(4 + i * stride, stride) for i in range(count)]


def valid_ref(ref, count, label):
    if not 0 <= ref < count:
        raise FormatError(f'{label}: reference {ref} outside 0..{count - 1}')


def load(directory):
    items = [name(b[:64], True) for b in fixed(directory / 'ITEMS.DBS', 0x10d)]
    r = Reader(directory / 'ITEMTABLES.DBS')
    groups, = r.unpack('<I', 0)
    count_offset = 4 + groups * 256
    r.take(4, groups * 256)
    for i in range(groups):
        name(r.take(4 + i * 256, 256))
    count, = r.unpack('<I', count_offset)
    start = count_offset + 4
    r.eof(start + count * 497)
    tables = []
    for t in range(count):
        b = r.take(start + t * 497, 497)
        slots = []
        for j in range(40):
            category, ref, w = struct.unpack_from('<HHB', b, 0x104 + 5 * j)
            if category:
                valid_ref(ref, len(items), f'table {t} slot {j}')
                slots.append((j, ref, w))
        tables.append(dict(name=name(b[:256]), slots=slots,
                           flag=b[0x1cc], dice=dice(b[0x1cd:0x1d1])))
    monsters = []
    for m, b in enumerate(fixed(directory / 'MONSTERS.DBS', 0x297)):
        slots = []
        for j in range(8):
            offset = 0x1f3 + j * 10
            kind, category, ref, c = struct.unpack_from('<BHHB', b, offset)
            if category and kind in (0, 1):
                valid_ref(ref, len(items) if kind == 0 else len(tables), f'monster {m} slot {j}')
            slots.append(dict(slot=j, kind=kind, category=category, ref=ref,
                              c=c, dice=dice(b[offset + 6:offset + 10])))
        monsters.append(dict(name=name(b[:48], True), slots=slots))
    r = Reader(directory / 'NPC.DBS')
    count, = r.unpack('<I', 0)
    cursor = 4
    npcs = []
    # Each record has a fixed body, followed by a conditional stock list.
    if count > (len(r.data) - 4) // 777:
        raise FormatError(f'{r.label}: record count exceeds available bodies')
    for npc_id in range(count):
        b = r.take(cursor, 777)
        cursor += 777
        d, = struct.unpack_from('<I', b, 0x64)
        stock = []
        if struct.unpack_from('<H', b, 0)[0] > 1 and b[0x9d] == 0:
            n, = r.unpack('<I', cursor)
            cursor += 4
            raw = r.take(cursor, n * 6)
            cursor += n * 6
            for ref, quantity, flag in struct.iter_unpack('<IBB', raw):
                valid_ref(ref, len(items), f'NPC {npc_id} stock')
                stock.append((ref, quantity, flag))
        npcs.append(dict(name=name(b[4:0x64], True),
                         death=d - 1 if 0 < d < len(tables) else None, stock=stock))
    r.eof(cursor)
    return items, tables, monsters, npcs


def row(*values):
    # Escape separators so each record remains one TSV line.
    print('\t'.join(str(v).replace('\\', '\\\\').replace('\t', '\\t').replace('\r', '\\r').replace('\n', '\\n') for v in values))


def select(names, query):
    if query.isdecimal():
        ref = int(query)
        valid_ref(ref, len(names), 'query')
        return [ref]
    matches = [i for i, value in enumerate(names) if query.casefold() in value.casefold()]
    if not matches:
        raise FormatError(f'no match for {query!r}')
    return matches


def monster_table_index(slot):
    return slot['ref']  # Monster kind 1 references are zero-based.


def table_row(t, table, slot, item, w):
    row('TABLE', t, table['name'], slot, item, w, len(table['slots']),
        sum(s[2] for s in table['slots']), table['flag'], table['dice'])


def source_row(m, monster, slot, via):
    row('MONSTER_SOURCE', m, monster['name'], slot['slot'], slot['kind'], via,
        slot['c'], f"{min(slot['c'] + 1, 100) / 100:.2f}", slot['dice'])


def item_query(ids, items, tables, monsters, npcs):
    for item in ids:
        row('ITEM', item, items[item])
        containing = set()
        for t, table in enumerate(tables):
            for j, ref, w in table['slots']:
                if ref == item:
                    containing.add(t)
                    table_row(t, table, j, item, w)
        for m, monster in enumerate(monsters):
            for slot in monster['slots']:
                if not slot['category']:
                    continue
                if slot['kind'] == 0 and slot['ref'] == item:
                    source_row(m, monster, slot, 'direct')
                elif slot['kind'] == 1 and monster_table_index(slot) in containing:
                    source_row(m, monster, slot, monster_table_index(slot))
        for n, npc in enumerate(npcs):
            if npc['death'] in containing:
                row('NPC_DEATH', n, npc['name'], npc['death'])
            for ref, quantity, flag in npc['stock']:
                if ref == item:
                    row('NPC_SHOP', n, npc['name'], ref, quantity, flag)


def monster_query(ids, items, tables, monsters):
    for m in ids:
        monster = monsters[m]
        row('MONSTER', m, monster['name'])
        for slot in monster['slots']:
            kind, ref = slot['kind'], slot['ref']
            target = 'empty' if not slot['category'] else 'unhandled kind'
            if slot['category'] and kind == 0:
                target = items[ref]
            elif slot['category'] and kind == 1:
                ref = monster_table_index(slot)
                target = tables[ref]['name']
            row('SLOT', slot['slot'], kind, slot['category'], ref, target,
                slot['c'], f"{min(slot['c'] + 1, 100) / 100:.2f}", slot['dice'])
            if slot['category'] and kind == 1:
                table = tables[ref]
                for j, item, w in table['slots']:
                    table_row(ref, table, j, item, w)
                    row('TABLE_ITEM', ref, j, item, items[item], w)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('item', 'monster'))
    parser.add_argument('databases', type=Path)
    parser.add_argument('query')
    args = parser.parse_args()
    try:
        items, tables, monsters, npcs = load(args.databases)
        ids = select(items if args.mode == 'item' else [m['name'] for m in monsters], args.query)
        row('# TABLE: id name slot item w nonempty_slots w_sum filter_flag draw_dice; w/w_sum is NOT a probability')
        row('# MONSTER_SOURCE: id name slot kind via_table_or_direct c approximate_gate_probability quantity_dice')
        row('# NPC_DEATH: id name table; NPC_SHOP: id name item quantity flag')
        row('# SLOT: slot kind category ref name_or_status c approximate_gate_probability quantity_dice')
        row('# TABLE_ITEM: table slot item name w; all IDs and slots are zero-based')
        if args.mode == 'item':
            item_query(ids, items, tables, monsters, npcs)
        else:
            monster_query(ids, items, tables, monsters)
    except (OSError, ValueError, struct.error) as exc:
        print(f'ERROR loot-where: {exc}', file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
