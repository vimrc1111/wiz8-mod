#!/usr/bin/env bash
# dbsname.test.sh：票 M 驗收。ITEMS.DBS／MONSTERS.DBS 名稱欄 decode／encode（tools/dbsname）。
# 原檔從玩家自有 DATA.SLF 現抽（不入版控）；找不到原版就 FAIL（沒原版等於沒驗）。
# 1 往返逐 byte 相同；2 改一欄只動新字＋一個 NUL，其後原 bytes 不動（物品欄尾 2 units、怪物欄尾 1 unit 是記錄資料）；3 超長拒絕且不寫出；4 譯文經 build-zh-patch.sh 與 zhpatch 都進 PATCH.002 且兩者相同。
# spells（票 N，SPELLTABLES.DBS）：檔頭 8 bytes（筆數＋版本），其餘同。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
G=${WIZ8_DIR:-"/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8"}
[ -f "$G/Data/DATA.SLF" ] || { echo "FAIL dbsname 找不到原版 $G"; exit 1; }
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT
go build -o "$S/slf" ./tools/slf; go build -o "$S/zhpatch" ./tools/zhpatch
go build -o "$S/dbsname" ./tools/dbsname || { echo "FAIL dbsname tools/dbsname 編譯失敗或不存在"; exit 1; }
"$S/slf" list "$G/Data/DATA.SLF" > "$S/list"
member() { # member <SLF 內檔名> <輸出>
  read -r off size < <(N="$1" awk -F'\t' '$3==ENVIRON["N"] {print $1, $2}' "$S/list")
  dd if="$G/Data/DATA.SLF" of="$2" iflag=skip_bytes,count_bytes skip="$off" count="$size" status=none
}
member 'DATABASES\ITEMS.DBS' "$S/items"; member 'DATABASES\MONSTERS.DBS' "$S/monsters"; member 'DATABASES\SPELLTABLES.DBS' "$S/spells"

# 1 往返；筆數與每筆欄數
for k in items monsters spells; do
  "$S/dbsname" decode "$k" "$S/$k" > "$S/$k.json" || { echo "FAIL dbsname $k decode 失敗"; exit 1; }
  "$S/dbsname" encode "$k" "$S/$k.json" "$S/$k.rt"
  cmp "$S/$k" "$S/$k.rt" || { echo "FAIL dbsname $k 往返不一致"; exit 1; }
done
[ "$(jq '.records | length' "$S/items.json")" = 819 ] && [ "$(jq '[.records[].text | length] | unique' -c "$S/items.json")" = '[1]' ] \
  || { echo "FAIL dbsname items 應 819 筆、每筆 1 欄"; exit 1; }
[ "$(jq '.records | length' "$S/monsters.json")" = 595 ] && [ "$(jq '[.records[].text | length] | unique' -c "$S/monsters.json")" = '[4]' ] \
  || { echo "FAIL dbsname monsters 應 595 筆、每筆 4 欄"; exit 1; }
[ "$(jq '.records | length' "$S/spells.json")" = 150 ] && [ "$(jq '[.records[].text | length] | unique' -c "$S/spells.json")" = '[1]' ] \
  || { echo "FAIL dbsname spells 應 150 筆、每筆 1 欄"; exit 1; }
echo "PASS dbsname 往返逐 byte 相同（items 819×1、monsters 595×4、spells 150×1）"

# 2 改一欄：只有新字＋一個 NUL 的位元組變，欄內其後保留原 bytes（記錄 stride、欄位位移、欄寬照 R6）
check_one() { # check_one <kind> <rec> <field> <stride> <off> <width> <text> [檔頭 bytes，預設 4]
  jq --argjson r "$2" --argjson f "$3" --arg t "$7" '.records[$r].text[$f] = $t' "$S/$1.json" > "$S/$1.mod.json"
  "$S/dbsname" encode "$1" "$S/$1.mod.json" "$S/$1.mod"
  python3 -I - "$S/$1" "$S/$1.mod" "$2" "$4" "$5" "$6" "$7" "${8:-4}" <<'PY' || { echo "FAIL dbsname $1 記錄 $2 欄 $3 改寫不對"; exit 1; }
import sys
a, b = open(sys.argv[1], 'rb').read(), open(sys.argv[2], 'rb').read()
rec, stride, off, width, text, hdr = int(sys.argv[3]), int(sys.argv[4]), int(sys.argv[5]), int(sys.argv[6]), sys.argv[7], int(sys.argv[8])
lo = hdr + rec * stride + off; hi = lo + 2 * width
assert len(a) == len(b) and a[:lo] == b[:lo] and a[hi:] == b[hi:], "欄外有變動"
want = text.encode('utf-16le') + b'\0\0'
assert b[lo:lo + len(want)] == want, "欄開頭不是新字＋NUL"
assert b[lo + len(want):hi] == a[lo + len(want):hi], "NUL 之後的原 bytes 被改（記錄資料）"
PY
}
check_one items 7 0 $((0x10d)) 0 32 '長劍'
check_one monsters 589 1 $((0x297)) $((0x30)) 24 '無煙煤魔像'
check_one monsters 589 3 $((0x297)) $((0x90)) 24 '魔像'
check_one spells 1 0 $((0x2c0)) $((0x19c)) 64 '酸液飛濺' 8
echo "PASS dbsname 改一欄只動該欄"

# 3 超長（新字＋NUL 不得碰到欄尾資料：物品 >29、怪物 >22、法術 >63 units）→ 非零、不寫出
long() { python3 -I -c "print('字' * $1)"; }
for c in "items 0 0 $(long 30)" "monsters 0 2 $(long 23)" "spells 1 0 $(long 64)"; do
  set -- $c
  jq --argjson r "$2" --argjson f "$3" --arg t "$4" '.records[$r].text[$f] = $t' "$S/$1.json" > "$S/$1.long.json"
  rm -f "$S/$1.long"
  if "$S/dbsname" encode "$1" "$S/$1.long.json" "$S/$1.long" 2>/dev/null; then echo "FAIL dbsname $1 超長卻成功"; exit 1; fi
  [ ! -e "$S/$1.long" ] || { echo "FAIL dbsname $1 超長仍寫出"; exit 1; }
done
ok29=$(long 29); jq --arg t "$ok29" '.records[0].text[0] = $t' "$S/items.json" > "$S/items.31.json"
"$S/dbsname" encode items "$S/items.31.json" "$S/items.31" || { echo "FAIL dbsname items 29 units 應可"; exit 1; }
ok63=$(long 63); jq --arg t "$ok63" '.records[1].text[0] = $t' "$S/spells.json" > "$S/spells.63.json"
"$S/dbsname" encode spells "$S/spells.63.json" "$S/spells.63" || { echo "FAIL dbsname spells 63 units 應可"; exit 1; }
echo "PASS dbsname 超長拒絕（物品 29 units 可、30 拒；怪物 23 拒；法術 63 可、64 拒）"

# 4 整合：譯文 id＝物品 記錄序號；怪物 4×記錄＋欄位。用出貨譯文原樣（只換一份會踩全表機檢，如 spell-names；稽核 Acodec F1）。
# build-zh-patch.sh 與 zhpatch 產物相同，且 PATCH.002 內是譯文（期望值從譯文檔讀）
cp -r translations "$S/tr"
zh() { jq -r --argjson i "$2" '.[] | select(.id == $i) | .zh' "translations/$1.json"; }
TRANSLATIONS="$S/tr/stringdata.json" bash scripts/build-zh-patch.sh "$S/want" >"$S/build.log" 2>&1 || { tail -5 "$S/build.log"; echo "FAIL dbsname build-zh-patch.sh 帶 DBS 名稱譯文失敗"; exit 1; }
"$S/zhpatch" "$G" "$S/tr" "$S/got" || { echo "FAIL dbsname zhpatch 帶 DBS 名稱譯文失敗"; exit 1; }
cmp "$S/want" "$S/got" || { echo "FAIL dbsname build-zh-patch.sh 與 zhpatch 產物不同"; exit 1; }
"$S/slf" extract "$S/got" "$S/x" >/dev/null
for k in items monsters spells; do
  n=$k; [ $k != spells ] || n=spelltables
  f=$(find "$S/x" -iname "$n.dbs" | head -1); [ -n "$f" ] || { echo "FAIL dbsname PATCH.002 沒有 $k.dbs"; exit 1; }
  "$S/dbsname" decode "$k" "$f" > "$S/$k.got.json"
done
[ "$(jq -r '.records[7].text[0]' "$S/items.got.json")" = "$(zh items 7)" ] && [ "$(jq -r '.records[589].text[1]' "$S/monsters.got.json")" = "$(zh monsters 2357)" ] \
  && [ "$(jq -r '.records[1].text[0]' "$S/spells.got.json")" = "$(zh spells 1)" ] \
  || { echo "FAIL dbsname PATCH.002 內名稱不是譯文"; exit 1; }
# sha 不符要拒
jq '.[0].src_sha256 = ("0" * 64)' "$S/tr/monsters.json" > "$S/m" && mv "$S/m" "$S/tr/monsters.json"; rm -f "$S/bad"
if "$S/zhpatch" "$G" "$S/tr" "$S/bad" 2>/dev/null; then echo "FAIL dbsname zhpatch 怪物 sha 不符卻成功"; exit 1; fi
[ ! -e "$S/bad" ] || { echo "FAIL dbsname zhpatch 怪物 sha 不符仍寫出"; exit 1; }
echo "PASS dbsname 整合：兩條建置路相同、PATCH.002 內為譯文、sha 不符被拒"
echo "ok dbsname.test"
