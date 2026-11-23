# R12 研究：Wizardry 8 物品中與歷代共通者的日文／中文既有譯名

驗收: bash scripts/check-research.sh notes/research/2026-10-08-R12-classic-item-names.md
規模: 中（研究，一份報告）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單以外的檔；不寫入遊戲目錄；不執行遊戲或任何下載物；不送 email 或真實姓名；報告不抄原作說明文字，只列名稱
白名單: notes/research/2026-10-08-R12-classic-item-names.md
預期紅: research:2026-10-08-R12-classic-item-names
執行器: muse

## 背景
Wizardry 8 中文化。物品名目前多半「專名留英文、類別譯中文」（例：Cane of Corpus→「Corpus 手杖」）。MG 指出部分物品是歷代經典武器，日文版早有通行譯名，例：Cane of Corpus＝「骸の杖」（MG 確定）。要找出 Wiz8 物品中有歷代既有譯名的那些，之後 cc 據此定中文譯名。
- Wiz8 英文物品名清單：`build/items-en.tsv`（id、英文名、目前中文；gitignored，本機材料，不要抄進報告全表）。

## 問題（每條結論附來源 URL 與三態標記）
1. 從清單挑出在 Wizardry 1–7、外傳、日本製作作品出現過的物品（如 Muramasa Blade、Cane of Corpus、Excalibur、Blade Cusinart 類）。列：英文名、出現作品。
2. 每件的日文官方譯名（作品名＋平台；PS／SS／PC-98／GB 版本不同時都列）。以官方攻略本、手冊、遊戲畫面為優先，粉絲 wiki 次之並標明。
3. 「骸の杖」找到出處（哪一作、哪個平台）。
4. 有無官方或通行的中文（台灣／香港）譯名（如舊代理版、雜誌攻略）。沒有就寫 [未取得]。
5. Wiz8 本身是否有日文版；有的話其物品名。

## 報告格式
`notes/research/2026-10-08-R12-classic-item-names.md`，含 `## 發現` 段，每條 `- ` 結論附來源與 `[已驗證]`／`[推測]`／`[未取得]`；另附一張表（英文｜日文｜作品／平台｜來源）。另含 `## 規格疑慮` 段（沒有就寫「無」）。
