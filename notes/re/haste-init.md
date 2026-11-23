# I5 研究：Haste 與先攻（靜態分析，未執行遊戲）

範圍：只限數值；地圖、移動、投射物不碰。偏移說明：SPELLTABLES 記憶體位址 = 磁碟偏移 - 0x101。
結論摘要：先攻決定「100 tick 回合內第一次出手的時間點」，攻擊次數決定後續出手的間距；
Haste 作用於哪個欄位本次未取得（見最後幾條 `[未取得]`），下一票的 Haste 輸入須設計成可明示的覆寫值。

## 發現

- Haste 是法術表第 56 筆（Fire 領域、法術等級 4、SP/PL 8、目標型別 2、射程 0），記錄內 +0x141 欄位值 17（Slow 為 -1、Superman 為 18、Bless 為 30） [已驗證] build/data-x/DATABASES/SPELLTABLES.DBS（stride 704，欄位對照 notes/re/combat-2.md:60）
- 法術說明文字：Haste「給全隊額外能量以加快動作並抵銷 Slowed 效果」；Superman（Water 4）同時提升命中、傷害、速度與 AC；Slow 為對一群敵人降低速度 [已驗證] build/data-x/DATABASES/SPELLDESC.DBS:13273（Haste）、build/data-x/DATABASES/SPELLDESC.DBS:14641（Superman）
- Haste 法術是全隊目標（說明文字 all party members），沒有骰子欄位（Z/N/S 皆 0），即效果強度不由骰子決定 [已驗證] build/data-x/DATABASES/SPELLTABLES.DBS:39432（第 56 筆起點）
- 物品來源：ITEMS.DBS 內有 Potion of Haste、Book of Haste、Scroll of Haste 三個名稱；Book of Haste 說明為「Psionic 法術書的學習者，學會 Haste 並增加 Fire 領域點數」 [已驗證] build/data-x/DATABASES/ITEMDESC.DBS:11255（物品記錄欄位本次未解碼） 
- exe 內有一張附魔名稱指標表（視覺／圖示載入用），Haste 字串位於 Wiz8.exe:0x20d694，圖示字串 Haste.sti 位於 Wiz8.exe:0x23285c；載入函式 0x4acf10 取 [idx*4+0x60d50c]；這是圖示表而非遊戲邏輯 [已驗證] build/wiz8.asm:232034
- 法術記錄 +0x141 欄位 17／18 與圖示表索引相差 15 的對應只在 Haste、Superman、Bless 三筆相符，Armorplate 不符 [推測] build/wiz8.asm:232034（只憑數值對照，未見讀取該欄位的程式碼）
- 每角色先攻（C+0xe81）公式：floor((等級欄 C+0x89 + 1)/2) + floor(C+0x161/5) + floor(C+0x14d/5) - 10；Snake Speed 旗標 C+0x767 成立時加 floor(C+0x76d/10)+1；再加物品效果和陣列第 0 格（int8，C+0x1770）；再依負重等級 C+0xbc9 (0..4) 加 0/-1/-2/-4/-8 [已驗證] build/wiz8.asm:319348
- 上列公式中 C+0x14d 對應速度（SPD）、C+0x161 對應感知（SEN）、C+0x89 對應等級，屬依公式形狀的欄位識別 [推測] build/wiz8.asm:319348
- 社群公式 Initiative = 11 + SPD/5 + SEN/5 + Snake + floor((Level-1)/2)，與上列逐項一致（11 = 常數項 -10 加 floor 取整後的等級項換算） [推測] https://steamcommunity.com/sharedfiles/filedetails/?id=899973393
- 回合開始排程：對每個存活隊員取 x = R(4) + C+0xe81；若指令為攻擊類再加所選手的武器先攻（H+0x1d）；夾在 [-10, 89]；行動延遲 = 90 - x（存於 T+0x18） [已驗證] build/wiz8.asm:312253
- 延遲再由 0x4f0480 在 G+0x910 非零時進位到 10 的倍數、夾到 100 以內並檢查在 1..100 [已驗證] build/wiz8.asm:322000
- 戰鬥時鐘 G+0x8 為 100 tick 的回合；某行動者的延遲等於時鐘時出手，延遲小者先出手 [已驗證] build/wiz8.asm:313003
- 怪物的延遲同形：90 - clamp(R(4) + 怪物參照 +0xe4 的位元組, -10, 89) [已驗證] build/wiz8.asm:312418
- 手的挑選：0x53d390 取攻擊次數（H+0x15）較高者，平手取武器先攻（H+0x1d）較高者 [已驗證] build/wiz8.asm:431395
- 每手剩餘攻擊數由 H+0x15 載入 T+0x40+4h；出手後同一手仍有剩餘時，下一次出手時間加上 (T+0x9c - 現在 + 100)/(剩餘+1)；換手時加上兩手先攻差；結果夾到 100 並進位 [已驗證] build/wiz8.asm:315004
- 怪物（其他）出手後的間距：下一次出手時間加 (100 - 現在)/(剩餘次數+1) [已驗證] build/wiz8.asm:313003
- 攻擊次數 F08：a = tz((floor((DEX+SPD)/2) + M + P + E + K)/3)，N = 1 + [a>=50] + [a>=100]；副手門檻改用 75 [已驗證] build/wiz8.asm:319790
- 揮擊數 F09：v = tz((SPD + 10*H+0x1d + M + P + E + K)/3)，S = 1 + [v>=67] + [v>=100]；物品 +0x48（int8）在 0x4ee66a 寫入 H+0x1d，所以武器 Init 同時進「揮擊數」、「首次出手時間」與「換手間距」三處 [已驗證] build/wiz8.asm:319861
- 空手（F26）的先攻另加 floor(K/10) [已驗證] notes/re/formulas.md:1
- 結論：先攻不改攻擊次數 N；它只經過 H+0x1d 影響揮擊數 S（10 倍權重）和出手時間點；SPD 同時出現在 F08、F09 與每角色先攻，故若 Haste 是以加 SPD 實作，三者會一起變，但此對應本次未驗證 [推測] build/wiz8.asm:319790
- 社群：Haste 追不上 Slowed 時，保護類法術施放者的先攻須為最高（Haste 與出手順序有關的說法，非數值） [推測] https://steamcommunity.com/sharedfiles/filedetails/?id=899973393
- 擊殺回合規則（由排程推得）：同一 100 tick 回合內延遲較小者先動；我方延遲 < 怪物延遲時，我方第一次攻擊可在該怪物行動前結算；若首擊已殺死怪物，其行動不發生。延遲含隨機項 R(4)，且排程模式 G+0x910 會進位到 10 的倍數，同一進位格內平手的先後本次未取得 [推測] build/wiz8.asm:312253
- Haste 究竟修改哪個角色欄位（SPD、先攻、攻擊次數或行動延遲）：已在先攻、揮擊、排程與回合開始碼中查過，所見範圍內沒有直接讀 Haste 狀態的項；施法分派中對應法術 56 的處理位置也未定位 [未取得] build/wiz8.asm:312253
- Haste 的持續時間與法力等級（PL）的關係：所見範圍內未見公式；法術記錄只有 SP/PL = 8 與上述 +0x141 欄位 [未取得] build/data-x/DATABASES/SPELLTABLES.DBS:39432
- Haste 與 Slowed 的數值大小（加減多少）：未取得；說明文字只說「抵銷 Slowed」 [未取得] build/data-x/DATABASES/SPELLDESC.DBS:13273
- 職業／種族可學性（Monk、Psionic、其他）與 Gadgeteer 可製作的 Haste 道具：未解碼職業法術書表（exe 內有職業特性表，位址 0x61507c／0x615130 附近，未讀）與物品記錄 [未取得] Wiz8.exe:0x21507c
- 社群對戰鬥速度與先攻的討論（R13、R14）沒有給出 Haste 數值 [未取得] notes/research/2026-10-09-R13-combat-speed-community.md:1

## 給下一張實作票的文字規格

1. Haste 情境輸入：因機制未取得，做成「明示覆寫」而非內建數值。建議旗標 `--haste-spd +N`（對 SPD 加 N，預設 0 即不啟用），並在輸出標明「Haste 數值為假設輸入」。
   SPD 進 F08（攻擊次數）、F09（揮擊數）與先攻三處，所以 calc fight 重算這三項；其餘輸入不動。
2. 先攻不進「期望傷害」本身（它不改 N；對 S 的影響已由 F09 帶入）。
3. 先攻另列為「首擊時間」欄：我方延遲 = 90 - clamp(R(4) + Ini + 武器 Init, -10, 89)，怪物延遲 = 90 - clamp(R(4) + 怪物先攻, -10, 89)；
   只輸出兩者的期望差或比較結果，不做回合模擬；R(4) 與進位細節標「未取得」。
4. 擊殺回合數：若首擊先於怪物行動且單回合傷害足以擊殺，視為怪物未行動；此判斷放為附註欄，不改既有回合數算法。
5. 驗證路徑：Haste 實際機制要靠實機對照（MG 操作）或下一輪靜態分析（找 Haste 附魔的讀取點）補上，再把覆寫值換成內建值。

## 規格疑慮

- Haste 機制未取得，規格 1 的 SPD 加成只是占位假設。

## MG 實機對照（2026-10-12，補第 31、33 條）

MG 在戰鬥中施放 Haste 後截圖屬性面板（武僧、武士兩例），原話：「看起來是速度大幅提升，全隊施法後速度都是 125」。
- 施法後顯示 SPD 全隊＝125。武士施法前 SPD 97（notes/calc-qa/mg-party-l25.md 的面板；截圖為 L23 存檔，非同一時點）。97＋固定加成要剛好得 125，需加 28；全隊不同 SPD 都落在 125，與「設為 125」或「加成後封頂 125」相符，與「固定加 N 不封頂」不符 [推測：兩例＋MG 口述全隊，未對 SPD>125 的角色驗證]。
- 法術說明面板：法術階 4、消耗 8 法力／威力級、法術書 靈能者、可使用 戰鬥、目標 隊伍、射程 短、持續 1 回合／威力級（MG 截圖）。補第 32 條：持續時間＝PL 回合。
- 建模：Haste 情境把 SPD 覆寫成 125（取代規格 1 的 `+N` 占位）；SPD 經 F08、F09、先攻三處生效（第 28 條）。持續 PL 回合，長戰鬥要按回合數折算 [未建模]。
