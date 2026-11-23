#!/usr/bin/env bash
# zhpatch.test.sh：票 L 驗收。tools/zhpatch 的產物要與 scripts/build-zh-patch.sh 逐 byte 相同；原文 sha 不符要拒絕且不寫出；能交叉編譯成 Windows exe。
# 需要玩家自有原版（WIZ8_DIR）；找不到就 FAIL（不 skip：沒原版等於沒驗）。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
G=${WIZ8_DIR:-"/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8"}
[ -f "$G/Data/DATA.SLF" ] || { echo "FAIL zhpatch 找不到原版 $G"; exit 1; }
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT
go build -o "$S/zhpatch" ./tools/zhpatch
GOOS=windows GOARCH=amd64 go build -o "$S/zhpatch.exe" ./tools/zhpatch || { echo "FAIL zhpatch Windows 交叉編譯"; exit 1; }
bash scripts/build-zh-patch.sh "$S/want" >/dev/null 2>&1 || { echo "FAIL zhpatch 參考產物 build-zh-patch.sh 失敗"; exit 1; }
"$S/zhpatch" "$G" translations "$S/got" || { echo "FAIL zhpatch 執行失敗"; exit 1; }
cmp "$S/want" "$S/got" || { echo "FAIL zhpatch 與 build-zh-patch.sh 產物不同"; exit 1; }
echo "PASS zhpatch：逐 byte 相同（$(sha256sum "$S/got" | cut -c1-16)…）"
# 注入：一筆 src_sha256 改壞（NSF、字串庫、STRINGDATA 各一）→ 要非零退出、不寫出輸出檔
for f in nsf/NPC_MYLES.json msg/ARNIKA.json stringdata.json fact.json; do
  rm -rf "$S/tr"; cp -r translations "$S/tr"
  python3 -I - "$S/tr/$f" <<'PY'
import json, sys
p = sys.argv[1]; d = json.load(open(p))
d[0]["src_sha256"] = "0" * 64
open(p, "w").write(json.dumps(d, ensure_ascii=False))
PY
  rm -f "$S/bad"
  if "$S/zhpatch" "$G" "$S/tr" "$S/bad" 2>/dev/null; then echo "FAIL zhpatch $f sha 不符卻成功"; exit 1; fi
  [ ! -e "$S/bad" ] || { echo "FAIL zhpatch $f sha 不符仍寫出檔案"; exit 1; }
  echo "PASS zhpatch：$f sha 不符被拒"
done
echo "ok zhpatch.test"
