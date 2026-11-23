# 排名試算沿革（2026-10-08..13）

> 歷史：L28 排名、基準對齊、H3／I4 試算、I9 全隊最佳化。現行結論見 notes/party.md。
> 2026-11-01 由 9 份原檔逐行併入（標題降一級、內文不改）；原檔見 tag `wiz8-notes-pre-merge`（85393087）。

目錄：
- 第 20 行起：28 級戰士：種族 × 武器每回合期望傷害（階段 8.3）（原 `notes/research/2026-10-08-L28-fighter.md`）
- 第 104 行起：L28 近戰全職業排名：單持、雙持、徒手、Lightning Strikes（原 `notes/research/2026-10-08-L28-ranking.md`）
- 第 366 行起：妖精可穿的防具＋Rod of Sprites 為何排前面（原 `notes/research/2026-10-09-faerie-armor.md`）
- 第 385 行起：jeffludwig.com Wizardry 8 頁面與原版資料交叉核對（原 `notes/research/2026-10-08-jeffludwig-crosscheck.md`）
- 第 404 行起：I10 基準對齊：10-08 兩隊以 calc rank kill 重算＋8 怪 Expert（原 `notes/research/2026-10-12-baseline-align.md`）
- 第 610 行起：H3：calc rank 試算 v0（近戰，L1／10／28）（原 `notes/research/2026-10-12-H3-rank-trial.md`）
- 第 682 行起：I4：calc rank 試算 v1（近戰，雙持 vs 單持）（原 `notes/research/2026-10-12-I4-rank-v1.md`）
- 第 827 行起：I9b 全隊最佳化：Expert 8 怪（原 `notes/research/2026-10-13-I9-party-expert.md`）
- 第 921 行起：I9c 全隊最佳化：Expert 重篩武器池（原 `notes/research/2026-10-13-I9c-pool-expert.md`）


---
<!-- 原檔 notes/research/2026-10-08-L28-fighter.md -->
## 28 級戰士：種族 × 武器每回合期望傷害（階段 8.3）

日期：2026-10-08。MG：「計算一個 28 級的戰士有多少升級點跟技能點，然後選什麼種族會拿到更多可以分配的點數，那個時候就知道拿這些武器的強度」（`goals/stage8.md:3`）。
數字由 `calc build`（票 R，c14dc84；雙持 K 經 grok 稽核 cXan F1 更正於 9eff044）對玩家自有 Wiz8.exe／ITEMS.DBS 算出；產生腳本與原始輸出在 scratchpad（不入版）。本筆記只列計算結果與少量物品引用值，不轉錄原版表。

### 設定

- 職業 Fighter，L=28，P=PMult×L=4×28=112。屬性優先序 STR、DEX、SPD、VIT，其餘依索引序（`goals/stage8.md:23`）。
- 技能（手動 258 點，不含創角自動技能與練技，**全部是下限**）：
  - 雙持：劍 100、Close Combat 100、Dual Weapons 29、匕首 29。
  - 單持：該武器技能 100、Close Combat 100、Power Strike 58。
- 模式：
  - 「一般」＝主手依武器模式遮罩自選，0x3 為 mode 0／1 各 5/9、4/9，0x2 必 mode 1（套 B15 的抽法）。
  - 「狂暴」＝主手 mode 3。
  - 副手一律依自己遮罩（匕首必 mode 1），不承襲主手。
- 目標：
  - 假想怪：AC 0 → DefHit=DefPen=10，四屬性和 200，吸收 0。
  - 後期怪：MONSTERS.DBS id 564「Dark Savant# 38 cc2」，L38，AC −22 → DefHit 32（DefPen 以 DefHit 近似），Attr[1..4] 和 410，吸收 0。
- 未建模：負重、裝備屬性加成、F28 特殊物品分支、背刺、Kill／KO／Hex 等特效、練技成長、遠程。

### 發現

- **L28 時種族幾乎不影響近戰輸出**：照上述優先序，11 族的 STR、DEX、SPD 在 28 級都到 100（Faerie 也是：創角預算只剩 10，但升級的 162 點足以補滿）。對假想怪，命中門檻封頂 95%，11 族數值完全相同；對後期怪只差 Δ 項（INT+DEX+SPD+SEN 對目標）的 ±1–3 點 ARhit，差距 ≤4%。來源：internal/calc/build_test.go:27（PlanAttrs golden）、notes/re/growth.md:20 [已驗證：calc golden＋3 格手算]
- **例外 Mook：SPD 只到 97**，v=99 差 1 點未到第三揮門檻 100，揮擊上限從 3 掉到 2；這讓 Giant's Sword（Mook 專用）在單持武器中最弱。要改善得把 SPD 排到 DEX 前，或靠裝備加 SPD（未建模）。來源：notes/re/formulas.md:33（F09 v≥100）[已驗證：手算 Mook 屬性 100,50,25,95,100,97,55、v=99]
- **雙持對後期怪最弱**：雙持旗標讓 K 的分母從 20 變 35（F07，只加回 Dual Weapons 29），主手 K 74、M −27，對 Dark Savant 的 ARhit 只剩 69／59（mode 0／1）；Canezou 副手 ARhit 9（M=−37、HitMod 減半、mode 1 −10），每回合只貢獻約 3.5。雙持對 Dark Savant 113–120，比任何單持武器都低；對假想怪也略輸單持 Ivory Blade／Excaliber。來源：internal/calc/build_test.go:96（雙持 golden）、notes/re/formulas.md:29（F07）[已驗證：手算 Hobbit LS+Canezou 119.71]
- **Light Sword 單持在兩種目標、兩種模式都最高**（256／513、256／432–437），其次 Excaliber、Ivory Blade：它的骰子（5d8）比 Ivory Blade 好，Init +7 讓 v 到 130，HitMod +6，對 Dark Savant 命中仍封頂 95%。Excaliber 與 Dread Spear 種族遮罩排除 Faerie，Giant's Sword 只限 Mook（ITEMS.DBS +0x78）。來源：notes/re/combat-2.md:96 [已驗證：cc 讀 ITEMS.DBS 種族遮罩]
- **雙持要到主手第三次攻擊才划算**：主手 a≥100 需要 K+M≥88（Hobbit；F07 雙持 K、M 都隨 Dual Weapons 漲）。DW 與匕首都練高時（DW 90＋匕首 100，或 DW 100＋匕首 80），主手攻擊從 2 次跳 3 次，對 Dark Savant 一般攻擊從約 170 跳到 250–262，略勝 Light Sword 單持 256；但狂暴仍是單持高（437 對 352–369），因為副手不承襲 mode 3。門檻之前每 10 點 DW 只多約 6。來源：表 C、notes/re/formulas.md:29（F07）[已驗證：ref8 獨立重算 DW100/匕首100＝262.44、DW90/匕首80＝169.21、DW100/匕首80＝255.48，與 calc 相同]
- **狂暴對假想怪剛好翻倍，對後期怪單持只多 40–55%、雙持約 20%**：mode 3 多一組骰，但命中／穿透 −30，對 DefHit 32 的目標從 95% 封頂跌下來。來源：notes/re/formulas.md:55（F20）、notes/re/backstab.md:39（B17）[已驗證：calc golden TestBuildBerserkL28]
- 「一般」攻擊的主手模式是否真的走遮罩自選（Q+5=−1），本表沿用副手的 B15 抽法；戰士狂暴是否對所有武器開放（feature 0x14 的 UI 條件）asm 未查；MG 實玩經驗：「戰士依據我的經驗可以裝備的武器都可以狂暴（主手一定有」（2026-10-08）。來源：notes/re/backstab.md:61 [推測：主手一般模式無實測支撐；狂暴可用性只有 MG 實玩經驗，asm 未覆核]
- 單持的 Ivory Blade、Fang、Excaliber 若是單手武器，可再加副手，本表未算：這三欄是下限。單／雙手欄位已定位為物品 +0x41（bit2 雙手、bit3 可副手），雙持排名見 notes/research/ranking-history.md:122。來源：notes/re/combat-2.md:89 [推測：本表未改算三件的雙持組合]

### 表 A：對 AC 0 假想怪（一般／狂暴，每回合期望傷害）

| 種族 | LS+Canezou | LS+Stiletto | Light Sword | Ivory Blade | Fang | Excaliber | Giant's Sword | Dread Spear |
|---|---|---|---|---|---|---|---|---|
| Human、Elf、Dwarf、Gnome、Hobbit、Lizardman、Dracon、Felpurr、Rawulf | 208／379 | 193／364 | 256／513 | 217／433 | 194／388 | 235／470 | — | 194／388 |
| Faerie | 208／379 | 193／364 | 256／513 | 217／433 | 194／388 | — | — | — |
| Mook | 208／379 | 193／364 | 256／513 | 217／433 | 194／388 | 235／470 | 180／359 | 194／388 |

### 表 B：對 Dark Savant（id 564，DefHit 32）

| 種族 | LS+Canezou | LS+Stiletto | Light Sword | Ivory Blade | Fang | Excaliber | Giant's Sword | Dread Spear |
|---|---|---|---|---|---|---|---|---|
| Human | 118／140 | 116／139 | 256／432 | 212／319 | 175／245 | 230／346 | — | 163／245 |
| Elf | 115／136 | 114／135 | 256／427 | 211／315 | 172／241 | 229／342 | — | 161／241 |
| Dwarf | 113／132 | 112／131 | 256／421 | 210／310 | 170／237 | 227／337 | — | 159／237 |
| Gnome | 118／140 | 116／139 | 256／432 | 212／319 | 175／245 | 230／346 | — | 163／245 |
| Hobbit | 120／144 | 118／142 | 256／437 | 213／324 | 177／249 | 231／351 | — | 165／249 |
| Faerie | 120／144 | 118／142 | 256／437 | 213／324 | 177／249 | — | — | — |
| Lizardman | 113／132 | 112／131 | 256／421 | 210／310 | 170／237 | 227／337 | — | 159／237 |
| Dracon | 115／136 | 114／135 | 256／427 | 211／315 | 172／241 | 229／342 | — | 161／241 |
| Felpurr | 120／144 | 118／142 | 256／437 | 213／324 | 177／249 | 231／351 | — | 165／249 |
| Rawulf | 113／132 | 112／131 | 256／421 | 210／310 | 170／237 | 227／337 | — | 159／237 |
| Mook | 120／144 | 118／142 | 256／437 | 213／324 | 177／249 | 231／351 | 164／231 | 165／249 |

標記：全部格子的計算路徑由 calc golden 測試覆蓋 `[已驗證：calc golden]`，輸入假設（一般模式抽法、狂暴可用、技能下限）`[推測]`；「—」＝種族遮罩不允許。

### 表 C：雙持 Dual Weapons／匕首技能掃描（Hobbit × LS+Canezou × Dark Savant，一般／狂暴）

不受 258 點預算限制（代表練技成長後）；劍 100、Close Combat 100 固定。對照：Light Sword 單持 256／437。

| 匕首＼DW | 29 | 40 | 50 | 60 | 70 | 80 | 90 | 100 |
|---|---|---|---|---|---|---|---|---|
| 29 | 120／144 | 126／156 | 133／168 | 138／176 | 144／188 | 149／196 | 155／208 | 161／220 |
| 80 | 134／167 | 139／175 | 145／187 | 151／199 | 158／211 | 164／223 | 169／231 | 255／357 |
| 100 | 139／176 | 144／184 | 151／196 | 157／208 | 164／220 | 168／228 | 254／352 | 262／369 |

Lizardman 同形狀，一般低 7–9、狂暴低 12–17；Stiletto 副手比 Canezou 低 1–4；對假想怪只有跳階那格會變（208→294／379→550）。

### 手算核對（3 格，cc 以獨立 Python 參考實作重算，與 `calc build` 逐值相同）

1. Human × Excaliber × 假想怪 × 一般＝235.125：屬性 100,52,45,95,100,100,45。K=⌊10·⌊2(125+200)/3⌋/20⌋=108；a=tz((100+112+108)/3)=106 → 3 次攻擊；v=tz((100−50+112+108)/3)=90 → 揮擊上限 2，期望 1.5，共 4.5 揮；HitMod J=4+5+5+0+(1+⌊58/20⌋)=17；BAR=60+tz((50+216+112)/3)=186；Δ=297−200=97；ARhit=186+85−50+9=230 → 95%；mode 1 為 220／240 同樣封頂。單擊平均 (5+5×4.5)×200%=55，4.5×0.95×55=235.125。
2. Hobbit × LS+Canezou × Dark Savant × 一般＝119.71：屬性 100,53,30,99,100,100,50。主手 u=⌊2(125+200)/3⌋+⌊29/2⌋+29=259，K=⌊2590/35⌋=74，M=−10−⌊71/4⌋=−27，a=86（2 次攻擊）、v=109（期望 2 揮）→ 4 揮，BAR 154、J 17；mode 0 ARhit 69 → 4×0.69×45=124.2，mode 1 為 59／79 → 106.2；副手 K 57、a 77、v 74 → 3 揮，ARhit 9／ARpen 29 → 3×0.09×13=3.51。5/9·127.71＋4/9·109.71＝119.71。
3. Mook × Giant's Sword × Dark Savant × 狂暴＝230.58：屬性 100,50,25,95,100,97,55；a=106、v=99 → 4.5 揮；J=2+5+5+0+3=15；ARhit=186+75−30−160+tz((302−410)/10)=61 → 61%；兩組骰平均 2×(6+3×5)=42，×200%＝84；4.5×0.61×84＝230.58。

### 規格疑慮

- 初版（47ac19a）雙持 K 漏了 F07 的 +D 與 d=35（票 R 規格錯，grok 稽核 cXan F1 抓到），雙持欄偏高 20–25%；本版已重算，單持欄不受影響。

- DefPen 以 DefHit 近似；C05 的 runtime 修正未納入，後期怪的命中率可能偏高或偏低。
- 表 A／B 的技能配置只是一種；DW 與匕首的影響見表 C（只掃 Hobbit、Lizardman）。Mook 的 SPD 97 會讓門檻變 K+M≥90，未掃。

---
<!-- 原檔 notes/research/2026-10-08-L28-ranking.md -->
## L28 近戰全職業排名：單持、雙持、徒手、Lightning Strikes

日期：2026-10-08。MG：「考慮雙持武器、空手戰鬥等技能，重新做完整排序前十名，如果還沒有數據的就先確認數據是什麼」。
接續 `notes/research/ranking-history.md:20`（戰士表）。數字由 cc 的獨立 Python 參考實作（scratchpad，不入版）對玩家自有 Wiz8.exe／ITEMS.DBS 算出；單持部分與 `calc build` 逐格比對 16,558 格，除兩件同名物品（Stinger id 748／749）的比對鍵相撞外全部相同。本筆記只列結論與少量引用值，不轉錄原版表。

### 設定

- L=28，屬性優先序 STR、DEX、SPD、VIT（同戰士表），15 職業 × 11 族 × 職業／種族遮罩允許的近戰物品（技能 0–5）。
- 技能：
  - 單持：武器 100、Close Combat 100、Power Strike 58（可用的職業）。
  - 雙持「高」：主副手武器技能各 100、Dual Weapons 100、CC 100、PS 58。
  - 雙持「低」：主手 100、CC 100、DW 29、副手技能 29（戰士 258 點預算那套）。
  - 徒手：Martial Arts 100、CC 100、PS 58。
- 目標同戰士表：AC 0 假想怪，與 Dark Savant（MONSTERS.DBS id 564，DefHit 32）。
- 未建模：KO／Hex 等特效（即死見「即死」節）、Ninja feature 5／0x15、目標狀態的倍傷（F22）、第二物品槽的 double-vs、裝備屬性加成、負重、遠程（C10）、同一件物品主副手各一把。

### 發現

- **物品 +0x41 是手部欄位**：bit2＝雙手武器（網頁標雙手的 40 件全中），bit3＝可當副手（網頁標 2x 的 31 件全中；另 2 件木桿類 Wooden Dowel、Notched Shaft bit3 有設但網頁沒標）。村正 0x41=0（只能主手、也不是雙手）。https://jeffludwig.com/wizardry8/weaponlist.php、build/wiz8.asm:385209（裝備端 test 0x4） [已驗證但斷言較寬：與網頁逐件比對，asm 消費點只看到 bit2 的 test，bit3 的消費點未逐條讀]
- **Lightning Strikes 只對劍**：條件 C+0x114e==0 的 C+0x114e 是主手武器技能（手部重建把物品 +0x46 寫進去，空手寫 14），0＝劍；本筆記原假設所有近戰武器都成立，已改成只算劍（Light Sword、村正都是劍，前十名不變）。build/wiz8.asm:319553、build/wiz8.asm:319556 [已驗證：cc 讀 asm；C+0xa15 意義仍未取得]
- **Lightning Strikes（feature 0x10，Samurai）**：每回合 R(100) 小於 scale(L,12)，L>20 時為 12%（L20 為 11%，C02）；觸發後主手揮擊數加 3 或 4，R(11)+職業等級 ≥20 為 +4，所以 L≥20 必為 +4；每回合最多一次，用後清旗標。期望每回合多 0.48 揮，Samurai LS 單持 256→277。build/wiz8.asm:312014、build/wiz8.asm:312034、build/wiz8.asm:432040、build/wiz8.asm:432062、notes/re/combat-2.md:13 [已驗證：cc 讀 asm；另需 C+0x114e 與 C+0xa15 為 0，兩欄意義未取得，假設一般近戰武器成立為推測]
- **徒手只有 Ninja、Monk 能用**（職業×技能可用矩陣 Martial Arts 欄）。空手給技能 14，B 取 Close Combat；兩手都空時兩手都出拳，**不設雙持旗標**（M=0、d=20、無 W_other 項）；Init 加 ⌊K/10⌋；模式遮罩 0x20，MA≥5 為 0x60（模式 5、6）。骰表在 0x6172f0、以 ⌊W/11⌋ 索引，主手固定項再 +2；W=125 時主手 6+4d4、副手 4+4d4。build/wiz8.asm:319550、build/wiz8.asm:319608、build/wiz8.asm:319633、build/wiz8.asm:319919、build/wiz8.asm:390474 [已驗證：cc 讀 asm 與 exe 骰表；模式 5／6 只改命中穿透、不改傷害為推測]
- **雙持要 DW 與副手技能都練高才划算**：「高」配置下雙持佔前六名（LS＋可副手物品 301–318），「低」配置最好只有 192（Samurai LS＋Enchanted Wakizashi），比單持低。build/wiz8.asm:319700（F07 雙持旗標）、notes/re/formulas.md:29 [已驗證：參考實作與 calc 單持逐格相同；雙持格由同一實作算，未逐格對 calc]
- **盜賊背刺每一下都有，不必躲藏**：feature 9（盜賊）、當手是射程 0 的劍或匕首就設背刺旗標，逐手算（B01）。效果是命中 +10，傷害多擲 b 組骰，L≥21 時 b＝0／1／2／3 機率 15／60／20／5%（期望 1.15 組，同一個 R(100) 比三次）；mode 4、7、8 不吃。等於比狂暴多 0.15 組骰又沒有 −30，所以盜賊雙持 LS 對 Dark Savant 424，比原第一名 Ninja 321 高三成；擊殺數仍輸戰士狂暴（盜賊等級係數 PMult 3、戰士系 4，L28 的 P 少 28 點；擊殺數落差的拆解未做）。build/wiz8.asm:437475、build/wiz8.asm:437499、notes/re/backstab.md:43、notes/re/formulas.md:57 [已驗證：cc 讀 asm；B03 一般目標有效性未取得，實機是否每下都背刺未驗]
- **double-vs：武器 +0x61 等於怪物 +0xcb 類別時多一組骰，任何模式都算**：只對怪物目標、每手各查一次。原版只有 7 件：Beastslayer（類別 16，L20 以上 8 隻）、Woodsman's Axe（17）、Anointed Flail（20）、Demonsbane（21，L20 以上 11 隻）、LS 與 Mystery Ray（22，2 隻）、Dragon Slayer（24，Bela）、Cobaltine Powerglove（26，沒有怪）。類別名稱未取得。第二物品槽（+0x74 指的物品）也會再查一次，未建模。build/wiz8.asm:437557 [已驗證：cc 讀 asm 與 ITEMS／MONSTERS.DBS；類別名稱為推測]
- **徒手輸武器約 25–35%**：Monk 211（Faerie 最高）、Ninja 132，對 Dark Savant；Monk 主手 3 攻×2 揮、副手 2 攻×2 揮，副手命中與傷害減半（c=2）。徒手 10% KO（STR 100）未建模。build/wiz8.asm:319955 [已驗證：cc 手算 Monk Faerie 主手 6×0.81×32、副手 4×0.66×21，與實作相符]
- **村正排不進前十**：Samurai 單持 215 對 LS 277。骰 4+3d8（平均 17.5）對 5d8（22.5），Init／HitMod +5／+4 對 +7／+6，揮擊數相同；村正長處是即死欄 15（LS 10），即死機率與怪物抗性（F25）未建模。notes/re/combat-2.md:151 [已驗證：參考實作；即死換算未取得]
- **法術不和近戰同表**：L28、PL7、Power Cast 100 單體，對無抗性目標 Cerebral Hemorrhage／Concussion 220、Boiling Blood／Dehydrate 176、Crush 141；對 L22 一般怪（id 411 Mad Marten，六領域抗性 100）只剩 55／44／28；對 Dark Savant（抗性 98–130）≈0。L20 以上怪的抗性中位數 95–130，照 C14 公式大約只剩兩成到四分之一傷害。notes/re/combat-2.md:63、notes/re/combat-2.md:71 [推測：C14 對抗性 100 的解讀未經實機驗證；職業可學法術的對照未取得]
- **即死機率＝發動骰×抵抗骰，每一下有傷害的揮擊擲一次**：攻擊傷害大於 0 才進特效整理；第 5 槽 e＝武器 +0x55（村正 15、LS 10）＋Critical Strike 技能 K 的 ⌊K/25⌋（餘數再擲，期望 K/25）；R(100)<e 才發動，再進抵抗。build/wiz8.asm:433559、build/wiz8.asm:433649、build/wiz8.asm:438973 [已驗證：cc 讀 asm；手部 +0x1152＝16 代表近戰為推測，Critical Strike 只有 Samurai／Ninja／Monk 能學（職業×技能矩陣）]
- **抵抗骰**：x＝3(怪物等級−S)＋Divine 抗性（領域 5）＋24，先乘難度、再 clamp 到 5..95 得 a，R(100)<a 為抵抗；S＝L＋min(L,15)，L28 為 43；24 是狀態抵抗 bonus 表第 18 項（死亡），exe 抽出。build/wiz8.asm:459256、build/wiz8.asm:459275 [已驗證：cc 讀 asm＋抽表]
- **難度直接改即死通過率**：怪物端難度函式在怪物 +0x16＝1 時，難度 0 乘 0.6、難度 2 乘 1.4（+0x16＝2 時對調，Normal 一律不變），取整後才 clamp。難度 0／2 是 Novice／Expert：同一全域值在玩家命中端是 ×1.4／×0.6（F18），與網頁「新手 ×1.4、專家 ×0.6」同向。L20 以上非免疫 41 隻的通過率中位數：Novice 63%、Normal 38%、Expert 14%（四分位 57–67／28–45／5–23）；Expert 下約四分之一的怪只剩保底 5%。Expert 另把玩家命中乘 0.6，有傷害的揮擊變少，即死嘗試次數也跟著少。build/wiz8.asm:474798、notes/re/formulas.md:51、notes/research/2026-10-08-R11-formulas.md:14 [已驗證：cc 讀 asm；怪物 +0x16＝1 代表敵方為推測，未追寫入點]
- **硬免疫靠怪物 +0x1be 旗標**：+0x1be 非零時死亡直接拒絕；另一張類別免疫表只有 3 列、都不含死亡。L20 以上 56 隻怪有 15 隻免疫，Dark Savant 免疫（所以對 Savant 的排名不變）；其餘 41 隻的通過率四分位 5%／29%／38%／45%／72%。build/wiz8.asm:458721、build/wiz8.asm:458731 [已驗證：cc 讀 asm 與 MONSTERS.DBS]
- **加進即死後村正勝 Light Sword**：對 Mad Marten（id 411，L22，Divine 100 → 通過 39%，HP 期望 555），Samurai 單持村正每回合即死 37%、期望 2.04 回合殺完，LS 28.5%、2.23 回合；但村正每回合傷害 215 對 277。通過率 5% 的高等怪（等級 ≥35、抗性 100）即死幾乎無效，LS 回到前面。build/wiz8.asm:439167 [推測：每回合揮擊數取期望值、即死次數用 Poisson 近似、傷害取平均不擲骰]
- **遠程（C10）尚不能排**：據子 agent 讀 asm，命中是真的投射物碰撞（ARhit 只決定瞄準偏移，偏移與距離成正比），命中後另擲 1+R(100)≤ARpen 才穿甲、沒有 ≤5 保底。命中率要目標碰撞半徑，未取得。MG：距離要 DEX、穿透要 STR（Power Strike）、次數要 SPD。notes/re/combat-2.md:44 [推測：子 agent 結論，cc 未逐條覆核；碰撞半徑、多發射擊、弓箭骰合併、PS／STR 是否適用遠程均未取得]
- **種族主要靠專屬裝備進榜，屬性差距很小**：L28 時 STR／DEX／SPD 各族都到 100（Mook SPD 97），同一組武器換種族只差 ±0.01 擊殺（Δ 項）；拉開差距的是種族遮罩（ITEMS.DBS +0x78）：Faerie 專屬 Rod of Sprites、Sprite's Dagger、Cane of Corpus，Lizardman／Dracon 專屬 Serpent's Tooth（侍、女武神第四、領主第六），Chieftan's Sceptre（領主第三）不是人類專屬：種族遮罩 0xffdf 只排除 Faerie、職業遮罩 0xb＝戰士／領主／遊俠，表中標「人」是同分判定（2026-10-09 更正）。蜥蜴人 INT 25、SEN 30 讓 Δ 最低。notes/research/ranking-history.md:45 [已驗證：參考實作 8 職業 × 11 族全展開，遮罩由實作讀 ITEMS.DBS；Rod of Sprites 等的遮罩值 cc 未逐件目視]
- **Power Strike 開練時間各族不同，未建模**：PS 要 STR 裸值 100 才生效（F14）；STR 優先配點下戰士 Lizardman L9、Dracon L10、Human／Hobbit L12、Faerie L15 到 100，盜賊 Lizardman L12、Faerie L21。PS 每 20 點 = HitMod +1 = ARhit +5，大於 Hobbit 對 Lizardman 的 Δ 優勢（INT+DEX+SPD+SEN 303 對 279，ARhit +2–3）。本筆記各族一律 PS 58，早到 STR 100 的種族被低估；PS 練滿後各族相同，差距主要在中期。notes/re/formulas.md:43 [推測：到 100 的等級由 growth.md 配點實算，每級技能點與練技速度未取得，「蜥蜴人反超」未模擬]
- **只有 mode 7 的鞭鏈類不能雙持**：MG 實玩：九尾鞭、吸血鬼之鏈「都是主手可以裝盾牌，但不能再雙持第二把副手武器」（2026-10-08）。物品 +0x41 手部欄位分不出來（Cat'o 與 Fang 都是 0x00），共同點是模式遮罩只有 0x80（mode 7）：Bullwhip、Cat'o Nine Tails、Chain、Vampire Chain，本筆記把這 4 件從雙持主手排除（1,188 組）。notes/research/ranking-history.md:32 [推測：以 mode 7 遮罩為判準只有 MG 兩件的實玩支撐，裝備端 asm 未查]
- **女武神的長柄加成已算進去，Dread Spear 仍不划算**：女武神主技能是 Polearm，有效技能 100→125，K 116 對劍類 100；但 Dread Spear 單持只 0.287（171），同條件 The Avenger 0.427、LS 0.417、Excaliber 0.354，差在骰子與 Init／HitMod。internal/calc/build_test.go:50（主技能加成） [已驗證：參考實作；長柄另有的職業特效未查]
- **Rapax 女王任務的祭祀刀＝Canezou Dagger**：MG 實機截圖（傷害 3–14、命中 +2、先攻 −1、邪咒 5%、即死 1%，主副手皆可）。與 ITEMS.DBS 讀值一致（HitMod +2、Init −1）。internal/calc/build_test.go:136 [已驗證：MG 截圖對 cc 讀值]

### 前十名（對 Dark Savant、一般攻擊，每回合期望傷害；含盜賊背刺）

| # | 職業 × 武器 | 對 Savant | 種族 |
|---|---|---|---|
| 1 | Rogue 雙持高 LS＋Sprite's Dagger | 424 | Faerie（他族 Thieves／Canezou Dagger 約 414–420） |
| 2 | Rogue 單持 LS | 368 | 任何族（Dwarf 等 360–364） |
| 3 | Ninja 雙持高 Cane of Corpus＋Rod of Sprites | 321 | Faerie |
| 4 | Samurai 雙持高 LS＋Rod of Sprites | 318 | Faerie；他族 LS＋Enchanted Wakizashi 306 |
| 5 | Lord 雙持高 LS＋Diamond Eyes | 312 | Human |
| 6 | Fighter 雙持高 LS＋Diamond Eyes | 308 | Human；狂暴 459 |
| 7 | Ranger 雙持高 LS＋Diamond Eyes | 305 | Human |
| 8 | Valkyrie 雙持高 LS＋Diamond Eyes | 301 | Human |
| 9 | Ninja 單持 Cane of Corpus | 292 | Faerie |
| 10 | Samurai 單持 LS | 277 | 含 Lightning Strikes |

徒手：Monk 211、Ninja 132。狂暴只有 Fighter 有（feature 0x14）。Dark Savant 即死免疫，本表不受即死影響；Savant 類別 27，沒有任何武器對它 double-vs。初版（e7ff753）漏了背刺，盜賊整列缺席，MG 指出「前十名沒有盜賊應該是有問題」後補上。

### 含即死：對 Mad Marten（id 411，L22，HP 期望 555，即死通過 39%）

期望回合數＝⌈555／每回合傷害⌉ 回合內每回合以即死率提早結束。**本表算於背刺建模之前，盜賊未列入**；排名以下節為準。Critical Strike 技能 100（Ninja 主技能 125）。

| # | 職業 × 武器 | 期望回合 | 每回合即死 | 每回合傷害 | 種族 |
|---|---|---|---|---|---|
| 1 | Ninja 雙持高 Cane of Corpus＋Ninjato | 1.59 | 41% | 323 | Faerie |
| 2 | Ninja 單持 Cane of Corpus | 1.64 | 36% | 305 | Faerie |
| 3 | Ninja 雙持高 LS＋Ninjato | 1.65 | 35% | 280 | Hobbit |
| 4 | Samurai 雙持高 LS＋Enchanted Wakizashi | 1.67 | 33% | 309 | Hobbit |
| 5 | Lord 雙持高 LS＋Chieftan's Sceptre | 1.79 | 21% | 291 | Human |
| 6 | Fighter／Ranger 雙持高 LS＋Chieftan's Sceptre | 1.79 | 21% | 290 | Hobbit |
| 7 | Valkyrie 雙持高 LS＋Canezou Dagger | 1.80 | 21% | 280 | Human |
| 8 | Samurai 雙持高 Ivory Blade＋Rod of Sprites | 1.87 | 13% | 280 | Faerie |
| 9 | Samurai 雙持高 村正＋Enchanted Wakizashi | 1.95 | 41% | 248 | Hobbit |
| 10 | Samurai 單持村正 | 2.04 | 37% | 215 | Human |

對照：Samurai 單持 LS 2.23（28.5%、277）、Fighter 單持 LS 2.44（20%、256）；徒手 Monk 2.66（12%、230）。

標記：計算路徑 `[已驗證：參考實作，單持與 calc golden 一致]`；輸入假設（一般模式抽法、技能配置、未建模項）`[推測]`。

### 最多擊殺前十名（L20 以上 56 隻怪平均，每回合擊殺數）

每回合擊殺數＝1／期望回合數，期望回合數同上節（含即死、吸收 +0xcf、背刺、double-vs），56 隻等權平均；免疫的 15 隻即死通過率 0。Normal 難度；候選＝Marten 期望回合前 500＋Savant 傷害前 200＋全部單持與徒手＋盜賊全部組合，共 17,880 組。

| # | 職業 × 武器（一般攻擊） | 每回合擊殺 | 不算即死 | 種族 |
|---|---|---|---|---|
| 1 | Rogue 雙持高 LS＋Thieves Dagger | 0.623 | 0.605 | Faerie |
| 2 | Ninja 雙持高 Cane of Corpus＋Rod of Sprites | 0.549 | 0.471 | Faerie |
| 3 | Samurai 雙持高 LS＋Rod of Sprites | 0.516 | 0.451 | Faerie |
| 4 | Rogue 單持 LS | 0.511 | 0.490 | Faerie |
| 5 | Ninja 雙持高 LS＋Sai | 0.503 | 0.419 | Hobbit |
| 6 | Ninja 單持 Cane of Corpus | 0.494 | 0.411 | Faerie |
| 7 | Samurai 雙持高 村正＋Rod of Sprites | 0.483 | 0.386 | Faerie |
| 8 | Lord 雙持高 LS＋Diamond Eyes | 0.481 | 0.442 | Human |
| 9 | Fighter 雙持高 LS＋Diamond Eyes | 0.476 | 0.438 | Human |
| 10 | Ranger／Valkyrie 雙持高 LS＋Diamond Eyes | 0.473 | 0.435 | Human |

對照：Samurai 單持 LS 0.458（不算即死 0.395）、單持村正 0.424（0.323）。
難度（背刺建模前算的）：Ninja 第 2 名 Novice 0.602／Expert 0.505，Samurai 單持村正 0.493／0.369、單持 LS 0.494／0.414（只改即死通過率，玩家命中的難度倍率未一起套）。

**含戰士狂暴的合併前十**（MG 2026-10-08：「戰士依據我的經驗可以裝備的武器都可以狂暴（主手一定有」；副手不承襲 mode 3，照舊）：

| # | 職業 × 武器 | 每回合擊殺 | 不算即死 | 種族 |
|---|---|---|---|---|
| 1 | Fighter 雙持高狂暴 LS＋Diamond Eyes | 0.751 | 0.738 | Hobbit |
| 2 | Fighter 單持狂暴 The Avenger | 0.699 | 0.685 | Hobbit |
| 3 | Fighter 單持狂暴 LS | 0.672 | 0.655 | Hobbit |
| 4 | Fighter 雙持高狂暴 Ivory Blade＋Diamond Eyes | 0.631 | 0.631 | Human |
| 5 | Rogue 雙持高 LS＋Thieves Dagger | 0.623 | 0.605 | Faerie |
| 6 | Fighter 單持狂暴 Excaliber | 0.593 | 0.585 | Hobbit |
| 7 | Fighter 雙持高狂暴 Fang＋Diamond Eyes | 0.571 | 0.548 | Hobbit |
| 8 | Fighter 單持狂暴 Ivory Blade | 0.568 | 0.568 | Human |
| 9 | Ninja 雙持高 Cane of Corpus＋Rod of Sprites | 0.549 | 0.471 | Faerie |
| 10 | Fighter 雙持高狂暴 Coil of The Serpent＋Diamond Eyes | 0.548 | 0.548 | Felpurr |

（原第 9 名 Cat'o Nine Tails＋Diamond Eyes 0.551 已刪：九尾鞭不能雙持，見發現段。）

戰士沒有 Critical Strike，即死只靠武器 +0x55，所以算不算即死只差 0–0.023（即死欄為 0 的武器完全不變）。

標記：計算路徑 `[已驗證：參考實作，單持與 calc golden 一致；背刺與 double-vs 分支未與 calc 對過]`；等權平均、Poisson 近似、取平均不擲骰、溢出傷害不帶到下一隻 `[推測]`；56 隻含 5 筆 Dark Savant 變體與 NPC，權重偏向 Savant。

### 非戰士／盜賊職業：種族 × 武器前十（L20 以上 56 隻怪平均每回合擊殺，括號為對 Dark Savant 每回合傷害）

MG 2026-10-08：「非戰賊的職業 僧、侍、女、忍、領、遊、詩、工 這些種族跟武器搭配的前十名」。8 職業 × 11 族 × 全部可裝組合（不經候選篩選），共 131,623 組；設定與上節相同、Normal。每列一個武器組合，種族欄列出擊殺數並列最高的所有種族（沒寫＝所有能裝的種族同分），「/n」是能裝的種族數。種族：人 Human、精 Elf、矮 Dwarf、侏 Gnome、哈 Hobbit、妖 Faerie、蜥 Lizardman、龍 Dracon、貓 Felpurr、狼 Rawulf、毛 Mook。

**忍者**：全部 Cane of Corpus 主手、只有妖精。

| # | 副手 | 擊殺（傷害） |
|---|---|---|
| 1 | Rod of Sprites | 0.549（321） |
| 2 | Sai | 0.538（303） |
| 3 | Enchanted Wakizashi | 0.536（307） |
| 4 | Ninjato | 0.532（302） |
| 5–8 | Wand of Winds／Nunchaka／Ymmu's Paw／Sprite's Dagger | 0.531（304–307） |
| 9 | Thunder Stick | 0.528（303） |
| 10 | Canezou Dagger | 0.522（303） |

**侍、女武神、領主、遊俠**：全部雙持高、主手 Light Sword，表內是副手。

| # | 侍 | 女武神 | 領主 | 遊俠 |
|---|---|---|---|---|
| 1 | Rod of Sprites 妖/1 0.516（318） | Diamond Eyes 人哈/9 0.473（301） | Diamond Eyes 人精侏哈貓/9 0.481（312） | Diamond Eyes 哈貓/9 0.476（307） |
| 2 | Enchanted Wakizashi /9 0.510 | Stiletto 人哈/9 0.468 | Stiletto /9 0.469 | Stiletto 哈貓/10 0.469 |
| 3 | Sprite's Dagger 妖/1 0.502 | Wand of Winds 人哈狼/9 0.462 | Chieftan's Sceptre 人/9 0.469 | Chieftan's Sceptre 人哈貓毛/9 0.465 |
| 4 | Serpent's Tooth 蜥龍/2 0.502 | Serpent's Tooth 蜥龍/2 0.462 | Wand of Winds /9 0.463 | Wand of Winds 人哈妖貓毛/10 0.463 |
| 5 | Poison Dagger /9 0.496 | Ymmu's Paw 人矮哈蜥龍狼/9 0.459 | Ymmu's Paw /9 0.463 | Ymmu's Paw 哈貓/10 0.463 |
| 6 | Wand of Winds /9 0.496 | Thunder Stick 人矮哈蜥龍狼/9 0.459 | Serpent's Tooth 蜥龍/2 0.463 | Rod of Sprites 妖/1 0.463 |
| 7 | Wakizashi /9 0.496 | Canezou Dagger 人矮哈蜥龍狼/9 0.457 | Thunder Stick /9 0.460 | Sprite's Dagger 妖/1 0.463 |
| 8 | Winterwand /9 0.496 | Winterwand 人矮哈蜥龍狼/9 0.452 | Canezou Dagger /9 0.457 | Thunder Stick 人哈妖貓毛/10 0.460 |
| 9 | Ymmu's Paw /9 0.496 | Poison Dagger 人矮哈蜥龍狼/9 0.451 | Winterwand /9 0.456 | Canezou Dagger 人哈妖貓毛/10 0.457 |
| 10 | FlameQuencher Wand /9 0.496 | Poignard 人矮哈蜥龍狼/9 0.439 | Poison Dagger /9 0.453 | Winterwand 哈貓/10 0.456 |

**吟遊詩人、工匠**：兩職業數字相同（PMult 都是 3、技能設定相同），全部雙持高 Light Sword＋副手：Rod of Sprites 妖 0.368（197）、Stiletto 0.364、Sprite's Dagger 妖 0.351、Wand of Winds 0.347、Ymmu's Paw 0.346、Canezou Dagger 0.341、Thunder Stick 0.339、Winterwand 0.336、Poison Dagger 0.335、Serpent's Tooth 龍 0.335。並列種族兩職業略有不同（詩人可當的種族 10、工匠 9）。

**僧侶**：唯一不走 Light Sword。

| # | 組合 | 種族 | 擊殺（傷害） |
|---|---|---|---|
| 1 | 單持 Staff of Doom | 人精侏哈妖貓狼毛/8 | 0.390（188） |
| 2 | 徒手 Martial Arts | 人精侏哈妖貓狼毛/8 | 0.380（211） |
| 3 | 單持 Zatoichi Bo | 人精侏哈妖貓狼/8 | 0.378（219） |
| 4 | 雙持高 Coil of The Serpent＋Rod of Sprites | 妖/1 | 0.374（207） |
| 5 | 雙持高 Coil of The Serpent＋Wand of Winds | 妖/8 | 0.356（191） |
| 6 | 雙持高 Coil of The Serpent＋Nunchaka | 精妖貓/8 | 0.354（190–192） |
| 7 | 雙持高 Coil of The Serpent＋Ymmu's Paw | 妖/8 | 0.354（190） |
| 8 | 雙持高 Coil of The Serpent＋Thunder Stick | 妖/8 | 0.352（188） |
| 9 | 雙持高 Coil of The Serpent＋Winterwand | 妖/8 | 0.349（185） |
| 10 | 單持 The Mauler | 人精侏哈貓/7 | 0.348（187–190） |

（九尾鞭、吸血鬼之鏈的雙持列已刪，改排。）

上節「種族」欄是候選篩選後剩下的種族中最高者，不代表其他種族較差；本節全展開後同分種族全列。
標記：同上節 `[已驗證：參考實作]`／`[推測]`；另未建模法術、詩人的歌、工匠道具、遠程、PS 開練時間差。

### 考慮裝備唯一性（每件物品只有一把）

MG 2026-10-08：「主手神兵例如 fang、ivory blade、cane of corpus、光劍... 等都是唯一取得…副手的鑽眼、rapax 女王任務祭司刀都只有一把…考慮到裝備的唯一性，重新排隊前十名」。物品是否唯一 ITEMS.DBS 沒有欄位，**一律當一把**（可買多把的 Stiletto、Wand 等會讓後段偏低）。資料：Fighter、Lord、Valkyrie、Ranger、Samurai、Ninja、Monk、Rogue、Gadgeteer、Bard 皆 11 族全展開；同分以三位小數判定；鞭鏈類不雙持。

**表一：依序搶（不限職業）**：擊殺最高的先拿，用過的物品後面不能再用。

| # | 職業 | 武器 | 種族 | 擊殺（傷害） |
|---|---|---|---|---|
| 1 | Fighter 狂暴 | LS＋Diamond Eyes | 妖精以外同分/10 | 0.751（440–459） |
| 2 | Fighter 狂暴 | The Avenger 單持 | 各族同分 | 0.699（370–388） |
| 3 | Fighter 狂暴 | Ivory Blade＋Rod of Sprites | 妖 | 0.610（337） |
| 4 | Fighter 狂暴 | Excaliber 單持 | 哈貓毛/10 | 0.593（351） |
| 5 | Fighter 狂暴 | Fang＋Chieftan's Sceptre | 貓/10 | 0.553（255） |
| 6 | Fighter 狂暴 | The Mauler 單持 | 人精侏哈龍貓毛/10 | 0.541（276–285） |
| 7 | Ninja | Cane of Corpus＋Sai | 妖 | 0.538（303） |
| 8 | Fighter 狂暴 | Staff of Doom 單持 | 哈妖貓毛/11 | 0.531（290） |
| 9 | Fighter 狂暴 | Coil of The Serpent＋Wand of Winds | 人侏哈妖貓毛/11 | 0.514（258–263） |
| 10 | Fighter 狂暴 | Dragon Slayer 單持 | 哈貓毛/10 | 0.484（249） |

11 名後：Fighter 狂暴 Cat'o Nine Tails 單持 0.480、Samurai 村正＋Enchanted Wakizashi 0.473。

**表二：每職業一人，總擊殺最大**：精確解（分支定界，候選先以「較好且所用物品不被他職業需要」支配剪枝；「#」是該組合在本職業不考慮唯一性時的名次）。

10 職業全上（總和 4.387；依序搶為 4.207）：

| 職業 | 武器 | 種族 | 擊殺（傷害） | # |
|---|---|---|---|---|
| Fighter 狂暴 | The Avenger 單持 | 各族同分 | 0.699（370–388） | 13 |
| Rogue | LS＋Thieves Dagger | 各族同分 | 0.623（396–420） | 1 |
| Ninja | Cane of Corpus＋Sai | 妖 | 0.538（303） | 2 |
| Samurai | 村正＋Enchanted Wakizashi | 哈貓毛/9 | 0.473（241） | 4 |
| Ranger | Ivory Blade＋Diamond Eyes | 人哈貓毛/9 | 0.404（252–255） | 26 |
| Monk | Staff of Doom 單持 | 各族同分 | 0.390（182–188） | 1 |
| Lord | The Mauler＋Chieftan's Sceptre | 人貓/9 | 0.378（223） | 33 |
| Valkyrie | Fang＋Wand of Winds | 人哈/9 | 0.378（177） | 27 |
| Bard | Coil of The Serpent＋Rod of Sprites | 妖 | 0.269（133） | 23 |
| Gadgeteer | Bloodlust＋Stiletto | 哈貓/8 | 0.234（65） | 31 |

6 人隊（10 職業選 6，總和 3.155）：Fighter 狂暴 The Avenger 0.699、Rogue LS＋Thieves Dagger 0.623、Ninja Cane of Corpus＋Rod of Sprites 0.549（妖）、Samurai 村正＋Enchanted Wakizashi 0.473、Lord Fang＋Diamond Eyes 0.421（人精侏哈貓）、Monk Staff of Doom 0.390。

- 最佳解把 LS 給盜賊、戰士改拿 The Avenger：戰士狂暴換武器只掉 0.05，盜賊沒有 LS 會掉 0.19（背刺只吃劍／匕首）。
- 村正與 Staff of Doom 幾乎只有侍、僧侶要，所以他們拿第一順位也不擋人。

標記：最佳化在候選集合內為精確 `[已驗證：分支定界跑完]`；候選每職業先取前 400 組再剪枝 `[推測：400 以外不會進最佳解，未證]`；物品一律唯一、只算近戰、未建模項同上節 `[推測]`。

**MG 存檔的實際限制與組隊討論（2026-10-09）**

MG 最滿意的 legit 存檔（手動練技，S/L 多次拿到 Light Sword）沒有光盾、Excaliber、The Avenger。The Avenger、Excaliber、Staff of Doom 都是雙手武器（物品 +0x41＝0x04），上表「單持」本就沒配副手。

去掉 The Avenger、Excaliber 的 6 人隊（總和 3.066）：Rogue LS＋Thieves Dagger .623、Fighter 狂暴 Ivory Blade＋Sprite's Dagger .610（妖）、Ninja Cane＋Rod of Sprites .549（妖）、Samurai 村正＋Enchanted Wakizashi .473、Lord Fang＋Diamond Eyes .421、Monk Staff of Doom .390。

同上再強制含 Bard、Gadgeteer（總和 2.785）：Fighter 狂暴 Ivory Blade＋Diamond Eyes .631、Rogue LS＋TD .623、Ninja Cane＋Sai .538、Samurai 村正＋EW .473、Bard Coil＋Rod of Sprites .269（妖）、Gadgeteer Fang＋Stiletto .251（貓）。近戰掉 0.28，換來的道具施法本表不計。

8 人隊形狀（MG：NPC 一般選 Vi＝牧師法術、RFS-81＝僧侶，android 無靈能法術）：自創 6 人強制含 Bard、Gadgeteer、排除 Monk，再排一次與上一組完全相同（2.785）；RFS-81 拿走 Staff of Doom 也不變（最佳解本來沒用它）。NPC 本身未建模、不排名。

Rod of Sprites 運氣好不只一把（MG）：同形狀以兩把重排（總和 2.796），只有忍者副手從 Sai（.538）換成 Rod（.549），其餘不變；無限把時 4 人拿 Rod、總和 2.812（侍改 村正＋Rod .483）。

MG 實玩經驗（擊殺率模型都沒算）：
- 混合職業：領主學牧師法術、侍學法師法術、忍者學鍊金術師法術、僧侶學靈能者法術。最高法術等級不比純施法職業低，但一般 Lv 6 以後才開始學，而破關很難到 L28，所以高級法術要更晚才學到。legit 玩法常在 Arnika 練功。
- 詩人、工匠不學法術書：樂器、工具各對應四系中的部分法術，施法花的是回復很快的 stamina。女性角色另有很多性別限定、可回復 stamina 的護身符。「這代沒有詩人跟工匠很吃虧」。

### 規格疑慮

- 即死表只對一隻代表怪；怪物 HP 越高、等級越低，即死越重要，反之傷害越重要。
- 雙持格與徒手格只由參考實作算，`calc build` 還不支援徒手；要入 calc 需開票。
- 怪物抗性 100 是否真的擋掉約 75% 法術傷害，影響法術與近戰能不能比，需要實機看一次。

### 更正（2026-10-12，calc fight 重算）
> 上文是 2026-10-08 依社群公式手算；以下用 `calc fight`（D4 節點串：004ee220 0053d870 00541c00 00541ec0 005420b0 00543270）對 Dark Savant（564）重算前十名，Difficulty 1、Mode 0、站位固定相鄰前排（數值規劃範圍）。情境檔 `notes/calc-qa/fight-l28-01..10.json`，重跑：`calc fight -exe build/Wiz8.exe -dbs <DBS> <情境>`；`scripts/fight.test.sh` 斷言每檔可跑。上文不改。

| # | 組合 | calc 每回合傷害 | 上文 | calc main.hit |
|---|---|---|---|---|
| 1 | Rogue 妖精 LS＋Sprite's Dagger | 337.0 | 424 | 0.775 |
| 2 | Rogue LS | 302.6 | 368 | 0.782 |
| 3 | Ninja 妖精 Cane＋Rod | 275.3 | 321 | 0.745 |
| 4 | Samurai 妖精 LS＋Rod | 269.2 | 318 | 0.866 |
| 5 | Lord LS＋DE | 276.4 | 312 | 0.859 |
| 6 | Fighter LS＋DE | 268.1 | 308 | 0.838 |
| 7 | Ranger LS＋DE | 263.4 | 305 | 0.824 |
| 8 | Valkyrie LS＋DE | 257.9 | 301 | 0.810 |
| 9 | Ninja 妖精 Cane | 251.1 | 292 | 0.775 |
| 10 | Samurai LS | 248.0 | 277（含 Lightning Strikes） | 0.919 |

- 全部比上文低 11–21%；排名前兩名不變；Lord（#5，276.4）升到第 3，高於 Ninja 妖精（275.3）與 Samurai 妖精（269.2）。
- 主因是命中：上文隱含約 0.95，calc 0.75–0.92。盜賊 ARhit 96 仍只 0.78，疑為逐部位穿透門檻 [推測：無實測支撐]。
- 未建模：Lightning Strikes（#10 差額一部分）、combatSkill 0x11（投擲／遠程）、T74≠−1。無副手時 53d870 的 item7 以 2（空物品記錄）代 −1，是否等價未證實。細節 `notes/calc.md:15`。
- 待 MG 過目（stage12 A1）。

---
<!-- 原檔 notes/research/2026-10-09-faerie-armor.md -->
## 妖精可穿的防具＋Rod of Sprites 為何排前面

日期：2026-10-09。MG：「Rod of Sprites 真的有這麼好哦？爲什麼，記得就是 hex 50% 的特效。但 faerie 的防具非常少，你可以記錄這篇文章，然後下個過濾幫我找哪些防具 faerie 可以用嗎？」
數字由 cc 讀玩家自有 ITEMS.DBS 算出（scratchpad，不入版）；只列結論與少量物品引用值，不轉錄原版表或網頁表。

### 參考文章

- Jeff Ludwig「Wizardry 8 Mod → Armor List」：https://jeffludwig.com/wizardry8mod/armorlist.php （2026-10-09 取得）。
  注意：這是 **Christian Coder's Mod** 的防具表，數值與限制可能和原版不同；MG 玩的是原版，本筆記的篩選以原版 ITEMS.DBS 為準，文章只拿來對欄位。

### 發現

- ITEMS.DBS 記錄（0x10d bytes）：+0x3e 是部位（5 盾、6 身、7 腿、8 頭、9 手、10 腳、11 飾品、12 披風），+0x62 是 AC（有號），+0x78 是種族遮罩（bit5＝Faerie）。對照文章：Faerie Cap AC +4、「Faerie only」與 DBS 的 0x62=4、0x78=0x20 一致。來源：https://jeffludwig.com/wizardry8mod/armorlist.php、notes/re/combat-2.md:96（種族遮罩 +0x78）[已驗證：cc 逐筆讀 Cloth Shirt／Plate Mail (U)／Faerie Cap 等 14 件對部位與 AC]
- 原版妖精能穿的防具（飾品除外）：盾 0/15，身 4/44，腿 3/33，頭 11/33，手 1/9，腳 3/15，披風 6/12。來源：notes/research/ranking-history.md:370（cc 讀 ITEMS.DBS）[已驗證：cc 讀 ITEMS.DBS 種族遮罩 bit5]
- 各部位最好的（不分職業）：身 Zynaryx Plate +6（妖精專屬，忍者／僧侶不可）、腿 Doll's Armor (L) +5（同）、頭 Canezou Helm +7、手 Doll's Gloves +3（妖精專屬，忍僧靈法不可）、腳 Spiked Boots +5、披風 Cloak of Many Colors +3；盾一件都沒有。飾品 76 件中 74 件可用，不是瓶頸。來源：notes/research/ranking-history.md:370 [已驗證：cc 讀 ITEMS.DBS AC +0x62 與職業遮罩 +0x76]
- Rod of Sprites（妖精專屬、全職業、技能 5）排得高靠骰子與速度，不靠特效：4d4+2（平均 12），Init +5，HitMod +2；Sprite's Dagger 是 1d12+2（平均 8.5）、Init +4、HitMod +1。模型的特效欄（+0x55）對 Rod 是 0，MG 記得的 hex 50% 沒有算進去，所以實際只會更好。來源：notes/research/ranking-history.md:122 [推測：hex 特效在 DBS 的位置與機率未查，模型未建模]

---
<!-- 原檔 notes/research/2026-10-08-jeffludwig-crosscheck.md -->
## jeffludwig.com Wizardry 8 頁面與原版資料交叉核對

日期：2026-10-08。MG 提供四頁網址；cc 以 curl 抓取（存 scratchpad，不入版），與同版 `Wiz8.exe`、`ITEMS.DBS`、`SPELLTABLES.DBS` 比對。本筆記只記比對結論與欄位意義，不轉錄網頁表格。
注意：該站同時放 Christian Coder's Mod 資料；races 頁標「not valid for version 2.0 of the mod」。下列「相符」證明所比對的欄位與原版一致，不證明整頁都是原版值。

### 發現

- 物品效果陣列 `I+0x50` 的 16 欄意義：欄 0 Sleep、1 Paralyze、2 Poison、3 Hex、4 Disease、5 Kill、6 KO、7 Blind、12 Stamina，以網頁「特效 n%」與 DBS 該欄值相等投票，142 件同名武器入比對；欄 5 Kill 34 票、欄 6 KO 31 票、欄 3 Hex 6 票、欄 1 Paralyze 10 票、欄 2 Poison 9 票 https://jeffludwig.com/wizardry8/weaponlist.php [已驗證但斷言較寬：值相等投票，欄 0／4／7／12 票數 2–5，欄 14／15 Nauseate／Insane 票數不足未定]
- 武器 +0x48 i8 即網頁 Init 欄：*Light* *Sword* +7、Excaliber −5、Dread Spear +2 三例相符；Dread Spear 欄 3=50 對網頁 Hex 50%、Giant's Sword 欄 6=30 對 KO 30% https://jeffludwig.com/wizardry8/weaponlist.php [已驗證：cc 讀 ITEMS.DBS 三例]
- 種族初值表 `Wiz8.exe@0x214cf0`（11×7 u32）與網頁 11 族最低屬性逐值相符，證實 exe 列序 Human、Elf、Dwarf、Gnome、Hobbit、Faerie、Lizardman、Dracon、Felpurr、Rawulf、Mook（growth.md 的手冊順序映射）；Lizardman「Slower Magic Power recovery」、Faerie「Faster」對 abilities.md 的 0x1b ×0.75／Faerie ×1.25 https://jeffludwig.com/wizardry8/races.php [已驗證：cc 人工逐列比對 11 族 7 值]
- 職業門檻表 `Wiz8.exe@0x214e24`（15×7）與網頁 Required Attributes：13 職業逐值相符；Gadgeteer 網頁 Int 50、exe 55，Psionic 網頁 Sen 55、exe 60，以 exe 為準。XP 基數 1000／1200／1400／1600 分組與 growth.md cc 抽查一致 https://jeffludwig.com/wizardry8/professions.php [已驗證：cc 人工比對 15 列]
- 法術：網頁 95 個與 SPELLTABLES 同名的法術，SP 成本＝+0x14a、傷害範圍＝骰 `Z+N .. Z+N·S`（+0x152）全數相符，0 差異 https://jeffludwig.com/wizardry8/spelllist.php [已驗證：cc 腳本比對 95 筆]

### 規格疑慮

- 網頁未標版本的欄位（例如物品需求職業、種族抗性數值）不當原版事實引用；要用時先對 exe／DBS。

---
<!-- 原檔 notes/research/2026-10-12-baseline-align.md -->
## I10 基準對齊：10-08 兩隊以 calc rank kill 重算＋8 怪 Expert

日期：2026-10-12。goals/stage16.md I10。MG：「基準要對後續才有用，我們要先對齊基準」；「基準好了後，8 怪直接跑 Expert 即可」；「僧侶配座頭市之棍比較強」。
對象：notes/research/ranking-history.md:325–327 兩隊成員，加 Monk Zatoichi Bo（81）、Valkyrie Staff of Doom（86）。

### 設定

- `calc rank`（build/i10/calc，含 840656b），Metric `kill`（每怪 1／期望回合數，含 round.instakill，多怪等權平均）。
- L28；屬性優先序 STR DEX SPD VIT（取 prio 列 `0,4,5,3`）；不開 Haste；AttrMin 0；SkillPrio 空、Train 直接設定技能：武器 0–5、CC 16、DW 18、CS 19 為 100，PS 34 為 58。
  - 職業不可用的技能 Train 不給（見「發現」）：CS 只有 Samurai／Ninja／Monk。Ninja 主技能是 CS，有效值 125（`EffectiveSkill`，exe 主技能表 0x6161dc）；MG：「算 cs 的時候，ninja 要當作 125 這就是它的特長」，模型已如此。
- 第一部分：L20 以上 56 隻怪（同 10-08 的集合），Normal（Difficulty 1）。第二部分：8 怪逐隻，Expert（Difficulty 2）。
- Fighter 用 berserk，其餘 attack（一般攻擊模式分布）。
- 種族：每格一族（10-08 註明同分或指定的那族）：Rogue／Lord／Monk／Valkyrie Human、Fighter Faerie（Sprite's Dagger 妖精專屬）或 Human（Diamond Eyes 遮罩排除妖精）、Ninja／Bard Faerie、Samurai Hobbit、Gadgeteer Felpurr。
- 情境檔與輸出在 scratchpad（不入版）；重跑：照上列設定寫 rank 範本，`calc rank -exe build/Wiz8.exe -dbs <DBS> <範本>`。

### 發現：rank Train 給了職業不可用的技能（已修，840656b）

- `PlanSkills` 回傳全 41 格技能，rank 的 Train 迴圈對每一格加點，沒查職業×技能可用表（exe 0x615840）。Fighter／Rogue／Lord／Valkyrie／Bard／Gadgeteer 因此帶了 CS 100，即死被高估。
- 修正：Train 跳過 Avail＝0 的技能。TestRankKill 的 oracle 抄了同一迴圈，一併修；fixture 改挑自帶即死欄的劍。四段注入：拿掉守衛 → `FAIL TestRankKill`（got .3145 want .2992）→ 副本還原 → PASS。
- 修正前後（Normal 56 怪）：Fighter 妖精 .567→.552、Lord .427→.398、Bard .278→.253、Gadgeteer .271→.242、Valkyrie SoD .383→.362；CS 合法的 Samurai／Ninja／Monk 不受影響。對 Dark Savant（即死免疫）的傷害修正前後逐格相同（7 格比對）。
- 受影響範圍：Metric kill（I8，79f64a0）且 Train 含 19 的 rank 輸出，非 CS 職業擊殺偏高。damage 指標不受影響（上列 7 格 Savant 傷害逐格相同），所以 I4 的傷害排名不變 [推測：I4 範本未逐格重跑，依據是 CS 只進即死欄]。

### 一、基準（Normal、56 隻 L20+）

歸因欄：「Savant 比」＝calc 對 Dark Savant（564，即死免疫）每回合傷害 ÷ 10-08 括號值，只反映傷害公式差（10-12 更正段：主因命中，10-08 隱含約 0.95，calc 0.75–0.92）。擊殺差小於傷害差的部分，來源可能是即死模型（10-08 Poisson 近似、取平均不擲骰）與擊殺的回合取整，拆不開，寫未取得。

| 成員 | 10-08 擊殺 | calc 擊殺 | 差 | 10-08 不算即死 | calc Savant 傷害／10-08 | Savant 比 | 歸因 |
|---|---|---|---|---|---|---|---|
| Rogue LS＋Thieves Dagger | .623 | .615 | −.008（−1.3%） | .605 | 344.5／414–420 | .82–.83 | 傷害 −17%；擊殺只 −1.3%，其餘（即死欄 LS 10／TD 5 與取整）未取得 |
| Fighter 狂暴 Ivory Blade＋Sprite's Dagger（妖） | .610 | .552 | −.058（−9.5%） | .610 | 241.3／337 | .72 | 兩邊即死皆 0（武器即死欄 0、Fighter 無 CS）；差全在傷害公式 |
| Fighter 狂暴 Ivory Blade＋Diamond Eyes（人） | .631 | .577 | −.054（−8.5%） | .631 | 249.4／未取得 | 未取得 | 同上：差全在傷害公式 |
| Ninja Cane of Corpus＋Rod of Sprites（妖） | .549 | .542 | −.007（−1.3%） | .471 | 277.2／321 | .86 | 傷害 −14%；擊殺只 −1.3%，其餘由即死補回，比例未取得 |
| Ninja Cane of Corpus＋Sai（妖） | .538 | .523 | −.015（−2.8%） | 未取得 | 259.9／303 | .86 | 同上 |
| Samurai 村正＋Enchanted Wakizashi（哈） | .473 | .451 | −.022（−4.6%） | 未取得 | 201.0／241 | .83 | 傷害 −17%；其餘未取得 |
| Lord Fang＋Diamond Eyes（人） | .421 | .398 | −.023（−5.5%） | 未取得 | 185.4／未取得 | 未取得 | 未取得（10-08 無本組傷害、無不算即死值） |
| Monk Staff of Doom（人） | .390 | .385 | −.005（−1.4%） | 未取得 | 147.9／182–188 | .79–.81 | 傷害 −20%；其餘未取得 |
| Monk Zatoichi Bo（人） | .378 | .370 | −.008（−2.2%） | 未取得 | 190.6／219 | .87 | 傷害 −13%；其餘未取得 |
| Bard Coil of The Serpent＋Rod of Sprites（妖） | .269 | .253 | −.016（−5.8%） | 未取得 | 106.9／133 | .80 | 傷害 −20%；其餘未取得 |
| Gadgeteer Fang＋Stiletto（貓） | .251 | .242 | −.009（−3.6%） | 未取得 | 73.7／未取得 | 未取得 | 未取得 |
| Valkyrie Staff of Doom（人） | 無（10-08 未算） | .362 | — | — | 143.8／— | — | — |

- 10-08 → calc 普遍降 1–10%，名次不變：Rogue ≥ Fighter ＞ Ninja ＞ Samurai ＞ Lord ＞ Monk ＞ Bard ＞ Gadgeteer。
- 降最多的是 Fighter（−8.5～−9.5%）：沒有即死可補，傷害公式差（Savant 比 .72）全反映在擊殺上。有 CS 的 Ninja／Monk 降最少。
- Monk：SoD .385 仍高於 Zatoichi .370（10-08 .390／.378，同序）。

### 二、8 怪 Expert（每回合擊殺；Difficulty 2）

怪：564 Dark Savant、114 熔融焦炎獸、184 高等惡魔、171 日之巨靈、129 暴風、158 鐵木守護者、120 稜光凝視蛇、192 開悟之靈（名稱取 translations/monsters.json）。

| 成員 | 564 | 114 | 184 | 171 | 129 | 158 | 120 | 192 | 平均 |
|---|---|---|---|---|---|---|---|---|---|
| Ninja Cane＋Rod（妖） | .167 | .590 | .634 | .368 | .288 | .335 | .393 | .500 | **.409** |
| Rogue LS＋TD | .200 | .535 | .553 | .307 | .280 | .362 | .517 | .500 | **.407** |
| Ninja Cane＋Sai（妖） | .143 | .598 | .541 | .384 | .297 | .344 | .398 | .500 | .401 |
| Samurai 村正＋EW（哈） | .125 | .467 | .535 | .349 | .272 | .306 | .324 | .333 | .339 |
| Fighter 狂暴 Ivory＋Sprite's（妖）／＋DE（人） | .143 | .500 | .500 | .200 | .200 | .250 | .333 | .500 | .328 |
| Lord Fang＋DE（人） | .111 | .386 | .414 | .219 | .205 | .241 | .279 | .333 | .274 |
| Monk Zatoichi Bo（人） | .111 | .383 | .408 | .215 | .181 | .236 | .277 | .333 | .268 |
| Monk Staff of Doom（人） | .091 | .376 | .446 | .289 | .205 | .220 | .262 | .250 | .267 |
| Valkyrie Staff of Doom（人） | .083 | .347 | .400 | .245 | .173 | .201 | .247 | .250 | .243 |
| Gadgeteer Fang＋Stiletto（貓） | .045 | .207 | .248 | .148 | .108 | .135 | .142 | .167 | .150 |
| Bard Coil＋Rod（妖） | .067 | .200 | .200 | .091 | .091 | .125 | .167 | .250 | .149 |

- Fighter 兩組 8 怪逐格相同：兩組都沒有即死，傷害差不跨回合數取整門檻（每格是 1／n）。
- Expert 下 Fighter 從 Normal 第 2 掉到第 4（按職業）：命中 ×0.6 時沒有即死可補；Ninja 升到與 Rogue 並列第一。
- Monk：SoD 對高 HP 低抗的 171（日之巨靈）、184 強，Zatoichi 對 564、192 強；平均 .267 對 .268，打平。

### 三、女武神武器（人類；MG 2026-10-12：恐怖茅有 125 長柄加成也贏不了雙持嗎）

女武神主技能是長柄（主技能表 0x6161dc），長柄 raw 100 → 有效 125。設定同第一、二部分。

| 武器 | Normal 56 怪 | Expert 8 怪 |
|---|---|---|
| Dread Spear（815，雙手，9+2d7，即死欄 0） | .287 | .200 |
| Staff of Doom（86，雙手） | .362 | .243 |
| Fang＋Diamond Eyes | .384 | .266 |
| Ivory Blade＋Diamond Eyes | .390 | .255 |
| Light Sword＋Diamond Eyes | .458 | .348 |

- Dread Spear 最差：雙手沒有副手揮擊、骰平均 17、無即死；有效 125 只進命中與攻擊次數，補不回少一手（10-08 Normal 也是 .287）。
- 同拿 Fang＋DE，女武神與領主接近（Expert .266 對 .274，Normal .384 對 .398）。
- LS＋DE 的 Expert .348 高於侍 .339，但 LS 只有一把；給了女武神，盜賊沒有 LS（10-08 Normal 掉 .19，背刺只吃劍／匕首）。
- 第六人取僧不取女的理由：僧的 Zatoichi／SoD 沒人搶，女武神要跟領主搶 Fang／DE、跟盜賊搶 LS [推測：依上表推論，未跑裝備唯一的全隊最佳化]。

### 四、領主 Dual Weapons 125 做什麼（MG 2026-10-12：「領主的雙持是 125 這是他的特長，這個技能到底能幹嗎」）

領主主技能是 Dual Weapons（主技能表 0x6161dc），raw 100 → 有效 125。只在雙持時生效（F07–F10，internal/calc/build.go:163–176），不進每下傷害與即死：

1. 雙持懲罰 M＝−10−tz((100−DW)/4)，副手再 −10。
2. 合成技能 K：雙持時 u 加 DW，K＝⌊10u/35⌋。
3. M、K 進攻擊次數上限 a（主手 a≥50／100 為 2／3 下，副手 a≥75 為 2 下）與命中底值 BARraw（直接加到 ARhit）。

L28 領主（DEX、SPD 100，P＝4×28＝112，CC 與武器技能有效 100）：

| DW 有效值 | 手 | K | M | a | 次數 | BARraw |
|---|---|---|---|---|---|---|
| 0 | 主 | 71 | −35 | 82 | 2 | 149 |
| 0 | 副 | 71 | −45 | 79 | 2 | 146 |
| 100 | 主 | 100 | −10 | 100 | 3 | 177 |
| 100 | 副 | 100 | −20 | 97 | 2 | 174 |
| 125 | 主 | 107 | −4 | 105 | 3 | 184 |
| 125 | 副 | 107 | −14 | 101 | 2 | 180 |

- L28 攻擊次數已到頂（主 3、副 2），125 的收穫是命中：BARraw 主手 +7、副手 +6；主手 a 從剛好 100 變成 105，不再踩門檻。
- 低等級時 P 小、a 離門檻近，多出的 4–5 點 a 可能多一下攻擊；DW 練不高時 M 懲罰更大。
- 第一、二部分領主的值已含有效 125（`EffectiveSkill` 套到所有技能）。

標記：表內數字 `static_analysis`（照 build.go 公式手算，Python 向零截斷重算一次）；次數門檻、BARraw 的遊戲內效果未實機驗。

### 五、遊俠近戰（MG 2026-10-12：「先前是不是提到可以算遊俠？」）

設定同第一、二部分；遊俠不可用 CS、不可裝 Fang（class-mask）。本節取 24 種屬性順序中最佳一列（多數與 0,4,5,3 同值，差異 ≤ .02）。

人類：

| 武器 | Normal 56 怪 | Expert 8 怪 |
|---|---|---|
| Light Sword＋Diamond Eyes | .460 | .352 |
| Ivory Blade＋Diamond Eyes | .393 | .255 |
| Staff of Doom | .363 | .247 |

- 10-08 的 LS＋DE .476、Ivory＋DE .404 是修 rank Train 前的值（當時遊俠被誤給 CS）。

種族（LS＋DE，Expert 8 怪）：侏儒、哈比、貓人、雪人 .370；人類、Dracon .352；精靈、矮人、Rawulf .271；妖精不能裝 DE（race-mask）；蜥蜴人建角不可。

巨人之劍（778，唯一的 extended 劍，雙手）只有雪人能裝（race-mask 在 0..10 只開雪人）：雪人遊俠 Normal .259、Expert .173，約為同族 LS＋DE 的一半。MG 2026-10-12：遠程跟 SEN 有關，雪人 SEN 高，常見選雪人遊俠拿巨人之劍；不是唯一選擇。

LS＋DE 跨職業（Expert 8 怪，屬性順序 0,4,5,3）：領 .377、戰 .370、遊 .352（哈比 .370）、女 .348；侍、工不能裝 LS（class-mask）。能拿 LS 的近戰職業差 < .03，遊俠排在村正武士（.339）之上是武器差，不是職業差；換 Ivory＋DE 也一樣（遊 .393、女 .390、領 Fang＋DE .398，Normal）。

遊俠的職業特色在遠程，近戰吃不到：主技能（主技能表 0x6161dc）是技能 17 遠程（MG 2026-10-12 確認），另有遠程即死（MG 2026-10-12；機率未取得）。

未建模：遠程射擊（C10，SEN 只在這裡起作用）、extended 從後排近戰的站位價值。巨人之劍與雪人的價值主要在這兩項，本表低估。

### 六、僧：空手、Zatoichi、SoD、Mindblast Rod（MG 2026-10-13：「好，開 I11 修，但 sod 跟 zbo 都有特技哦，空手是偶爾可以打暈」）

I11（e846fa4）前 calc 漏算空手副手拳，Monk 空手 Expert 只有 .166；修後兩手數值對上 MG 面板（拳法 125：踢 20–44×3、拳 12–30×2）。以下皆 I11 後，人類 L28 代理 NPC（RFS-81 實際屬性未建模），技能 0..5、14、16、18、19 練到 100，8 怪：

| 武器 | Expert | Normal（Difficulty 1） |
|---|---|---|
| Staff of Doom | .267 | .396 |
| Zatoichi Bo | .268 | .366 |
| 空手 | .243 | .343 |
| Mindblast Rod | .128 | .202 |

Vi＋RFS-81 分配（兩人合計，Expert／Normal）：Vi SoD＋RFS-81 Zatoichi .511／.727；**Vi SoD＋RFS-81 空手 .486／.705**；RFS-81 SoD＋Vi Dread Spear .467／.683。甲隊 Monk 已拿 Zatoichi，所以 Vi 拿 SoD、RFS-81 空手；Mindblast Rod 約為空手一半，不用。

等級（空手 vs Zatoichi，Expert／Normal）：

| 等級 | 空手 | Zatoichi |
|---|---|---|
| L20 | .144／.234 | .120／.197 |
| L28 | .243／.343 | .268／.366 |
| L35 | .271／.372 | .312／.401 |
| L40 | .304／.421 | .369／.441 |
| L50 | .363／.450 | .430／.472 |

前期空手領先，L28 起 Zatoichi 反超，差距隨等級擴大；養得好的主力 Monk 拿 Zatoichi。

為什麼被反超（calc fight，怪 564 Expert，同上設定）：

| | 空手 L20 | 空手 L40 | Zatoichi L20 | Zatoichi L40 |
|---|---|---|---|---|
| 每回合揮擊（兩手合計） | 8.0 | 10.0 | 3.0 | 6.0 |
| 命中 | .31 主／.22 副 | .51 主／.42 副 | .39 | .59 |
| 每中傷害 | 32 主／24 副 | 同左 | 41 | 41 |
| 每回合傷害 | 60 | 138 | 47 | 144 |

每中傷害不隨等級變（空手骰在有效技能 125 封頂、Zatoichi 是武器骰），命中兩邊都約 +.20；差在揮擊數：L20→L40 Zatoichi 3→6（×2），空手 8→10（+25%）。Zatoichi 每中傷害較高、即死觸發較高（.09 對 .04；564 免疫即死），出手追上後就反超。等級如何換成揮擊數的公式未拆（004ee220 hAttacks）。

特技：SoD 即死欄 15、Zatoichi 5 已含在擊殺內。未建模：空手打暈、Mindblast Rod 的打昏／扣耐力／降先手、先攻；打暈機率未取得。

### 七、NPC 兩人選擇（MG 2026-10-13：能到最終登天峰的 NPC：Vi 女、RFS-81 僧、Saxx 詩、Rodan 領、Drazic 忍）

條件：甲隊已用的物品（62、5、501、430、29、70、41、51、59、25、81）不給 NPC，The Avenger 排除；兩名 NPC 之間不共用（Stiletto 可多把）。候選池為 I9 範本每職業武器組（build/i9/party-c*.json）扣除上列，另補 SoD、Dread Spear。人類 L28 代理（NPC 實際屬性、T'rang 種族未建模），設定同第六部分，8 怪 Expert，擊殺／回合，選隊用 `scripts/i9-party.py` 的 solve。

各職業最佳：Drazic 忍 SoD .272（空手 .167、The Mauler .215）；RFS-81 僧 SoD .267（空手 .243）；Rodan 領 The Mauler＋Stiletto .254（Excaliber .249、SoD .243）；Vi 女 Excaliber .249（SoD .243、Dread Spear .200）；Saxx 詩 Coil of The Serpent＋Stiletto .148。

| 兩人 | 合計 |
|---|---|
| Rodan（Mauler＋Stiletto）＋Drazic（SoD） | .526 |
| Vi（Excaliber）＋Drazic（SoD） | .521 |
| Rodan＋RFS-81（SoD） | .521 |
| Vi（Excaliber）＋RFS-81（SoD） | .517 |
| Drazic（SoD）＋RFS-81（空手） | .515 |
| Vi（SoD）＋RFS-81（空手） | .486 |
| 任一＋Saxx | ≤ .420 |

- 前四組差 < .01，在代理誤差內，由非戰鬥能力決定；Saxx 剔除，賊 NPC MG 已放棄（人類賊最佳 .211）。
- Excaliber：MG 2026-10-13，水怪（尼斯）會掉，可刷，比 The Avenger 好取得。Excaliber、The Avenger、SoD 都是雙手；The Mauler 單手但不能放副手。
- Drazic（MG 給的 L14 起始：拳法 70、棍 46、CS 62）：忍者拳法封頂 100（無僧的 125 加成），練滿後空手 .167 遠低於 SoD；即死來自 Critical Strike（技能 19）主技能加成到 125，已建模（每下觸發 .05，僧 .04）。L14 實際值對 8 怪 Expert 每回合傷害空手約 2.0、SoD 約 2.7，兩者都打不動終局怪。T'rang 能否裝 SoD 未取得（種族遮罩位元未知，待 MG 實機）。
- 只跑 Expert 8 怪；Normal 56 怪未跑。

### 結論

1. **兩隊配裝不用改**。Normal 56 怪的名次與 10-08 相同，calc 值普遍低 1–10%（傷害公式，非設定差）；兩隊每位成員的武器在本職業內仍是 10-08 選的那組，本票未重跑全隊最佳化（I10 範圍）。
2. **答案要改的是數字，不是組合**：甲（Rogue／Fighter 妖 Ivory＋Sprite's／Ninja Cane＋Rod／Samurai／Lord／Monk SoD）總和 3.066→2.943；乙（Fighter 人 Ivory＋DE／Rogue／Ninja Cane＋Sai／Samurai／Bard／Gadgeteer）2.785→2.661。兩隊差 .28 不變。
3. **Monk 座頭市 vs 死亡之棍**：Normal SoD 略高（.385 對 .370），Expert 8 怪打平（.267 對 .268）；擊殺率模型沒算先攻（Zatoichi 先攻較高，MG）與 SoD 較高的即死欄（15 對 5，已含在擊殺內）。照 MG 實玩給 Monk Zatoichi、SoD 給 Valkyrie：Monk 在 Expert 不掉，Valkyrie 拿 SoD .362（Normal）／.243（Expert）。10-09 建議「SoD 給女武神」成立。
4. **Expert 對 MG 隊（侍、領、僧、遊、詩、工）的提示**：近戰擊殺 Samurai .339 ＞ Lord .274 ≈ Monk .268 ＞ 工 .150 ≈ 詩 .149；Ranger LS＋DE .352（人類）／.370（侏儒、哈比、貓人、雪人），但 LS 只一把（第五部分）。戰、賊在 Expert 擊殺仍高（.328／.407），但發瘋／反叛風險不在模型內（decisions/2026-10-12-mg-expert-party-experience.md）。
5. rank Train 的 CS 缺陷已修（840656b）；之後 rank 輸出才可當基準。
6. **Vi＋RFS-81（I11 後，第六部分）**：Vi 拿 SoD、RFS-81 空手（甲隊 Monk 已拿 Zatoichi），Mindblast Rod 不用；主力 Monk L28 起 Zatoichi 優於空手。
7. **NPC 兩人（第七部分）**：Rodan＋Drazic .526、Vi＋Drazic .521、Vi（Excaliber）＋RFS-81（SoD）.517 打平；Drazic 拿 SoD 的前提是 T'rang 能裝（未取得）。

標記：calc 值 `[已驗證：calc rank 跑完，prio 0,4,5,3 列]`；Savant 比 `[已驗證：calc rank 對 564 傷害]`；擊殺差中即死與取整的分攤 `未取得`；未建模（先攻順序、Lightning Strikes、遠程、法術、狀態）同 10-08。

---
<!-- 原檔 notes/research/2026-10-12-H3-rank-trial.md -->
## H3：calc rank 試算 v0（近戰，L1／10／28）

日期：2026-10-12。goals/stage15.md H3。cc 用 `calc rank`（H2）對 Dark Savant（id 564）與 Mad Marten（id 411，L22 中期怪）全種族 × 全職業 × 屬性排列試算。數字是 `round.damage`（每回合期望傷害），不是擊殺回合。

### 重跑

```sh
go run ./tools/calc rank -exe build/Wiz8.exe -dbs build/data-x/DATABASES notes/calc-qa/rank-h3-564.json > 564.tsv
go run ./tools/calc rank -exe build/Wiz8.exe -dbs build/data-x/DATABASES notes/calc-qa/rank-h3-411.json > 411.tsv
```

範本＝`rank-savant.json` 拿掉 `Classes` 限制（411 另改 `Monster`）：Attrs `[0,4,5,3]`（STR DEX SPD VIT 全排列 24 種）、SkillPrio `[0,16,18,34]`、主手 item 62／副手 item 25（fight-l28-06 的 LS＋Diamond Eyes）、Mode 0、Difficulty 1、無 Train。
3960 組各跑完 rc=0，skipped 1872（主手劍技能本職業不可用，或付不起職業門檻）。實測 564 那次 real 4m47s／user 36m35s（Fight 內部已用 goroutine 平行，`internal/calc/fight.go:201`）。

前置修正 3aa9a62：race 0 class 7 L1 時 004ee220 給出傷害% −5，原樣以無號 dword 綁進 5420b0 而 panic；改為符號延伸（`TestFightNegativeDamagePercent`，四段注入見該 commit）。

### 結果

種族／職業名依手冊順序對照 [推測：notes/re/growth.md:133，未讀 STRINGDATA 索引表]。

各職業 L28 最佳（種族 屬性序 → L1／L10／L28）：

| class | 564 Dark Savant | 411 Mad Marten |
|---|---|---|
| 0 Fighter | 10 Mook 0,5,4,3 → 2.42／3.22／98.78 | 同列 → 153.95 |
| 1 Lord | 0 Human 0,4,5,3 → 2.21／3.01／98.78 | 153.95 |
| 2 Valkyrie | 0 Human → 2.12／2.92／88.79 | 141.66 |
| 3 Ranger | 3 Gnome 0,5,4,3 → 2.15／2.96／94.78 | 149.04 |
| 4 Samurai | 4 Hobbit → 2.12／2.92／103.98 | 158.69 |
| 5 Ninja | 4 Hobbit → 2.15／2.96／94.78 | 149.04 |
| 7 Rogue | 5 Faerie 0,4,5,3 → 2.18／3.47／162.57 | 260.38 |
| 8 Gadgeteer | 4 Hobbit → 2.01／2.80／76.65 | 123.75 |
| 9 Bard | 5 Faerie → 1.98／2.78／76.65 | 123.75 |

Monk、Priest、Alchemist、Psionic、Mage、Bishop（6、10–14）不能用劍技能，整列跳過。

**前幾名（L28）**：兩隻怪都是 Rogue 第一（Felpurr／Faerie 並列 162.57／260.38），與 calc fight 的 L28 表 Rogue 居首同向（notes/calc.md:108）。Rogue 對第二名 Samurai 領先約六成，Fighter／Lord 再次之。

**Pareto 前緣（L1、L10、L28 三欄）**：
- 564：Hobbit Rogue（L28 158.6，L1 2.77）、Lizardman Rogue（L1 3.13／L10 4.69 最高，L28 113–151 依屬性序）、Felpurr Rogue（L28 162.57）。
- 411：同上三者，加 Faerie Rogue（L28 260.38）與 Dracon Fighter（L10 5.15 最高）。
- 前緣全是 Rogue（411 多一個 Fighter）：早期（L1／L10）Lizardman 最強，後期 Felpurr／Faerie 最強；同種族職業的不同屬性序常只差浮點噪音（≈1e-11），前緣中的同族多列多半是噪音而非真差異（H2 Pareto 精確比較，不設容差）。

### 假設與限制

- **練技不建模**：無 Train，技能只靠升級配點（創角 15、每級 9、每項 ≤5／≤3、裸值 ≤75，notes/re/growth.md）。所以 L28 主手劍技能 75、副手 Dagger 類技能（不在 SkillPrio）一律 0，絕對值遠低於 fight-l28 表（技能全 100），排名才有比較意義。
- **初始技能 0** [未取得：自動初始技能]。
- **性別不列舉**：職業的性別限制（Valkyrie 限女）不判定。
- **同一組終局武器用在每個等級**（L1 也拿 LS＋Diamond Eyes），只當能力曲線比較，不是實際裝備進程；裝備的種族／職業限制不檢查。
- **calc fight 未建模**（notes/calc.md:91）：Lightning Strikes（Samurai 被低估）、負重、裝備屬性加成、戰鬥技能 0x11、特效槽 5 以外、怪物回合內死亡、背後／側面幾何。
- 只比近戰；法術（A2）、生存（A3）、六人隊（A5）不在範圍。
- L10→L28 的大跳（約 30 倍）來自技能裸值與 Power Strike 解鎖（STR 裸值 100）疊加對高 AC 目標的命中率；未逐項拆解 [推測]。

### 補跑：fight-l28 十套武器（MG 2026-10-12 問雙持 CoC＋RoS 的 Faerie Ninja）

H3 主表全職業共用 LS＋Diamond Eyes，沒有逐職業挑武器；rank 也不檢查種族專屬裝備，所以 Rod of Sprites 那類組合不會出現。這裡改用 `notes/calc-qa/fight-l28-0N.json` 各自的種族、職業、武器，SkillPrio＝[主手技能, 副手技能（不同時）, 16, 18, 34]，其餘同主表；範本由 scratch 腳本產生，未入庫。L28 每回合傷害（564／411）：

| 套 | 組合 | 564 | 411 |
|---|---|---|---|
| 02 | Rogue Human 單持 LS | 199.3 | 294.1 |
| 10 | Samurai Human 單持 LS | 125.1 | 169.2 |
| 09 | Ninja Faerie 單持 Cane of Corpus | 111.2 | 164.2 |
| 03 | Ninja Faerie 雙持 Cane of Corpus＋Rod of Sprites | 110.6 | 181.1 |
| 01 | Rogue Faerie 雙持 LS＋Sprite's Dagger | 90.6 | 164.6 |
| 04 | Samurai Faerie 雙持 LS＋Rod of Sprites | 81.5 | 134.8 |
| 06 | Fighter Human 雙持 LS＋Diamond Eyes | 79.2 | 134.8 |

- **只靠升級配點時，雙持輸單持**：每級 9 點、每項 ≤3，SkillPrio 排在第 4、5 位的技能要到主手技能封頂（L25）之後才拿得到點，所以副手技能與 18 號技能一直偏低。fight-l28 表（技能全 100）裡雙持在前面，這裡反過來；這是配點模型造成的，不是遊戲結論。要比雙持，要在範本裡列舉 SkillPrio 或給 Train。
- 主表的 Fighter＝Lord、Samurai 第二：exe 主技能表（0x6161dc）的值是 Fighter 16、Lord 18、Samurai 0（劍）；主技能有效值 b+max(1,⌊b/4⌋)（internal/calc/build.go:146）。主表用劍，只有 Samurai 的加成落在武器技能上（75→93）。Fighter 的狂暴（Berserk）是攻擊模式，主表固定 Mode 0，沒有算到。Fighter 與 Lord 在 L28 到小數第 13 位都相同，原因未拆 [推測：兩者 16／18 號技能裸值都是 75，加成各落在一個對這組武器沒影響的技能上]。

---
<!-- 原檔 notes/research/2026-10-12-I4-rank-v1.md -->
## I4：calc rank 試算 v1（近戰，雙持 vs 單持）

日期：2026-10-12。goals/stage16.md I4（MG /goal）。數字是 `round.damage`（每回合期望傷害，對單一目標），不是擊殺回合。對 Dark Savant（564）、Mad Marten（411），L1／L10／L28。
種族／職業代碼依手冊順序 [推測：notes/re/growth.md:133]：種族 0 Human 1 Elf 2 Dwarf 3 Gnome 4 Hobbit 5 Faerie 6 Lizardman 7 Dracon 8 Felpurr 9 Rawulf 10 Mook；職業 0 Fighter 1 Lord 2 Valkyrie 3 Ranger 4 Samurai 5 Ninja 6 Monk 7 Rogue 8 Gadgeteer 9 Bard。
物品寫 id；中文名取 translations/items.json（501 不在表內，MG 原文 Ivory Blade）。

### 重跑

```sh
bash scripts/i4-rank.sh          # 輸出 build/i4/：summary.md（主情境表＋跨職業 Pareto）、main-*/ref*.tsv、party.tsv
```

需要 build/Wiz8.exe 與 build/data-x/DATABASES（本機原版，不入庫）。範本由 `scripts/i4-rank.py` 從 ITEMS.DBS 欄位產生，只寫 id。

### 情境

| 情境 | 屬性 | 技能 | Haste | 指令 | 武器組 |
|---|---|---|---|---|---|
| 主 | 每項 max(配點值, 100)，SPD 再 max(·,125) | 41 項全 Train 100（主技能經 b+⌊b/4⌋ 得 125） | 有 | Attack／Berserk 列舉 | 見下 |
| 次（實盤） | MG 面板值（notes/calc-qa/mg-party-l25.md），SPD max(·,125) | 面板裸值（主技能反推 raw） | 有 | Attack（Mode 0） | MG 實盤武器 |
| 參考 | 只配點，STR 先／DEX 先 | 只配點，SkillSet 列舉 | 無 | 列舉 | MG 四組 |

主情境武器組，每職業：
1. 篩選：143 件近戰武器（技能 0..5、射程 0/1、有骰、有模式）單持 × 10 職業 × 11 種族，L28，兩怪。
2. 主手＝篩選值前 4 名 ∪ MG 武器。
3. 副手池＝該職業合法副手（單手且可副手，共 30 件）中篩選值前 8 名 ∪ MG 副手（MG 2026-10-12：「先列舉出所有 top tier…當作武器池…再測試主手跟副手排列組合」）。
4. 武器組＝各主手單持＋主手×副手池（主≠副）。

剪枝驗證：先跑過一次 30 件全副手版（build/i4-full，59 分 36 秒），60 格（10 職業×2 怪×3 等級）的最佳雙持副手，在該職業合法副手中的名次全部 ≤7（Samurai L1 第 7，Monk 第 2–4，其餘多為第 1）。剪枝版的 summary 表與 Pareto 和全副手版逐字相同（本檔數字即兩者共同結果）。

### 第一題：雙持 vs 單持（主情境）

各格最佳雙持 ÷ 最佳單持（所有種族、指令、武器組中取最大）：

| 職業 | 564 L1 | L10 | L28 | 411 L1 | L10 | L28 |
|---|---|---|---|---|---|---|
| 0 Fighter | **0.75** | 1.08 | 1.12 | **0.77** | 1.17 | 1.07 |
| 1 Lord | 1.51 | 1.26 | 1.19 | 1.42 | 1.35 | 1.29 |
| 2 Valkyrie | 1.11 | 1.22 | 1.17 | 1.10 | 1.32 | 1.24 |
| 3 Ranger | 1.11 | 1.22 | 1.28 | 1.19 | 1.32 | 1.24 |
| 4 Samurai | 1.04 | 1.22 | 1.26 | 1.08 | 1.23 | 1.16 |
| 5 Ninja | 1.08 | 1.17 | 1.22 | 1.12 | 1.21 | 1.23 |
| 6 Monk | **0.83** | **0.76** | 1.11 | 1.00 | **0.89** | 1.15 |
| 7 Rogue | 1.08 | 1.08 | 1.28 | 1.13 | 1.11 | 1.32 |
| 8 Gadgeteer | 1.10 | 1.10 | 1.35 | 1.14 | 1.12 | 1.30 |
| 9 Bard | 1.10 | 1.10 | 1.35 | 1.14 | 1.12 | 1.30 |

結論（照實）：
- 屬性 100、技能 100、Haste 下，L28 每個職業都是雙持贏，幅度 1.07–1.35。與 MG「str、dex、spd 跟 dual wield 都滿的情況下，雙持不可能輸給單持」一致。
- 例外在低等級：
  - Fighter L1：單持 Berserk 光劍 62 贏（247 vs 185；411 為 338 vs 262）。
  - Monk L1／L10：座頭市之棍 81 單持贏或打平。Monk 雙持最佳是重擊者 772＋風之魔杖 42／雙節棍 56。
  - 原因 [推測：未逐項拆解]：低等級每回合攻擊次數少，副手的命中／傷害懲罰比多一手的收益大。
- 參考情境（只配點，不 Train、無 Haste）方向相反：MG 四組的雙持在 L28 全部輸給同主手單持（見「參考情境」）。
  雙持靠高雙持技能與屬性；只靠升級配點到不了那裡（MG 2026-10-12：「單靠升級配點，反而不是實際情況」）。

### 各職業最佳（主情境，L28）

| 職業 | 564 最佳（雙持） | 411 最佳 | MG 組（564／411） |
|---|---|---|---|
| 0 Fighter | 62 光劍＋25 鑽石之眼 Berserk 568 | 同組 597 | 501 Ivory Blade＋70 Berserk 456／503（Faerie） |
| 1 Lord | 62＋25 331 | 660 嗜血＋25 391 | 實盤 772＋25 見次情境 |
| 2 Valkyrie | 62＋25 328 | 660＋25 378 | — |
| 3 Ranger | 62＋25 328 | 660＋25 378 | — |
| 4 Samurai | 62＋70 妖精之杖 324（Faerie） | 660＋511 Canezou 匕首 357 | 41 村正＋51 附魔脇差 241／243 |
| 5 Ninja | 29 骸之杖＋70 375（Faerie） | 同組 377 | 同左（MG 組即最佳） |
| 6 Monk | 772＋42 259 | 772＋42 269 | 實盤 81 單持 |
| 7 Rogue | 62＋5 竊賊匕首 472 | 同組 485 | 同左（MG 組即最佳） |
| 8 Gadgeteer | 62＋70 230（Faerie） | 同組 239 | 實盤 62＋3 見次情境 |
| 9 Bard | 62＋70 230（Faerie） | 同組 239 | 實盤 59／780＋511 見次情境 |

- MG 四組：Ninja、Rogue 已是該職業最佳；Samurai 的村正＋附魔脇差約為最佳的 74%／68%；Fighter 的 Ivory Blade＋妖精之杖約 80%／84%（MG 2026-10-12 更正 Fighter 副手為 Rod of Sprites，原列 25 鑽石之眼 461／517）。
- 種族：屬性被 AttrMin 100 墊平後，同武器組各種族傷害相同；種族只透過物品種族遮罩影響（妖精之杖 70、骸之杖 29 只有 Faerie 合法，所以 Ninja／Samurai／Gadgeteer／Bard 的最佳列只有 Faerie）。
- Fighter 全部最佳都是 Berserk；其餘職業 Attack。
- 只比 Attack（不含 Berserk）時 Rogue 第一：Rogue 62＋5 472／485，Fighter Attack 最佳 62＋25 329／340。與 2026-10-08 L28 排名（notes/research/ranking-history.md:104，只列一般攻擊，Rogue 雙持 LS 第一）一致；加入 Berserk 後 Fighter 第一。
- Rogue 的數字含背刺：calc fight 由 53d870 設背刺旗標（Rogue、射程 0 的劍或匕首），5420b0 多擲骰。對照 L28 精靈單持：62 光劍 Rogue 368、Gadgeteer 171；19 號錘（不能背刺）兩者都是 33.5。

### Pareto

完整 Pareto 在 `build/i4/summary.md` 末段（跨職業，三個等級的支配比較，同值種族合併為一列）：

- 564：Fighter 62＋25 Berserk（185／308／568）、Fighter 62 單持 Berserk（247／287／507）、Rogue 62＋5（193／306／472）。
- 411：Fighter 62＋25 Berserk（262／399／597）、Fighter 62 單持 Berserk（338／342／513）。

各職業檔內的 Pareto（`# pareto` 段）見各 main-c*-*.tsv。

### 次情境：MG 實盤（calc fight，Haste，Mode 0）

| 位 | 單持 564 | 雙持 564 | 單持 411 | 雙持 411 |
|---|---|---|---|---|
| Samurai 41／＋51 | 167.2 | 170.4 | 199.5 | 228.9 |
| Lord 772／＋25 | 207.6 | 274.0 | 222.3 | 305.9 |
| Bard 59 獠牙／＋511 | 108.2 | **90.1** | 145.8 | **128.5** |
| Bard 780 蛇牙／＋511（精靈裝不了，僅參考） | 49.6 | 52.1 | 67.4 | 75.7 |
| Gadgeteer 62／＋3 | 86.6 | **67.8** | 124.1 | **107.5** |
| Monk 81（單持） | 53.0 | — | 89.6 | — |
| Ranger 778（單持） | 47.2 | — | 73.1 | — |

- Bard（獠牙）與 Gadgeteer 實盤雙持低於單持。兩人雙持裸值 76、43，主手技能也不是雙持；與 MG 的前提「雙持技能滿」不同，所以不算與 MG 實盤矛盾。
- Lord（雙持 115）、Samurai（雙持 92）雙持贏。
- 「龍牙 303」不是武器（技能欄 255、無骰），無法進 calc（實跑 Bard 主手 303：`calc attacks: invalid item type`）；MG 原文「牙」改跑獠牙 59 與蛇牙 780（匕首）。蛇牙種族限制只有蜥蜴人／龍人（全 ITEMS.DBS 唯一一件只限這兩族），實盤 Bard 是精靈，裝不了；calc fight 不查種族，那列只當參考，實盤以獠牙 59 為準（calc rank 有查種族，主情境不受影響）。

### 空手（I7，主情境）

MG 2026-10-12：「Monk 空手可以先開票，跟先前計算的一起列，因爲這也是近戰。記得 Monk 的空手是可以 125 的」。I7（977696c）接上空手後，每職業的武器組多一組 `unarmed`（無主手、無副手，技能 14），其餘條件同主情境。

| 職業 | 564 L1／L10／L28 | 411 L1／L10／L28 | 空手 ÷ 單持最佳（L28 564／411） |
|---|---|---|---|
| 5 Ninja | 51／61／118 | 70／81／124 | 0.38／0.40（29 骸之杖 308／308） |
| 6 Monk | 83／96／169 | 109／113／170 | 0.72／0.73（81 座頭市之棍 234／234） |
| 其他 8 職業 | NA（技能 14 不可用，計入 `skipped weapon-skill`） | NA | — |

- Monk 技能 14 是主技能：raw 100 經 b+⌊b/4⌋ 得有效 125（`TestFightUnarmed` (a)，與 MG 記憶一致）。即使 125，三個等級、兩隻怪，空手都低於座頭市之棍 81 單持，也低於雙持最佳；不進任何 Pareto。
- 空手不列 Berserk（rank 對無主手的組不列舉 Berserk，I7 規格）；Monk、Ninja 的最佳本來就是 Attack，不影響上表的比較。
- 數字是攻擊模式分布的期望：空手技能 14 有效值 ≥5 時可選模式 5、6，rank 依 AI 選模式的分布加權。單跑 calc fight Mode 6（Monk L28、屬性 100、564）是 182.4，Mode 5 是 157.8，加權後得 168.7。
- 加入空手後，既有武器組的 TSV 資料列、summary 其他欄、Pareto、party.tsv 與加入前逐字相同（只多了 `unarmed` 列與 skipped 計數）。
- **更正（I11，2026-10-13，e846fa4）**：上表與 113 行的空手值少算副手拳（calc 跳過 004ee220 開啟的第二手）。修後 Monk 空手（人類 L28，8 怪 Expert 擊殺）.243 對 81 單持 .268（約 0.91），L20 時空手反而領先（.144 對 .120）；見 `2026-10-12-baseline-align.md` 第六部分。本節表格未重跑。

### 參考情境：只配點（MG 四組，L1／L10／L28 最佳）

| 職業 | 564 雙持 | 564 單持主手 | 411 雙持 | 411 單持主手 |
|---|---|---|---|---|
| 0 Fighter 501＋70 | 3／4／52 | 3／4／108 | 3／4／87 | 3／4／182 |
| 4 Samurai 41＋51 | 2／2／63 | 1／3／100 | 2／2／88 | 1／3／125 |
| 5 Ninja 29＋70 | 2／3／84 | 1／2／91 | 2／3／122 | 1／2／123 |
| 7 Rogue 62＋5 | 3／5／129 | 2／4／188 | 3／5／194 | 2／8／250 |

雙持 SkillSet＝主手技能、副手技能、近身、雙持；單持＝主手技能、近身、強力打擊；屬性 STR 先／DEX 先。L1／L10 數字小，是因為只配點的技能值低、命中低。

### 假設與限制

- **屬性排列縮成 1 種**：主情境 AttrMin 100 下，原本的 Attrs `[0,4]` 兩種排列在 7554 組（種族×職業×武器組×指令×等級）裡每組只有一種相異值，所以縮成 `[0]`。參考情境保留兩種。
- **Train 範圍**：範本給全 41 項 100（目標是「用到的技能」）。對照只 Train 武器技能 0..5、近身 16、雙持 18、致命一擊 19、強力打擊 34：全 10 職業 × 怪 564／411 主情境 TSV 逐 byte 相同（20/20），其餘技能不影響近戰輸出。
- **Train 不看可學表**：rank 的 Train 對 41 項技能一律加，不看職業能不能學（Avail）。實驗：加上 Avail 過濾後，整份篩選 TSV 逐 byte 相同，所以沒有改（沒有會紅的測試）。
- 需求檢查用 Haste 後的屬性（SPD 125）[推測：遊戲判需求用當下值]。
- Haste 建模為 SPD＝max(SPD,125) 常駐，不建模持續回合（MG 2026-10-12：「就當作 speed 是 125」）。
- 次情境：面板屬性含裝備加成，當裸值餵入；主技能由面板值反推裸值；技能 id 34..40 對照 [推測]。
- 未建模：遠程副手（Ranger 三發弩 739）、calc fight 未建模清單（notes/calc.md:15#未建模）、練技成長、性別。
- 篩選只用 L28 單持值排主手與副手池；L1／L10 的最佳若落在池外，會漏 [推測：剪枝驗證只看了全副手版的 60 格]。
- 浮點：rank 的 fightExpect 原本依 goroutine 完成順序加總，同輸入重跑末位不同；781152d 改成排序後加總（`TestFightDeterministic`）。
- 執行時間：全副手版 59 分 36 秒；剪枝版 42 分 41 秒；d77612c 修正後剪枝版 18 分 18 秒（TSV 只差第 13 位有效數字，summary 與 Pareto 相同）。加入空手後 12 分 28 秒。
  Rogue 原本每檔約 21 分：5420b0 的連擊門檻 `R(100) < 20*count-20` 在 calc fight 被逐值展開成 100 支，疊在傷害骰之下，一場約 71 萬節點；count>1（Rogue 等級連擊、Berserk）才走到。遊戲本身只擲一次比門檻，是 calc 的列舉寫法，不是遊戲規則。改走二分後數值相同（`TestFightRogueStrikeBranches`）。

---
<!-- 原檔 notes/research/2026-10-13-I9-party-expert.md -->
## I9b 全隊最佳化：Expert 8 怪

日期：2026-10-13。goals/stage16.md I9b。MG：「不如跑隊伍最佳化」；基準依 MG 同日「Expert 才是比較基準」。

### 設定與偏離

- 同 baseline-align 第七部分：8 怪 564、114、184、171、129、158、120、192，Difficulty 2，L28，屬性 0,4,5,3 的全部 24 種排列，AttrMin 0，不開 Haste，Train 武器 0–5、14、16、18、19 為 100、PS 34 為 58，Metric kill。11 種族全開。計算器 build/i11/calc（含 I11 空手副手拳）。
- 偏離 I9 原設定：原 I9 是 Normal 56 怪，這次改 Expert 8 怪（MG 裁定）。
- 偏離 I9b 列「指令 attack」：Fighter 用 attack＋berserk 兩種都算（同 baseline-align 設定「Fighter 用 berserk」），其餘只 attack。
- 武器池沿用 I9 範本（build/i9/party-c*.json，篩選來自 Normal 56 怪的 I4，每職業主手前 6、副手前 8，加 MG 裝備與 Stiletto）。Expert 下可能漏掉好武器，見「池邊界」。
- 限制：每件物品一把（Stiletto 3 不限），The Avenger 61 排除（範本已排除）。

### 重跑

```
python3 -I scripts/i9-party.py expert build/i9 build/i9x
for c in 0 1 2 3 4 5 6 7 8 9; do build/i11/calc rank -exe build/Wiz8.exe -dbs build/data-x/DATABASES build/i9x/party-c$c.json > build/i9x/party-c$c.tsv; done
python3 -I scripts/i9-party.py party build/i9x
```

`party` 先對每職業跑 `check`：每個種族×配裝要嘛有輸出列組、要嘛記在 `# skipped`，每組列數＝屬性排列數 24，組數＝配裝對×指令數−berserk 略過數。10 職業全過。本次範本由暫存區產生，與 `expert build/i9` 重產的 10 份 JSON 解析後逐份相同（只差鍵順序）。

### 耗時

10 職業平行（另有一份 Fighter 只算 attack 的被中止，1202 s 浪費），牆鐘 03:11:49 → 03:59:24，約 48 分。單職業（秒）：戰 2001（attack＋berserk）、領 1873、女 2231、遊 1776、侍 2396、忍 2393、僧 2138、賊 2855、工 1838、詩 1798。

### 抽查（第七部分已算過的格，人類 L28，屬性 0,4,5,3）

| 格 | 第七部分 | 本次 |
|---|---|---|
| 僧 SoD（86） | 0.26740033117060003 | 相同 |
| 僧 空手 | 0.24266937469389999 | 相同 |
| 女 i62+i25 | 0.34798394042149999 | 相同 |
| 女 i59+i25 | 0.26644426201920002 | 相同 |
| 女 i501+i25 | 0.25520833333330001 | 相同 |
| 女 SoD（86）（補跑） | 0.24344764901230001 | 相同 |
| 女 Dread Spear（815）（補跑） | 0.200496031746 | 相同 |

SoD、Dread Spear 不在 I9 女武神池裡（池內主手 660、62、33、34、74、772＋MG 裝備），另以同設定補跑一份只含這兩件的女武神範本（build/i9x/party-c2.json 換 Loadouts，11 族×24 排列，68 s，`check` 通過）。11 族最佳仍是人類 0,4,5,3 的 .2434。把這兩件併進女武神選項重解 15 隊：只有戰賊忍女領遊改變（下表已用補池後的解），其餘 14 隊合計與成員不變。

### 結果

種族：r0 人類、r4 哈比、r5 妖精、r8 貓人、r10 雪人（Mook）。物品：3 Stiletto、25 Diamond Eyes、29 Cane of Corpus、34 Excaliber、41 Muramasa、48 Coil of the Serpent、51 Enchanted Wakizashi、59 Fang、62 Light Sword、70 Rod of Sprites、86 Staff of Doom、501 Ivory Blade、772 The Maul。

A 範本（戰賊忍＋從僧女領侍遊選 3）。未補池時戰賊忍女領遊為 1.941（女 i59+i25 .266、領 i772+i3 .254、遊 i501+i3 雪 .254）：

| 隊 | 合計 | 對甲隊 | 戰 | 賊 | 忍 | 其餘三人 |
|---|---|---|---|---|---|---|
| 戰賊忍僧領侍 | 2.073 | +.048 | i501+i3 berserk 雪 .345 | i62+i3 妖 .417 | i29+i70 妖 .409 | 僧 i86 妖 .280；領 i59+i3 人 .283；侍 i41+i51 哈 .339 |
| 戰賊忍僧女侍 | 2.056 | +.031 | i501+i3 berserk 雪 .345 | 同上 | 同上 | 僧 i86 妖 .280；女 i59+i25 人 .266；侍 i41+i51 哈 .339 |
| 戰賊忍女領侍 | 2.044 | +.019 | i34 berserk 雪 .341 | 同上 | 同上 | 女 i501+i25 人 .255；領 i59+i3 人 .283；侍 i41+i51 哈 .339 |
| 戰賊忍領侍遊 | 2.044 | +.019 | i34 berserk 雪 .341 | 同上 | 同上 | 領 i59+i3 人 .283；侍 i41+i51 哈 .339；遊 i501+i25 人 .255 † |
| 戰賊忍僧侍遊 | 2.041 | +.016 | i34 berserk 雪 .341 | 同上 | 同上 | 僧 i86 妖 .280；侍 i41+i51 哈 .339；遊 i501+i25 人 .255 † |
| 戰賊忍女侍遊 | 2.026 | +.001 | i34 berserk 雪 .341 | 同上 | 同上 | 女 i59+i25 人 .266；侍 i41+i51 哈 .339；遊 i501+i3 雪 .254 † |
| 戰賊忍僧女領 | 1.985 | −.040 | i34 berserk 雪 .341 | 同上 | 同上 | 僧 i86 妖 .280；女 i501+i25 人 .255；領 i59+i3 人 .283 |
| 戰賊忍僧領遊 | 1.985 | −.040 | i34 berserk 雪 .341 | 同上 | 同上 | 僧 i86 妖 .280；領 i59+i3 人 .283；遊 i501+i25 人 .255 † |
| 戰賊忍僧女遊 | 1.967 | −.058 | i34 berserk 雪 .341 | 同上 | 同上 | 僧 i86 妖 .280；女 i59+i25 人 .266；遊 i501+i3 雪 .254 † |
| 戰賊忍女領遊 | 1.949 | −.076 | i34 berserk 雪 .341 | 同上 | 同上 | 女 i86 人 .243（補池）；領 i59+i3 人 .283；遊 i501+i25 人 .255 † |

B 範本（詩工戰賊忍＋選 1；詩 i48+i3 妖 .148、工 i59+i3 貓 .150）：

| 隊 | 合計 | 戰 | 賊 | 忍 | 第六人 |
|---|---|---|---|---|---|
| 詩工戰賊忍侍 | 1.809 | i501+i3 berserk 雪 .345 | i62+i3 妖 .417 | i29+i70 妖 .409 | 侍 i41+i51 哈 .339 |
| 詩工戰賊忍僧 | 1.749 | i501+i3 berserk 雪 .345 | 同上 | 同上 | 僧 i86 妖 .280 |
| 詩工戰賊忍領 | 1.746 | i34 berserk 雪 .341 | 同上 | 同上 | 領 i501+i25 人 .280 † |
| 詩工戰賊忍女 | 1.721 | i34 berserk 雪 .341 | 同上 | 同上 | 女 i501+i25 人 .255 |
| 詩工戰賊忍遊 | 1.721 | i34 berserk 雪 .341 | 同上 | 同上 | 遊 i501+i25 人 .255 † |

值為每回合擊殺（8 怪等權平均），取該職業該配裝在 11 種族×24 屬性排列中的最高值（同分取先出現者）。換成固定 0,4,5,3 時差 ≤ .0031（戰 i34 雪 .3414→.3383 最大；戰 i501+i3、領 i59+i3、領 i772+i3、遊 i501+i3 也取非 0,4,5,3 排列，差 ≤ .0004），其餘成員最佳排列即 0,4,5,3。

### 對照甲隊（I10，baseline-align 第二部分 Expert）

甲隊＝戰 Ivory＋Sprite's .328、賊 LS＋TD .407、忍 Cane＋Rod .409、侍 村正＋EW .339、領 Fang＋DE .274、僧 Zatoichi .268，合計 2.025。本次輸出在甲隊各格（同族、0,4,5,3）重現：戰 妖 i501+i430 berserk .32827、賊 人 i62+i5 .40675、領 人 i59+i25 .27356、僧 人 i81 .26805、忍 .4091、侍 .3389。同職業組合的最佳隊 2.073（+.048，+2.4%）：

- 戰 .328→.345：Ivory 改配 Stiletto、雪人。
- 賊 .407→.417：LS 改配 Stiletto（TD 讓出）、妖精。
- 僧 .268→.280：Zatoichi 改 SoD、妖精（人類 SoD .267，與 Zatoichi 打平，見第七部分）。
- 領 .274→.283：Fang 改配 Stiletto（DE 讓出）。
- 忍、侍不變。

差距都在 .02 以內一人，模型沒算先攻、發瘋／反叛風險與 Normal 56 怪；選隊不必因此改。A 範本前 6 名都高於甲隊，其中前 3 名都含侍。

### 池邊界（†）

邊界＝該職業篩選排名的最後一名（主手第 6、副手第 8）。最佳隊成員落在邊界的：

- 遊 i501（Ivory Blade）：遊俠主手篩選第 6 名（660、62、33、778、772、501）。
- 領 i501：領主主手篩選第 6 名（660、62、33、34、772、501），只出現在 B 範本詩工戰賊忍領。

副手沒有人落在邊界。另，戰 i501、女 i59／i501、侍 i41 不在篩選排名內，是 MG 裝備補進池的；這些人的對手武器只看過篩選前 6。池放寬（Expert 重篩）可能換掉遊、領的武器，也可能讓女武神、遊俠與領主的排名互換。

---
<!-- 原檔 notes/research/2026-10-13-I9c-pool-expert.md -->
## I9c 全隊最佳化：Expert 重篩武器池

日期：2026-10-13。`goals/stage16.md` 的 I9c。設定照 I9b（`notes/research/ranking-history.md:827`）：8 怪 564、114、184、171、129、158、120、192，Difficulty 2，L28，11 種族、24 種屬性排列，Fighter 算 attack 和 berserk 兩種，Metric kill。只換武器池。

### 最高分隊：戰賊忍僧領侍 2.126

| 職業 | 種族 | 武器 | 指令 | 屬性順序 | 值 |
|---|---|---|---|---|---|
| 戰士 | 雪人（Mook；哈比、侏儒、人類、貓人差 ≤ .0003，並列） | The Avenger | berserk | 力 速 敏 體 | .399 |
| 盜賊 | 妖精 | Light Sword＋Stiletto | attack | 力 敏 速 體 | .417 |
| 忍者 | 妖精 | Cane of Corpus＋Rod of Sprites | attack | 力 敏 速 體 | .409 |
| 僧侶 | 妖精 | Staff of Doom | attack | 力 敏 速 體 | .280 |
| 領主 | 人類 | Fang＋Stiletto | attack | 敏 速 力 體 | .283 |
| 武士 | 哈比 | Muramasa＋Enchanted Wakizashi | attack | 力 敏 速 體 | .339 |

- 屬性 id：0 力、3 體、4 敏、5 速（「屬性順序」是 L28 加點的先後）。值是每回合擊殺數，取自 build/i9e/party-c*.tsv。
- 唯一物品：The Avenger、Light Sword 各一把，此隊各用一次。Fang 是必得的表外來源（見 notes/calc.md:263）。

**MG 追問（2026-10-13）**，數字取自 build/i9e/party-c6.tsv、party-c7.tsv：
- 盜賊的種族：i62+i3 在 11 族中最高是妖精 .4169、最低是矮人 .4147。貓人 .4167、精靈 .4162、哈比 .4162、侏儒 .4161、人類 .4160、雪人 .4160，和妖精差 ≤ .001，算並列。所以種族可以照負重或裝備限制選，模型裡幾乎沒差。
- 盜賊副手：Thieves Dagger（5）比 Stiletto（3）低，妖精 i62+i5 .4073 對 i62+i3 .4169（−.0096），各種族都低約 .01。即死有算進去（這把是 5，Stiletto 是 10）；另一個效果（MG：像是中毒）沒有算。
- 僧侶：Staff of Doom 對妖精、精靈、貓人最好（.2795／.2787／.2787）。哈比、人類、侏儒、雪人、狼人最好的是 Zatoichi Bo（.2683～.2676）。妖精用 Zatoichi Bo 是 .2715，比 Staff of Doom 低 .008。

### 重跑

```
bash scripts/i9c-run.sh              # 篩選（沒有 screen-c*.tsv 才跑）→ 建池 → 10 職業 rank → party.md
bash scripts/i9c-run.sh 8 10 1,2     # 池邊界：放寬主手前 8、副手前 10，只重跑指定職業
python3 -I scripts/i9-party.py compare build/i9x build/i9e   # 和 I9b 比對共有格，並列出池差與換人
```

- **篩選方式**：人類、L28、屬性 0,4,5,3、Expert 8 怪，每把近戰武器單持計算一次。副手武器也用它單持時的值排名。
- **池子**：每職業主手取前 6、副手取前 8，再加上 MG 裝備、Stiletto，以及該職業可用的種族專武。
- **The Avenger 的篩選**：它是在主篩選跑完後才移出排除清單，所以另外補跑戰、領、女三份 `screen-c{0,1,2}-a61.tsv`。`screened()` 會一併讀 `screen-c<n>-*.tsv`。全新重跑時主篩選本來就含 61，不需要這三份。
- **檢查**：`party` 先跑 `check`，斷言每職業的列數＝種族×配裝，NA 列也算。10 職業全部通過，三輪都有跑。

### 偏離 goal

- **種族專武**：篩選只算人類，所以人類不能用的武器排不進前 6。CoC、Giant's Sword 這類種族專武，依職業遮罩直接加進池子，不看篩選名次（MG：「除非是種族特有武器例如我提出的巨人之劍、coc 等」）。
- **排除清單**改成 85、93、96、465、489、748、749：
  - 這 7 個 id 是 MG 說遊戲中沒印象的 Rammbus Staff、Longstem Spade、Maenad's Lance、Flamequencher Wand、Instakiller、Stinger×2。
  - 它們在掉落表、怪物、商店裡也都查不到。
  - The Avenger（61）從排除改成計算：它在 Marten Chest 的掉落表（57）裡，權重 3/197。MG 說寶箱不會重生，也沒有怪掉這張表，所以只有一把。
  - 因為排除清單改了，現在用 `templates`／`expert` 重產 I9／I9b 範本，不會和 build/i9 逐 byte 相同：I9 的賊池原本有 489。
- **代碼**：cc 自己寫，沒有派給 codex。
- **執行順序**：第一版先用舊排除清單跑完 10 職業（build/i9e/v1，最佳隊 2.073）。之後才放進 The Avenger、拿掉 465，再重跑戰、領、女、僧（build/i9e/v2）。

**照 goal 原文排除 The Avenger 的結果**（第一版，build/i9e/v1/party-v1.md；和 I9b 共有格 89232 格 0 差）：

| 隊 | 合計 | 隊 | 合計 |
|---|---|---|---|
| 戰賊忍僧領侍 | 2.073 | 戰賊忍僧女領 | 1.985 |
| 戰賊忍僧女侍 | 2.056 | 戰賊忍僧領遊 | 1.985 |
| 戰賊忍領侍遊 | 2.052 | 戰賊忍僧女遊 | 1.968 |
| 戰賊忍女領侍 | 2.050 | 戰賊忍女領遊 | 1.964 |
| 戰賊忍僧侍遊 | 2.041 | 詩工戰賊忍侍 | 1.810 |
| 戰賊忍女侍遊 | 2.035 | 詩工戰賊忍領 | 1.752 |
| | | 詩工戰賊忍僧 | 1.749 |
| | | 詩工戰賊忍遊 | 1.728 |
| | | 詩工戰賊忍女 | 1.727 |

這版的戰士是 i501+i3 或 i86 berserk 雪人（.345／.347）。其他成員的差別逐隊見 party-v1.md（例如戰賊忍僧女遊的僧侶在這版是 i81 .272）。

### 耗時

| 段 | 時間 | 說明 |
|---|---|---|
| 篩選 10 職業平行 | 08:57 → 09:54 | 單職業 2032–3385 s |
| 全隊 10 職業平行 | 09:54 → 10:52 | 賊 3537 s 最久；中途降成 nice 19 讓出 CPU |
| 第一版產隊伍 | 10:52 → 約 11:09 | |
| The Avenger 補篩選 | 3 份 | 188–247 s，和上一段重疊 |
| 戰領女僧重跑 | 11:09:48 → 11:27:41 | 999–1073 s |
| 池邊界領女重跑（8/10） | 11:29:58 → 11:39:24 | 565 s |
| 和 I9b 比對 | 11:39 → 11:49 | |

- 牆鐘合計約 2 小時 52 分。另外有一次比對中途中止，浪費約 10 分鐘。
- 每職業的結束碼和秒數在 build/i9e/run.log。

### 結果

種族：r0 人類、r4 哈比、r5 妖精、r8 貓人、r10 雪人（Mook）。

物品：

| id | 物品 |
|---|---|
| 3 | Stiletto |
| 25 | Diamond Eyes |
| 29 | Cane of Corpus |
| 41 | Muramasa |
| 48 | Coil of the Serpent |
| 51 | Enchanted Wakizashi |
| 59 | Fang |
| 61 | The Avenger |
| 62 | Light Sword |
| 70 | Rod of Sprites |
| 86 | Staff of Doom |
| 501 | Ivory Blade |

每個隊都有的固定成員：
- 戰 i61 berserk 雪 .399
- 賊 i62+i3 妖 .417
- 忍 i29+i70 妖 .409

A 範本（戰賊忍，加上從僧女領侍遊選 3 人）：

| 隊 | 合計 | 對甲隊 | 其餘三人 |
|---|---|---|---|
| 戰賊忍僧領侍 | 2.126 | +.101 | 僧 i86 妖 .280；領 i59+i3 人 .283；侍 i41+i51 哈 .339 |
| 戰賊忍僧女侍 | 2.110 | +.085 | 僧 i86 妖 .280；女 i59+i25 人 .266；侍 i41+i51 哈 .339 |
| 戰賊忍領侍遊 | 2.105 | +.080 | 領 i59+i3 人 .283；侍 i41+i51 哈 .339；遊 i86 雪 .259 |
| 戰賊忍女領侍 | 2.102 | +.077 | 女 i501+i25 人 .255；領 i59+i3 人 .283；侍 i41+i51 哈 .339 |
| 戰賊忍僧侍遊 | 2.099 | +.074 | 僧 i86 妖 .280；侍 i41+i51 哈 .339；遊 i501+i25 人 .255 |
| 戰賊忍女侍遊 | 2.089 | +.064 | 女 i59+i25 人 .266；侍 i41+i51 哈 .339；遊 i86 雪 .259 |
| 戰賊忍僧女領 | 2.043 | +.018 | 僧 i86 妖 .280；女 i501+i25 人 .255；領 i59+i3 人 .283 |
| 戰賊忍僧領遊 | 2.043 | +.018 | 僧 i86 妖 .280；領 i59+i3 人 .283；遊 i501+i25 人 .255 |
| 戰賊忍僧女遊 | 2.025 | +.000 | 僧 i86 妖 .280；女 i59+i25 人 .266；遊 i501+i3 雪 .254 |
| 戰賊忍女領遊 | 2.022 | −.003 | 女 i501+i25 人 .255；領 i59+i3 人 .283；遊 i86 雪 .259 |

B 範本（詩工戰賊忍，加 1 人）。固定成員另有詩 i48+i3 妖 .148、工 i59+i3 貓 .150。

| 隊 | 合計 | 第六人 |
|---|---|---|
| 詩工戰賊忍侍 | 1.863 | 侍 i41+i51 哈 .339 |
| 詩工戰賊忍領 | 1.804 | 領 i501+i25 人 .280 |
| 詩工戰賊忍僧 | 1.803 | 僧 i86 妖 .280 |
| 詩工戰賊忍遊 | 1.782 | 遊 i86 雪 .259 |
| 詩工戰賊忍女 | 1.779 | 女 i501+i25 人 .255 |

完整輸出在 build/i9e/party.md。值的定義同 I9b：每回合擊殺，取 11 種族×24 排列中的最高值，同分取先出現的。

**種族：差距在 .001 以內寫「並列」**
- 戰 i61 berserk 的前 5 名是雪 .39906、哈 .39893、侏儒 .39893、人 .39880、貓 .39880，五族並列，不能說雪人最佳。
- 其餘 5 族在 .392 左右。
- MG 的遊玩經驗：蜥蜴人最快拿到 PS，但整體是哈比好，因為屬性不偏科，技能長得快。這是經驗，不是量測。種族在本模型只影響 L28 的屬性分配，升級速度沒有建模。

### 和 I9b 比對

- `compare build/i9x build/i9e`：兩次都有算到的格（同職業、種族、配裝、指令、屬性順序）共 89088 格，0 格不同。
- 第一版（舊排除清單）另外比過一次：89232 格，0 格不同。

**池差**（I9b → I9c，+ 新進、− 移出）。63、70、430 是妖精專武，因為種族專武例外加進每個可用的職業：

| 職業 | 主手 | 副手 |
|---|---|---|
| 戰 | +61、63、70、86、430；−16、33、74 | +4、63、68 |
| 領 | +61、63、70、86、430、778、815；−33、660 | +2、63、70、430、592 |
| 女 | +61、63、70、86、430、778、815；−33、660 | +2、63、70、430、592 |
| 遊 | +37、63、70、86、430、815；−660 | +4、63、68 |
| 侍 | +63、70、86、430；−33、660 | +4、63、68、676 |
| 忍 | +48、63、70、430 | +50、55、63、591 |
| 僧 | +63、70、780 | +63、64、681、780；−465 |
| 賊 | +5、36、63、70、430；−57、489 | +42、63、66、676 |
| 工 | +63、70、430、770；−660 | +2、63、68 |
| 詩 | +63、70、430、770；−660 | +2、63、68 |

Bloodlust（660）在 Expert 篩選裡只剩賊池還有它，而賊也沒選它。

**換人或換配裝**（I9b → I9c）：
- 15 隊的戰士全部換成 i61 berserk 雪 .399。原本是 i34 .341 或 i501+i3 .345，每隊因此 +.054～+.058。
- 遊 i501+i3／i501+i25 → i86 雪 .259，出現在戰賊忍女侍遊、戰賊忍領侍遊、詩工戰賊忍遊。
- 戰賊忍女領遊（+.080）：I9b 舊值 1.941 是補池前的值，補池後是 1.949。另外女 i59+i25 → i501+i25，領 i772+i3 → i59+i3，遊 → i86。
- 最佳隊從 2.073 變 2.126（+.053），成員組成不變。

**The Avenger 給誰**：
- 領、女拿它最好是 .294，人類。現在領 .283、女 .266，換它只多 .011～.028；戰士換它多 .054。
- 每件只有一把，解出來都給戰士。

**對甲隊**（I10，2.025）：最佳隊 2.126，+.101（+5.0%）。A 範本 10 組有 9 組不低於甲隊。

### 池邊界

邊界的定義：該職業篩選排名最後一名，也就是主手第 6、副手第 8。

- 第一輪（6/8）落在邊界的成員只有 B 範本詩工戰賊忍領的領主 i501+i25：Ivory 是領主主手的第 6 名。女武神也用 i501+i25，501 同樣是女武神主手第 6 名。
- 領、女放寬到主手前 8、副手前 10 重跑：
  - 主手新進 815、772（領），815、74（女）。
  - 副手新進 2、592。
  - 15 隊結果和放寬前逐字相同（`diff` 無差）。
  - 放寬後 501 是第 6／8 名，不在邊界。
- 女武神拿浮士德之戟（74）最好只有 .195。

### 沒有建模（會影響解讀）

- **先攻順序**：武器 Init 只進每回合的出手次數（`build.go` 的 V 有 10×Init 一項），誰先動手沒有建模。
  - The Avenger Init +2，石中劍 −5、浮士德之戟 −4。
  - 低 Init 武器實際上比這裡算的更差，所以 The Avenger 的優勢被低估。
- **疲勞**：固定為精神飽滿（fight.go:877）。
- **Backstab、裝備附加效果、升級路徑都沒有建模。**
- **Bloodlust**：
  - MG：最強用法是給賊一邊 berserk 一邊 backstab，這沒建模；它的 Attacks +1、Sword +5 也沒建模。
  - MG：「但這把上限低」，所以不開票。
- **Faust Halberd（74）**：在掉落表裡，有計算，沒排除。
- **即死有建模，其他特效沒有**：計算器讀 ITEMS +0x55（特效槽 5＝即死）並按怪物抗性折算，其他 15 槽固定為 0（`internal/calc/fight.go` 的 chance 迴圈）。
  - 即死槽值：Stiletto 10、Thieves Dagger 5、The Avenger 10、Light Sword 10、Muramasa 15、Staff of Doom 15、Zatoichi Bo 5、Cane of Corpus 15。
  - 其他非 0 特效槽（槽號從 0 起）：Thieves Dagger 槽 2＝20；Cane of Corpus 槽 1＝50、槽 2＝100；Staff of Doom 槽 1＝25、槽 3＝10；Zatoichi Bo 槽 1＝15、槽 6＝15。
  - MG 的遊玩經驗：Thieves Dagger 的另一個效果像是中毒。Zatoichi Bo 和徒手強是因為會 KO，KO 後受擊傷害加倍。
  - **槽號對照**：用 jeffludwig.com/wizardry8/weaponlist.php（2026-10-13 抓取）的 Special Powers 對照，5 把武器的值全部對得上：
    - 槽 1＝Paralyze：Zatoichi Bo 15、Staff of Doom 25、Cane of Corpus 50。
    - 槽 2＝Poison：Thieves Dagger 20（網站寫 Str. 2）、Cane of Corpus 100（Str. 10）。
    - 槽 3＝Hex：Staff of Doom 10。
    - 槽 5＝Kill：5 把都對得上。
    - 槽 6＝KO：Zatoichi Bo 15。
    - KO 是槽 6，和 `notes/re/formulas-2.md` 的推測（狀態碼 17、Earth）一致。Staff of Doom 沒有 KO，但麻痺比 Zatoichi Bo 高（25 對 15）。
  - 所以帶 KO 或中毒的武器、以及僧侶徒手，都被低估了。後續候選：KO 和麻痺對受擊傷害的效果、持續回合、抵抗式，查到之後建模（`formulas-2.md` 標 [未取得] 的那幾條）。

### 感知（SEN）優先序（MG 2026-10-13「等 muse 回來時順便算感知優先序」）

設定同 I9c，屬性集從 {力,敏,速,體} 換成 {力,敏,速,感}，24 種順序全排；同一輪另跑 {力,敏,速,體} 當基準。工作檔 `build/i9e/sen/c*-{sen,vit}.json`（不入版），每職只放 I9c 最佳隊用到的武器組（遊另加 i501+i25）。16 個工作共 1895 s。

| 職 | 體版最佳 | 感版最佳 | 差 |
|---|---|---|---|
| 戰 | .399 i61 berserk 鼠人 力速敏體 | .428 i61 berserk 哈比 力敏感速 | +.029 |
| 賊 | .417 i62+i3 妖精 力敏速體 | .440 i62+i3 貓人 力敏感速 | +.023 |
| 忍 | .409 i29+i70 妖精 力敏速體 | .430 i29+i70 妖精 力敏速感 | +.021 |
| 僧 | .280 i86 妖精 力敏速體 | .325 i81 人類 力敏感速 | +.045 |
| 領 | .283 i59+i3 人類 敏速力體 | .310 i59+i3 狼人 敏感力速 | +.027 |
| 侍 | .339 i41+i51 哈比 力敏速體 | .398 i41+i51 哈比 力敏感速 | +.059 |
| 女 | .266 i59+i25 人類 | .282 i501+i25 人類 力敏速感 | +.016 |
| 遊 | .259 i86 鼠人 | .310 i501+i25 貓人 力敏感速 | +.052 |

隊伍合計（各職取上表最佳，種族各自獨立）：戰賊忍僧侍遊 2.332、戰賊忍僧領侍 2.331（體版 2.126）、戰賊忍領侍遊 2.317、戰賊忍僧女侍 2.304、戰賊忍僧女遊 2.216。

- 每職、每個種族×武器組合都是感 > 體（+.002～+.059）。原因之一：模型取精神飽滿、不算疲勞，體的作用幾乎是 0，任何有作用的屬性換上去都會贏。感在哪條公式起作用未追 [未取得]。
- 僧在感版改選 Zatoichi Bo（i81），仍只算即死、不算 KO（見上節「未建模」）。
- MG 2026-10-13「不用太執着，sense 做完就停止」：本線到此停，不再續做最佳化。

### 最終建議（MG 2026-10-13 收尾版）

MG 條件：6 自創＋2 NPC（Umpani 領主、T'rang 忍者）；妖精只一人且當忍者；所有實際可掉落武器都算，唯一物（Light Sword、Avenger、Muramasa、Fang 等）只一把。取感知版各職最佳（上節），隊伍選戰賊忍僧侍遊 2.332（與戰賊忍僧領侍 2.331 並列，選遊把 Fang 讓給 NPC 領主）。

| 角色 | 種族 | 武器 | 值 | 創角自由點（60 扣門檻後） | 升級順序 |
|---|---|---|---|---|---|
| 戰 | 哈比 | The Avenger（berserk） | .428 | 力+14 敏+14 感+12 | 力敏感速 |
| 賊 | 貓人 | Light Sword＋Stiletto | .440 | 力+19 敏+19 感+17 | 力敏感速 |
| 忍 | 妖精 | Cane of Corpus＋Rod of Sprites | .430 | 力+3 敏+2 | 力敏速感 |
| 僧 | 人類 | Zatoichi Bo | .325 | 力+9 敏+9 感+7 | 力敏感速 |
| 侍 | 哈比 | Muramasa＋Enchanted Wakizashi | .398 | 力+9 敏+9 感+7 | 力敏感速 |
| 遊 | 貓人 | Ivory Blade＋Diamond Eyes | .310 | 力+5 敏+5 感+5 | 力敏感速 |
| NPC 領 | Umpani | Fang＋Stiletto | 人類代算 .283 | — | — |
| NPC 忍 | T'rang | Staff of Doom | 哈比代算 .284（非妖精、扣除已用唯一物後忍者第一） | — | — |

- 創角點數由 `PlanAttrs`（internal/calc/build.go:53）算：起點＝max(種族值, 職業門檻)，預算 60 扣門檻差，每項上限 ceil(預算/3)。升級每級 6 點、每項最多 3，依順序點到 100 再換下一項。
- 唯一物檢查：Avenger、Light Sword、Muramasa、Fang、Ivory Blade 各用一次；Stiletto、Diamond Eyes、Zatoichi Bo、Staff of Doom、Enchanted Wakizashi 有多個掉落來源（`scripts/loot-where.py item`）。
- NPC 的種族、屬性、等級模型沒有，值是同職業用模型種族代算，只作排序參考 [推測]。
- 仍未建模：KO、麻痺、中毒等特效（I14 擱置）、疲勞、防具。KO 建模後 Zatoichi Bo 會更高，不會翻轉僧的選擇。

#### 變體：NPC 改 Vi Domina（女）＋ RFS-81（僧）（MG 2026-10-13）

自創改戰賊忍僧侍領，前五人同上表，第六人改：

| 角色 | 種族 | 武器 | 值 | 創角自由點 | 升級順序 |
|---|---|---|---|---|---|
| 領 | 狼人 | Fang＋Stiletto | .310 | 敏+7 感+7 力+6（20） | 敏感力速 |
| NPC 女 Vi | 人類 | Ivory Blade＋Diamond Eyes | 人類代算 .282 | — | — |
| NPC 僧 RFS-81 | — | Zatoichi Bo（多來源，可與自創僧重複） | 人類代算 .325 | — | — |

- 第六人與 Vi 搶 Fang／Ivory 的三種分法（build/i9e/sen 感知版）：領 Fang＋Vi Ivory .310＋.282＝.592；遊 Ivory＋Vi Fang .310＋.274＝.585；遊 Staff of Doom＋Vi Ivory .302＋.282＝.584。取第一種。
- RFS-81 種族不在模型；雪人列 Staff of Doom（.302）勝 Zatoichi Bo（.275），其餘種族 Zatoichi Bo 勝。KO 未建模，偏向 Zatoichi Bo [推測]。
