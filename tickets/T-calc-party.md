# T calc party：創角 Pareto 前緣（goals/stage10.md B3 第一版）

驗收: bash scripts/party.test.sh
規模: 中（codex gpt-6-astra medium）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔；不改 scripts/party.test.sh；不寫入遊戲目錄；不執行遊戲；不下載任何東西；筆記不貼組語或反編譯碼；不把原版 bytes、從 exe 讀出的表寫進 repo（含測試 fixture 與 golden）
白名單: internal/calc/party.go, internal/calc/party_test.go, tools/calc/main.go, tools/calc/main_test.go, notes/calc.md:229
預期紅: party
執行器: codex

## 背景
創角規則已驗證，見 `notes/re/growth.md:11`–`:17`、`:21`；現有 `internal/calc/build.go` 的 `PlanAttrs` 讀同一套表（`exe.go` 的讀表函式可重用）。

## 要做的
1. `calc party -exe <Wiz8.exe> -metric <m1[,m2…]>`：枚舉 種族 0..10 × 職業 0..14 × 性別 0..1（職業 2 只允許性別 1；B<0 的組合不可選）。
   - 種族初值 R、職業門檻 M 從 exe 讀；D=Σmax(0,M−R)，B=60−D；Apre=max(R,M)。
   - 自由點 x：Σx≤B，0≤x[i]≤min(K,100−Apre[i])，K=min(B,max(3,ceil(B/3)))。
   - 指標：`attr:i`（Astart[i]，越大越好）、`l100:i`（1+ceil(max(0,100−Astart[i])/3)，越小越好）。i=0..6。
   - 只對指標用到的屬性配點即可（其餘 x=0）。
2. 輸出 TSV 到 stdout：第一列表頭，含欄位 `race class gender attrs` 加上每個指標名（原字串，如 `attr:0`、`l100:6`）；每列一個 Pareto 前緣上的點（指標值互不支配），同一指標值向量只印一列（任一達成者）。`attrs` 印 7 個 Astart 以逗號分隔。
3. `-metric` 缺省或不合法：非零退出、stderr 說明。
4. Go 測試 `TestParty*`：用合成的小 R/M 表（不得用 exe 數值）驗前緣、支配、職業 2 性別限制、B<0 排除、K 上限、100 上限。
5. `notes/calc.md:229`：用法、規則出處（引 growth.md 行號）、目前未納入的開局項（技能、法術、性別差、近戰指標）列為後續。

## 規格疑慮
寫在 `notes/calc.md:229` 末段 `## 規格疑慮`，並在最終回覆列出。
