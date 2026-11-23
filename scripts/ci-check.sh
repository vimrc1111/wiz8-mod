#!/usr/bin/env bash
# 專案檢查步驟；由 scripts/ci-run.sh 在 git archive 抽出的樹裡跑（cwd＝那棵樹）。
# 互不相干的檢查分道平行跑（各自 mktemp；只有 hook 道共用 hook/ddraw/out，道內照順序）。各道 log 跑完依序印出，任一道非 0 即紅。
set -euo pipefail
# 抽出樹在 /tmp/tmp.X/src：上層若有別人留的 .git，go build 的 VCS 戳記會 exit 128（10-31 兩輪 tr／lib／zhpatch 紅；isolated_simulation 重現）。工具不需要戳記。
export GOFLAGS=-buildvcs=false
test -z "$(gofmt -l .)" || { gofmt -l .; echo "gofmt 不乾淨" >&2; exit 1; }
go vet ./...
L=$(mktemp -d); trap 'rm -rf "$L"' EXIT
# 子殼不能放在 if／&&／|| 裡：那會讓子殼內的 set -e 失效；先 set +e 再取 $?
lane() { local n=$1; shift; set +e; ( set -euo pipefail; "$@" ) > "$L/$n.log" 2>&1; echo $? > "$L/$n.rc"; }
# emu.test.sh 會重寫 build/emu 的證據，跟 go test 的 TestEmu 同道照順序跑（分道平行時 TestEmu 讀到刪掉一半的檔）
go_tests() { go test -count=1 ./...; bash scripts/emu.test.sh; bash scripts/calc-qa.test.sh --progress; bash scripts/party.test.sh; bash scripts/fight.test.sh; }  # 沒有 build/Wiz8.exe 的 CI 副本印 SKIP；D3／D4 證據在本機跑
hook_tests() {
  bash scripts/check-research.test.sh
  bash scripts/check-formulas.test.sh
  python3 -I scripts/i9-party.py selftest
  bash scripts/loot-where.test.sh
  bash scripts/nodes/merge-node.test.sh
  bash scripts/wiz8-run.test.sh
  bash scripts/check-sigs.sh
  # ddraw 代理（4.2 繪製 detour）：建得出來（-Werror、detour 簽章讀 notes/re/draw.md）＋匯出／匯入機檢；行為驗收是實機 cjk-check，不在 CI
  bash hook/ddraw/build.sh >/dev/null
  bash hook/ddraw/check.sh
  # 圖集字表涵蓋全部譯文（6.4 實機：缺字畫成「A」）
  bash hook/ddraw/atlas-cover.sh
  # 截圖 BMP 檔頭（支線 C）：主機 gcc＋ffmpeg 往返
  bash hook/ddraw/bmp.test.sh
  # 中文斷行規則（5.3）與訊息佇列中英間距（票 H）：主機 gcc 對 wrap.h／space.h 逐條比對
  bash hook/ddraw/wrap.test.sh
  bash hook/ddraw/space.test.sh
  bash hook/ddraw/speed.test.sh
  bash hook/ddraw/menu.test.sh
  bash hook/ddraw/guide.test.sh
  bash hook/ddraw/nsf.test.sh
  bash hook/ddraw/mappos8.test.sh
}
tr_tests() {
  # 譯文全表機檢（stage5 5.2）＋四種注入必紅；原文從玩家自有 PATCH.001 現解
  bash scripts/check-translations.test.sh
  # 譯文像素寬與明示換行數（stage5 5.3）＋四種注入必紅
  bash scripts/check-fit.test.sh
  # 日誌欄寬 430px 對 MG 截圖校準（票 O）：9 條被切必紅、1232 必綠、縮短後必綠
  bash scripts/journal-fit.test.sh
}
# 字串庫譯文機檢（stage6 6.3：描述 DBS、MSG；translations/lib.tsv）＋注入必紅；原文從玩家自有 DATA.SLF 現解
lib_tests() { bash scripts/check-lib.test.sh; }
# 名稱 DBS codec（票 M／N）：往返、欄尾保留、超長拒、兩條建置路相同；不在 CI 時步驟 4 爛了沒人知（稽核 Acodec F1）
zhpatch_tests() { bash scripts/zhpatch.test.sh; bash scripts/dbsname.test.sh; }
names=(go hook tr lib zhpatch)
for n in "${names[@]}"; do lane "$n" "${n}_tests" & done
wait
rc=0
for n in "${names[@]}"; do
  echo "=== lane $n rc=$(cat "$L/$n.rc")"; cat "$L/$n.log"
  [ "$(cat "$L/$n.rc")" = 0 ] || rc=1
done
exit $rc
