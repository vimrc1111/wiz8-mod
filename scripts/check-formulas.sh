#!/usr/bin/env bash
# check-formulas.sh [--progress]：公式登記簿的機檢（goals/stage10.md B1／B2）。
# 預設＝完成模式：`公式` 類全部 D4 或合格 `未取得`。--progress 只驗結構與已宣稱的深度，印深度分布。
# 輸入（環境變數可覆寫，給 check-formulas.test.sh 用夾具）：
#   FUNCS  Ghidra 函式表（re/Cands.java；入口 大小 前16 呼叫 記錄基址 資料引用 基本塊）預設 build/re/funcs.tsv（原版衍生，不入 git）
#   FIELDS 欄位運算元表（入口<TAB>運算元,運算元…）預設 build/re/fields.tsv；另行執行：
#          bash scripts/func-fields.sh build/wiz8.asm build/re/funcs.tsv > build/re/fields.tsv
#          formula-candidates.sh 無參數會用 Ghidra 重產 funcs.tsv；本檢查不自動產生 fields.tsv。
#   FG     登記簿 notes/re/formula-graph.tsv：首行 `#candidates<TAB>數量<TAB>sha256`，次行表頭，之後 `addr class depth node calc evidence`
#   FGD    分批登記目錄 notes/re/formula-graph.d/*.tsv（平行票各寫一檔，欄位同上，可有表頭）；與 FG 合併，同位址出現兩次即紅
#   ONLY   只檢這份位址清單（一行一個 hex）是否都已歸類；分批票的驗收用，搭配 --progress
#   IFACE  D1 介面目錄 notes/re/iface/<addr>.tsv：`call|data|field<TAB>目標hex或欄位運算元<TAB>名稱或「無關」<TAB>理由`
#   CALC   D2 純函數目錄 internal/calc；EMU D3／D4 證據目錄 build/emu（<addr>.d3：`cases=N diff=0`；<addr>.hit：命中的基本塊，一行一個 hex；檔名可補零到 8 位，re/emu/run.sh 就是）
#   VTE    Ghidra 漏掉的 vtable 入口 notes/re/vtable-entries.tsv：`addr<TAB>槽位址hex<TAB>理由`；EXE（預設 build/Wiz8.exe）該槽的 dword 要等於 addr 才算函式入口，只准登記「排除」類
#   COVX   D4 走不到的基本塊登記 notes/re/cov-exempt.tsv：`addr<TAB>塊hex<TAB>理由`；同名目錄 cov-exempt.d/*.tsv（節點票各寫一檔）一併讀
set -uo pipefail
here=$(cd "$(dirname "$0")/.." && pwd)
FUNCS=${FUNCS:-$here/build/re/funcs.tsv} FG=${FG:-$here/notes/re/formula-graph.tsv} IFACE=${IFACE:-$here/notes/re/iface}
FIELDS=${FIELDS:-$here/build/re/fields.tsv}
fields_ok=0; [ -f "$FIELDS" ] && fields_ok=1
FGD=${FGD:-$here/notes/re/formula-graph.d} ONLY=${ONLY:-}
CALC=${CALC:-$here/internal/calc} EMU=${EMU:-$here/build/emu} COVX=${COVX:-$here/notes/re/cov-exempt.tsv}
VTE=${VTE:-$here/notes/re/vtable-entries.tsv} EXE=${EXE:-$here/build/Wiz8.exe}
ASM=${ASM:-$here/build/wiz8.asm} ASMSHA=${ASMSHA:-$here/notes/re/asm.sha256}
mode=final; [ "${1:-}" = --progress ] && mode=progress
[ -s "$FUNCS" ] || { echo "FAIL funcs：$FUNCS 不存在（先跑 bash scripts/formula-candidates.sh 產生）"; exit 1; }
[ -s "$FG" ] || { echo "FAIL graph：$FG 不存在"; exit 1; }
cand=$(bash "$here/scripts/formula-candidates.sh" "$FUNCS") || exit 1
n=$(printf '%s\n' "$cand" | grep -c .); sha=$(printf '%s\n' "$cand" | sha256sum | cut -d' ' -f1)
# calc 裡已定義的函數名（非測試檔）；有 TODO 就整個 D2 失格
calcfns=$(cat "$CALC"/*.go 2>/dev/null | grep -v '^//' | sed -n 's/^func \(([^)]*) \)\{0,1\}\([A-Za-z0-9_]*\)(.*/\2/p' | sort -u | tr '\n' ,)
todo=$(grep -ln 'TODO' "$CALC"/*.go 2>/dev/null | grep -v '_test.go' | tr '\n' ' ')
covx=$(cat "$COVX" "${COVX%.tsv}.d"/*.tsv 2>/dev/null | awk -F'\t' 'NF>=3 && $3!="" {print $1":"$2}' | tr '\n' ,)
# 同名 .jsonl 比 .d3 新＝差分產物換過、.d3 是舊的，不算（grok zoWN 發現 4）
d3=$(for f in "$EMU"/*.d3; do [ -f "$f" ] && ! [ "${f%.d3}.jsonl" -nt "$f" ] && grep -qE '^cases=[1-9][0-9]* diff=0$' "$f" && basename "$f" .d3 | sed 's/^0*//'; done 2>/dev/null | tr '\n' ,)
hits=$(for f in "$EMU"/*.hit; do [ -f "$f" ] && sed "s/^/$(basename "$f" .hit | sed 's/^0*//'):/" "$f"; done 2>/dev/null | tr '\n' ,)
ifaces=$(for f in "$IFACE"/*.tsv; do [ -f "$f" ] && awk -F'\t' -v a="$(basename "$f" .tsv)" 'NF>=4 && $3!="" && $4!="" {print a":"$1":"$2":"$3}' "$f"; done 2>/dev/null | tr '\n' ,)

# vtable 入口：從 exe 讀槽位的 dword 對 addr；不符或沒有 exe 就不算入口（awk 端照 nofunc 紅）
vtok=$(grep -v '^#' "$VTE" 2>/dev/null | python3 -I -c '
import struct, sys
try: d = open(sys.argv[1], "rb").read()
except OSError: sys.exit(0)
pe = struct.unpack_from("<I", d, 0x3c)[0]; ib = struct.unpack_from("<I", d, pe+0x34)[0]
so = pe+0x18+struct.unpack_from("<H", d, pe+0x14)[0]
secs = [struct.unpack_from("<IIII", d, so+40*i+8) for i in range(struct.unpack_from("<H", d, pe+6)[0])]
for l in sys.stdin:
    r = l.rstrip("\n").split("\t")
    if len(r) < 3 or not r[2]: continue
    va = int(r[1], 16)-ib
    for vs, sva, rs, ro in secs:
        if sva <= va < sva+min(vs, rs)-3 and struct.unpack_from("<I", d, ro+va-sva)[0] == int(r[0], 16): print(r[0])
' "$EXE" | tr '\n' ,)
# 證據行號 build/wiz8.asm:<行> 依賴 asm 逐行相同（scripts/gen-asm.sh）；第 2 行是 exe 路徑，不計
asmbad=; if [ ! -s "$ASM" ]; then asmbad="asm：$ASM 不存在（先跑 bash scripts/gen-asm.sh）"
elif [ "$(sed 2d "$ASM" | sha256sum | cut -d' ' -f1)" != "$(cut -f1 "$ASMSHA" 2>/dev/null)" ]; then asmbad="asm：$ASM 雜湊與 $ASMSHA 不符（證據行號可能錯位；用 scripts/gen-asm.sh 重產）"; fi
frags=$(ls "$FGD"/*.tsv 2>/dev/null | tr '\n' '|')
# 大字串走檔案不走 -v：節點多了 hits／ifaces 超過 ARG_MAX，awk 起不來（exit 126、沒有 FAIL 行）
V=$(mktemp -d); trap 'rm -rf "$V"' EXIT
for k in calcfns covx d3 hits ifaces vtok; do printf ',%s' "${!k}" > "$V/$k"; done
printf '%s\n' "$cand" | awk -F'\t' -v mode="$mode" -v n="$n" -v sha="$sha" -v V="$V" -v todo="$todo" -v FG="$FG" -v FUNCS="$FUNCS" -v FIELDS="$FIELDS" -v fields_ok="$fields_ok" -v frags="$frags" -v ONLY="$ONLY" -v asmbad="$asmbad" '
function bad(m) { print "FAIL " m; nb++ }
function addrow(l,   r, a) {
  if (l ~ /^#/ || l == "" || l ~ /^addr\t/) return
  split(l, r, "\t"); a = r[1]
  if (a in row) { bad("dup " a " 重複列"); return }
  row[a] = 1; cls[a] = r[2]; dep[a] = r[3]; node[a] = r[4]; calcf[a] = r[5]; ev[a] = r[6]
}
function lv(d) { return d ~ /^D[0-4]$/ ? substr(d, 2) + 0 : -1 }
function slurp(k,   l) { getline l < (V "/" k); close(V "/" k); return l }
BEGIN {
  calcfns = slurp("calcfns"); covx = slurp("covx"); d3 = slurp("d3"); hits = slurp("hits"); ifaces = slurp("ifaces"); vtok = slurp("vtok")
  while ((getline l < FUNCS) > 0) { split(l, x, "\t"); calls[x[1]] = x[4]; data[x[1]] = x[6]; blocks[x[1]] = x[7]; isfn[x[1]] = 1 }
  split(vtok, vt, ","); for (i in vt) if (vt[i] != "") { isfn[vt[i]] = 1; isvt[vt[i]] = 1 }
  if (fields_ok) while ((getline l < FIELDS) > 0) { split(l, x, "\t"); fields[x[1]] = x[2] }
  split(ifaces, it, ","); for (i in it) if (it[i] != "") { split(it[i], y, ":"); ifc[y[1] ":" y[2] ":" y[3]] = y[4] }
}
{ iscand[$1] = 1 }
END {
  if (asmbad != "") bad(asmbad)
  ln = 0
  while ((getline l < FG) > 0) {
    ln++
    if (ln == 1) { split(l, h, "\t"); if (h[1] != "#candidates" || h[2] != n || h[3] != sha) bad("candidates 候選清單變了：登記簿 " h[2] "／" substr(h[3], 1, 16) "，現在 " n "／" substr(sha, 1, 16)); continue }
    addrow(l)
  }
  nf = split(frags, fl, "|"); for (i = 1; i <= nf; i++) if (fl[i] != "") while ((getline l < fl[i]) > 0) addrow(l)
  if (ONLY != "") { while ((getline l < ONLY) > 0) if (l != "") { only[l] = 1; if (!(l in iscand)) bad("only " l " 不在候選清單") } }
  for (a in iscand) if ((ONLY == "" || (a in only)) && !(a in row)) bad("unclassified " a " 候選沒有登記列")
  for (a in row) {
    if (!(a in isfn)) { bad("nofunc " a " 不是 exe 內的函式入口"); continue }
    c = cls[a]; d = dep[a]
    if ((a in isvt) && !((a in calls) || c == "排除")) { bad("vtable " a " 只在 vtable 入口表，只准登記「排除」"); continue }
    if (c == "非公式" || c == "排除") {
      if (d != "-") bad("class " a " " c " 的深度欄要寫 -")
      if (ev[a] !~ /wiz8\.asm:[0-9]+/) bad("evidence " a " " c " 缺證據行 build/wiz8.asm:<行>")
      continue
    }
    if (c != "公式") { bad("class " a " 歸類「" c "」不是 公式／非公式／排除"); continue }
    if (d == "未取得") {
      if (ev[a] !~ /影響:/ || ev[a] !~ /嘗試:/) bad("unobtained " a " 未取得要寫「嘗試:」與「影響:」")
      hist["未取得"]++; continue
    }
    k = lv(d); if (k < 0) { bad("depth " a " 深度「" d "」不是 D0–D4／未取得"); continue }
    hist[d]++
    if (mode == "final" && k < 4) bad("final " a " " node[a] " 只到 " d "，完成要 D4 或未取得")
    if (k >= 1) {  # D1：子呼叫閉合；每個資料引用與欄位運算元要在介面檔命名
      if (!fields_ok && !fields_missing++) bad("fields：" FIELDS " 不存在（先跑 bash scripts/func-fields.sh build/wiz8.asm \"" FUNCS "\" > \"" FIELDS "\" 產生）")
      nc = split(calls[a], cs, ","); for (i = 1; i <= nc; i++) { t = cs[i]
        if (!(t in row) && !((a ":call:" t) in ifc)) bad("d1 " a " 子呼叫 " t " 沒登記成節點、也沒在 iface 標名稱或無關")
        else if ((t in row) && cls[t] == "公式" && lv(dep[t]) >= 0 && lv(dep[t]) < 1 && k >= 2) bad("d1 " a " 依賴 " t " 只到 " dep[t]) }
      nd = split(data[a], ds, ","); for (i = 1; i <= nd; i++) if (!((a ":data:" ds[i]) in ifc)) bad("d1 " a " 資料引用 " ds[i] " 沒在 iface 命名")
      nf2 = split(fields[a], fs, ","); for (i = 1; i <= nf2; i++) if (!((a ":field:" fs[i]) in ifc)) bad("d1 " a " 欄位 " fs[i] " 沒在 iface 命名（需名稱或無關與理由）")
    }
    if (k >= 2) {
      if (calcf[a] == "" || index(calcfns, "," calcf[a] ",") == 0) bad("d2 " a " calc 欄「" calcf[a] "」不是 internal/calc 的函數")
      if (todo != "") bad("d2 " a " internal/calc 有 TODO：" todo)
    }
    if (k >= 3 && index(d3, "," a ",") == 0) bad("d3 " a " 缺差分證據 build/emu/" a ".d3（cases>0 diff=0）")
    if (k >= 4) {
      nb2 = split(blocks[a], bs, ","); for (i = 1; i <= nb2; i++)
        if (index(hits, "," a ":" bs[i] ",") == 0 && index(covx, "," a ":" bs[i] ",") == 0) bad("d4 " a " 基本塊 " bs[i] " 沒走到、也沒登記 cov-exempt")
    }
  }
  for (hk in hist) printf "  %s %d\n", hk, hist[hk]
  if (nb) { printf "check-formulas: %d 處不符（%s 模式）\n", nb, mode; exit 1 }
  printf "ok check-formulas：%d 候選、%d 列（%s 模式）\n", n, length(row), mode
}'
