#!/usr/bin/env bash
# N5 (GOAL-wiz8calc.md): ranged per-round table → build/n5/ranged.tsv (human L28, Prio 4 6 5 0, skills 7/8/17=100, monster 564, difficulty 2＝Expert).
set -euo pipefail
cd "$(dirname "$0")/.."
out=build/n5
mkdir -p "$out"
go build -o "$out/calc" ./tools/calc
printf 'class\tmain\toff\tswings\thit\tdmg_per_hit\tinstakill_trigger\tdamage\tinstakill\tinstakill8\tpoison_pool\n' > "$out/ranged.tsv.part"
# instakill8: mean round.instakill over the melee kill metric's 8 monsters (i9-party.py:76); 564 Dark Savant alone is immune.
k8() {
  for mon in 564 114 184 171 129 158 120 192; do
    printf '{"Race":0,"Class":%d,"Level":28,"Prio":[4,6,5,0],"Skills":{"7":100,"8":100,"17":100},"Main":{"Item":%d},"Off":{"Item":%d},"Monster":%d,"Difficulty":2}\n' "$1" "$2" "$3" "$mon" > "$out/k8.json"
    nice -n 19 "$out/calc" fight -exe build/Wiz8.exe -dbs build/data-x/DATABASES "$out/k8.json" | awk -F'\t' '$1=="round.instakill"{print $2}'
  done | awk '{s+=$1; n++} END{if (n != 8) exit 1; printf "%f", s/n}'
}
# class main off: 3=遊俠 8=工匠
while read -r c m o; do
  j="$out/c$c-$m-$o.json"
  printf '{"Race":0,"Class":%d,"Level":28,"Prio":[4,6,5,0],"Skills":{"7":100,"8":100,"17":100},"Main":{"Item":%d},"Off":{"Item":%d},"Monster":564,"Difficulty":2}\n' "$c" "$m" "$o" > "$j"
  nice -n 19 "$out/calc" fight -exe build/Wiz8.exe -dbs build/data-x/DATABASES "$j" |
    awk -F'\t' -v c="$c" -v m="$m" -v o="$o" -v k8="$(k8 "$c" "$m" "$o")" '{v[$1]=$2} END{printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",c,m,o,v["round.swings"],v["main.hit"],v["main.dmg_per_hit"],v["main.instakill_trigger"],v["round.damage"],v["round.instakill"],k8,v["round.poison_pool"]}'
done >> "$out/ranged.tsv.part" <<'T'
3 771 131
3 739 139
3 119 131
3 771 130
3 771 128
3 771 129
3 739 765
3 739 131
8 610 131
8 610 139
T
mv "$out/ranged.tsv.part" "$out/ranged.tsv"
cat "$out/ranged.tsv"
