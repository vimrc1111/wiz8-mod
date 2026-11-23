#!/usr/bin/env bash
set -euo pipefail
here=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
# 由 case 欄位名稱取索引，判斷 noticeFmt0=39、noticeFmt1=0，
# 即人工格式為單一 apostrophe，NUL 後的字元不會參與格式化。
# 路徑還須 cond=5、target.type=1、charIdx=0..7、message 低位元組非零，
# r552250 非零且 C.buff5.dur 非零，u32(r552250-C.buff5.dur) 作 i32 <=0。
# 此時 551eb0 才會呼叫 590950；只含該文字但沒走通知分支不計數。
python3 -I - "$here" <<'PY'
import json
import sys
from pathlib import Path

try:
    root = Path(sys.argv[1])
    case = json.loads((root / 're/emu/cases/00551eb0.json').read_text())
    ix = {f['name']: i for i, f in enumerate(case['in'])}
    count = 0
    with (root / 'build/emu/00551eb0.jsonl').open() as rows:
        for line in rows:
            r = json.loads(line)['in']
            v = lambda name: r[ix[name]]
            delta = (v('r552250') - v('C.buff5.dur')) & 0xffffffff
            if (v('noticeFmt0') == 39 and v('noticeFmt1') == 0
                    and v('cond') == 5 and v('target.type') == 1
                    and 0 <= v('charIdx') < 8 and v('message') & 255
                    and v('r552250') != 0 and v('C.buff5.dur') != 0
                    and (delta == 0 or delta >= 0x80000000)):
                count += 1
    if not count:
        raise ValueError('no matching path')
except (OSError, ValueError, KeyError, IndexError, TypeError):
    print('FAIL g2-probe')
    sys.exit(1)
print(f'ok g2-probe {count}')
PY
