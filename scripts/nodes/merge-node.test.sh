#!/usr/bin/env bash
# merge-node 只改清單那一列：51e920 與 51e760 在 awk 裡都是科學記號（數值上同為 inf），比較要用字串。
set -uo pipefail
here=$(cd "$(dirname "$0")/../.." && pwd); t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
mkdir -p "$t/r/scripts/nodes" "$t/r/tickets/nodes" "$t/r/notes/re/formula-graph.d" "$t/wiz8-mod-N51e920"
cp "$here/scripts/nodes/merge-node.sh" "$t/r/scripts/nodes/"
printf '51e920\tF\n' > "$t/r/tickets/nodes/51e920.list"
printf 'addr\tclass\tdepth\tnode\tcalc\tevidence\n' > "$t/r/notes/re/formula-graph.tsv"
printf '51e760\t公式\tD0\ta\t\tx\n51e920\t公式\tD0\tb\t\tx\n' > "$t/r/notes/re/formula-graph.d/b00.tsv"
bash "$t/r/scripts/nodes/merge-node.sh" 51e920 || { echo "FAIL merge-node 執行失敗"; exit 1; }
got=$(cut -f1,3,5 "$t/r/notes/re/formula-graph.d/b00.tsv" | tr '\t\n' ' |')
[ "$got" = "51e760 D0 |51e920 D4 F|" ] && echo "PASS merge-node" || { echo "FAIL merge-node 改錯列：$got"; exit 1; }
# 多行清單（循環節點群）：每一行都標、清單外不動
printf '51e760\tF1\n51e800\tF2\n' > "$t/r/tickets/nodes/51e760.list"; mkdir -p "$t/wiz8-mod-N51e760"
printf '51e760\t公式\tD0\ta\t\tx\n51e800\t公式\tD0\tc\t\tx\n51e900\t公式\tD0\td\t\tx\n' > "$t/r/notes/re/formula-graph.d/b00.tsv"
bash "$t/r/scripts/nodes/merge-node.sh" 51e760 || { echo "FAIL merge-node 多行執行失敗"; exit 1; }
got=$(cut -f1,3,5 "$t/r/notes/re/formula-graph.d/b00.tsv" | tr '\t\n' ' |')
[ "$got" = "51e760 D4 F1|51e800 D4 F2|51e900 D0 |" ] && echo "PASS merge-node 多行" || { echo "FAIL merge-node 多行：$got"; exit 1; }
