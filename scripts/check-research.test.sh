#!/usr/bin/env bash
# check-research.sh 自測：好報告 PASS、各種壞報告 FAIL。
set -uo pipefail
here=$(dirname "$0"); t=$(mktemp -d); trap 'rm -rf "$t"' EXIT; fail=0
expect() { # <預期 0|1> <名稱> <內容>
  printf '%s\n' "$3" >"$t/$2.md"
  out=$(bash "$here/check-research.sh" "$t/$2.md"); rc=$?
  [ "$rc" = "$1" ] || { echo "FAIL $2 (rc=$rc, want $1)"; fail=1; return; }
  # 紅的時候要有 impl-run 認得的 `FAIL research:<名>` 行
  [ "$rc" = 0 ] || grep -qx "FAIL research:$2" <<<"$out" || { echo "FAIL $2 (缺 FAIL 行)"; fail=1; return; }
  echo "ok   $2"
}
expect 0 good   $'# R\n## 發現\n- 檔頭 40 bytes [已驗證] https://example.org/a\n- offset 表 [推測] tools/x.go:12\n## 其他\n- 不檢查這行'
expect 1 nosrc  $'## 發現\n- 檔頭 40 bytes [已驗證]'
expect 1 notag  $'## 發現\n- 檔頭 40 bytes https://example.org/a'
expect 1 nosec  $'## 摘要\n- x [已驗證] https://example.org'
expect 1 empty  $'## 發現\n\n## 下一段'
exit $fail
