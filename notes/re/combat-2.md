# P6：CTH、法術傷害、SP 回復與 DBS 欄位補完

本票只補 [P](formulas.md) F03／F06／F16–F19／F21、[P2](formulas-2.md) 的法術與抗性缺口、[P4](abilities.md) 的 SP 回復常數；成長與合法配點沿用 [P3](growth.md)。`[已驗證]` 表示本地靜態資料流及原始 bytes 相符，不表示實機量測。只讀 `build/`，未執行遊戲、下載或重產反編譯。

原檔 `build/Wiz8.exe` SHA-256：`18a74ff61c65b8a2d4cfa11ffce82ad7fef022a94eaf0c2f217e479e981420d2`。本文 .text／.rdata／.data 位址的檔偏移為 VA−0x400000；import 名稱依 PE 區段另行換算。所有簽章均直接讀原檔，沒有重新組譯。DBS 與整數欄位皆 little-endian，位移相對記錄起點；i8/u8 等表示符號性及位寬。

記號：`tz` 向零截斷，`floor` 向下取整，`f32` 表示寫回 IEEE binary32，`[條件]` 為 0/1。除特別指出的 byte／u32 回繞，算式限正常數值、不溢位。`R(n)`、`Dice(Z,N,S)` 沿用 P F01／P2「共用擲骰介面」。`C` 角色，`H=C+0x1149+0x5b*hand`，`G=*[0x6836a8]`，`T=G+0x18+0xd4*角色索引`，`Q=*[0x685178]+0x106*角色索引`；`D` 怪物模板，`M` 怪物實例，`I` 物品模板，`F` 載入後法術模板。注意 H 與 P2 表內未加 0x1149 的記號不同。

## 發現

- [已驗證] **C01 原始簽章與整數轉換證據補齊**：`0x40efa0`（前 16 bytes：`56 8b 74 24 08 85 f6 75 04 33 c0 5e c3 ff 15 d0`；`build/Wiz8.exe@0xefa0`）；`0x5e1c22`（前 16 bytes：`ff 25 14 b2 5e 00 cc cc cc cc cc cc cc cc 53 56`；`build/Wiz8.exe@0x1e1c22`）。PE import 實際將 IAT `0x5eb1d0` 指向 `MSVCRT.DLL!rand`，`0x5eb214` 指向 `MSVCRT.DLL!_ftol`，名稱位於 `build/Wiz8.exe@0x279768`／`build/Wiz8.exe@0x27980a`。這補上 P 規格疑慮第 4 條的 import 缺口；本文以該 CRT 的通常向零截斷語義解釋 `_ftol`。DLL 本體、seed、完整 RNG 呼叫歷史未附，不能宣稱逐次隨機結果可重播。`R(100)` 仍不是嚴格均勻；百分率式只供均勻近似，精確離散計算須沿用 P F01 的整數映射。來源：build/wiz8.asm:19149、build/wiz8.asm:652799。

- [已驗證] **C02 低等級 scale 的實值與截斷修正**：`0x5479b0`（前 16 bytes：`8b 44 24 04 8b 48 69 8b 84 88 8d 00 00 00 83 ec`；`build/Wiz8.exe@0x1479b0`）。`build/Wiz8.exe@0x1ee800` 的 `00 00 70 42` 是 f32 60；`build/Wiz8.exe@0x1ed490` 的 `0a d7 23 3c` 是 f32 `0.009999999776482582 = 5368709/536870912`。設 L 為當前職業等級：`F(L,m)=m`（unsigned L>20），否則 `F(L,m)=(2L+60)*f32(m)*5368709/536870912`，在 x87 中運算；呼叫處直接交 `_ftol`，不是先寫回 f32。對本票小整數 m，門檻可直接用整數式 `tz((2L+60)*m*5368709/536870912)`。**不可簡化成精確除 100 再截斷**：P 的 cc 補值所稱 L=20 連續、LS 門檻 12，需要修正。來源：build/wiz8.asm:445001、build/wiz8.asm:312021、build/wiz8.asm:437478。

  | 當前職業 L | LS：tz(F(L,12)) | 背刺：tz(F(L,100)) | feature 0x13：tz(F(L,5)) |
  |---|---:|---:|---:|
  | 1 | 7 | 61 | 3 |
  | 10 | 9 | 79 | 3 |
  | 20 | 11 | 99 | 4 |
  | 21、28 | 12 | 100 | 5 |

  LS 的資格、`R(100)<門檻`、追加主手揮擊沿用 P F02–F04；入口 `0x4e8370`（前 16 bytes：`51 a1 a8 36 68 00 8b 48 04 53 55 33 db 3b cb 56`；`build/Wiz8.exe@0xe8370`）、`0x53d870`（前 16 bytes：`8b 44 24 04 83 ec 24 83 f8 08 72 1a 6a 00 68 b4`；`build/Wiz8.exe@0x13d870`）。背刺沿用 F21：L≤20 時門檻為 `2L+59`，而非 `2L+60`；L≥21 為 100。其 `0x519ac0` 射程檢查現已補得：`0x519ac0`（前 16 bytes：`56 8b 74 24 0c 83 fe 02 72 53 74 1a 6a 00 68 f2`；`build/Wiz8.exe@0x119ac0`） 回傳停用手 −1，普通非武器分流 0，H+0 類型 1/3 則讀 I+0x47；故背刺的 ≤0 要保留，不能把所有近戰／延伸射程都列為合格。傷害入口 `0x5420b0`（前 16 bytes：`83 ec 20 8b 44 24 24 8b 15 74 51 68 00 8b c8 c1`；`build/Wiz8.exe@0x1420b0`）。能力判定入口 `0x547940`（前 16 bytes：`56 8b 74 24 08 85 f6 74 59 8b 46 69 83 f8 ff 8b`；`build/Wiz8.exe@0x147940`），職業／種族 feature 歸屬沿用 P4。

- [已驗證] **C03 職業加權等級表**：`0x4ee130`（前 16 bytes：`55 56 8b 74 24 0c 8b 46 69 57 33 ff 33 ed 83 f8`；`build/Wiz8.exe@0xee130`）；`build/Wiz8.exe@0xee210` 的 15 bytes 為 `00 00 00 00 00 00 00 01 01 01 02 02 02 02 02`，配合 `0x4ee204` 跳躍表，P F06 的倍率依職業索引 0..6／7..9／10..14 分別為 4／3／2。因此可以直接以職業歷史計算 P，不再需要把 P 當黑箱輸入。BARraw、雙持技能混合與負重仍用 P F07–F15，重算入口 `0x4ee220`（前 16 bytes：`83 ec 24 53 55 8b 6c 24 30 56 33 f6 57 89 74 24`；`build/Wiz8.exe@0xee220`）；本票不把角色總等級直接乘單一倍率。來源：build/wiz8.asm:319444、build/wiz8.asm:319790。

- [已驗證] **C04 完整 ARhit 分支（續 F16）**：`0x541c00`（前 16 bytes：`8b 0d 74 51 68 00 53 55 56 8b 74 24 10 8b c6 c1`；`build/Wiz8.exe@0x141c00`）。令 `B=i32(H+0x11)`、`J=i32(H+0x21)`、`b1=i8(C+0x1771)`、`f=Fatigue(C+0x169e)`（P F15）；H+5=7 時 `fh=floor(f/2)`，其餘 fh=f。先算 `A0=B+5J+5b1+HitMode[mode]+10[G+0x9a4≠0]+10[G+0x9a5≠0]−fh`。若 `u32(T+0x94)>0`，令 `x=max(0,SPD+B−25*u32(T+0x94)−50)`，再扣 `max(0,tz(50−x^0.75))`；否則不扣。第四參數的低 byte=0 時再加 `−5*defHit+tz(Δ/10)`，非零時整組省略。Δ 是攻方 INT+DEX+SPD+SEN 減守方同四項；原目標在 Q+0x1d，怪物讀 **M**+0x248..0x24b 四個 u8，人物讀 C+0xfd/+0x139/+0x14d/+0x161 四個 i32。若 C+0xa31≠0 且 A>10，無 feature 7 改為 10，有 feature 7 改為 `10+tz(F(L,A−10))`。最後才套 C06 的 Difficulty。這是整數輸出式；含冪次分支仍須保留浮點求值及截斷。常數 `build/Wiz8.exe@0x1ebf40`＝f64 0.75（`00 00 00 00 00 00 e8 3f`），`@0x1ee7b8`＝f64 50（`00 00 00 00 00 00 49 40`）。來源：build/wiz8.asm:436999、build/wiz8.asm:437040、build/wiz8.asm:437174。

- [已驗證] **C05 AC 如何進入命中／穿甲（續 F17）**：`0x542ee0`（前 16 bytes：`53 8b 5c 24 08 8b 03 83 f8 03 56 75 5d 8b 43 08`；`build/Wiz8.exe@0x142ee0`）；`0x542fc0`（前 16 bytes：`53 8b 5c 24 0c 55 56 57 8b 7c 24 14 53 57 e8 0d`；`build/Wiz8.exe@0x142fc0`）；`0x5468d0`（前 16 bytes：`53 55 56 8b 74 24 10 8b 06 57 33 ff 32 db 83 f8`；`build/Wiz8.exe@0x1468d0`）。怪物 `base=10−i8(D+0x15d)`，若 M+0x220≠0 先令 `base=min(base,−5)`；`defHit=base+U`。人物則 `defHit=i32(Ctarget+0xe85)+U`。怪物 `defPen=defHit+i8(M+0x1e0)−i8(D+0x16d+mode)−i8(D+0x166+loc)+i8(D+0x15d)+U`；人物 `defPen=defHit+i32(Ctarget+0xec1+4loc)−i32(Ctarget+0xe85)+U`。**U 確實加兩次**，不要去重。loc=0..6 是部位；怪物先以另一顆 R(100) 按 D+0x15f 的七個累積權重抽取。讀入的 AC 編碼越低，通常防禦越高，不能把 UI AC 直接塞進 ARhit。來源：build/wiz8.asm:438533、build/wiz8.asm:438615、build/wiz8.asm:443370、build/wiz8.asm:433174。

  U 的怪物完整分支：起值 `i8(M+0x226)+i8(M+0x1df)+floor(Fatigue(M+0x242)/10)`。戰鬥全域 byte[0x683f94]≠0 時，任一 G+0x9a4/+0x9a5 非零且 D+0xd0 的 mask 0x02 成立，扣 2；M+0x2e1=1 加 2；M+0x2e1=4、M+0x220=0 且 mode 不在 {4,7,8} 再扣 4。戰鬥外沒有後三项。

  U 的人物分支：起值 0；戰鬥中有上述任一 G 旗標時，扣 `max(Ctarget+0xe91,0)+max(Ctarget+0xe99,0)`；`0x4ed2b0()`=0 且 `0x4ed2d0(targetId,6,0)` 真時另扣 4；防禦旗標取 `0x4ed2b0()`=2，戰鬥外取 `0x41efb0()`。防禦旗標真、Ctarget+0x17b5=0 且 mode 不在 {4,7,8} 再扣 4。這些玩家姿態 helper 的語意不強行命名，計算器可接收其已解析布林值；對怪物輸出不依賴它們。

- [已驗證] **C06 模式表、CTP 與難度的運算次序**：`0x542e10`（前 16 bytes：`8b 44 24 04 56 33 f6 85 c0 8b 44 24 0c 75 4b 83`；`build/Wiz8.exe@0x142e10`）；`0x541ec0`（前 16 bytes：`8b 0d 74 51 68 00 53 8b 5c 24 08 8b c3 c1 e0 06`；`build/Wiz8.exe@0x141ec0`）；`0x55cc00`（前 16 bytes：`8b 4c 24 04 8b c1 c1 e0 06 03 c1 8d 04 40 c1 e0`；`build/Wiz8.exe@0x15cc00`）。mode 0..8 的 HitMode 為 `[0,−10,0,−30,−5,5,−10,10,0]`，PenMode 為 `[0,10,−5,−30,5,−5,0,−10,10]`。P F17 可直接實作為 `P0=B+5J+5*i32(H+0x25)+5b1+5*i8(C+0x1772)+PenMode−fp−5defPen+tz(Δactual/10)`，其中 H+5=7 時 fp=0，其他 fp=f；實際目標取 G+0x8c5。Difficulty(x) 在難度 0/1/2 為 `tz(140x/100)`／x／`tz(60x/100)`，C+0xa35>0 時交換 0、2；其它難度值原樣返回。ARhit、ARpen 分開套難度；此函式沒有另以戰鬥旗標開關。來源：build/wiz8.asm:438449、build/wiz8.asm:437222、build/wiz8.asm:474731。

- [已驗證] **C07 一揮的完整普通近戰判定（續 F19）**：`0x53e250`（前 16 bytes：`6a ff 68 9b 73 5e 00 64 a1 00 00 00 00 50 64 89`；`build/Wiz8.exe@0x13e250`）。適用 C02 的射程回值<2、有效目標且非 C10 特殊物品。先取得 C04 的 A，再擲 `r=1+R(100)`，保存 low=(r≤5)、high=(r>95)。先執行 C08 的失手轉向；沒有轉向時，high 直接 miss，否則 `hit=low 或 r≤A`；hit 後以同一 r 判斷 `penetrate=low 或 r≤ARpen`，成功才交 P F20–F23 傷害。所以普通原目標的明確整數判斷是 `hit=(r≤5) 或 (r≤95 且 r≤A)`，`damageGate=(r≤5) 或 (r≤95 且 r≤A 且 r≤ARpen)`，但必須先排除失手轉向。**不是 CTH 機率乘獨立 CTP 機率**。無失手時，均勻近似 hit%=`clamp(A,5,95)`、damageGate%=`clamp(min(A,ARpen),5,95)`。`0x543110` 只在 miss 後分類／呈現格擋等結果，沒有第二次命中骰；其簽章 `0x543110`（前 16 bytes：`8b 54 24 10 53 56 8b 74 24 14 2b f2 8b 54 24 10`；`build/Wiz8.exe@0x143110`）。來源：build/wiz8.asm:432824、build/wiz8.asm:433384、build/wiz8.asm:438745。

- [已驗證] **C08 高骰不一定只 miss：失手可換目標**：失手名單入口 `0x543dc0`（前 16 bytes：`51 53 8b 1d f4 ba 5e 00 55 56 33 c0 57 33 ff 89`；`build/Wiz8.exe@0x143dc0`），消費者 C07。以第四參數=1 重算 `Af`（省略目標防禦和屬性差，但保留盲、難度等），名單包含自己及可達的其他合格目標、排除原目標與已死亡目標。令 k=名單長度；Af≥100 時 f=0，否則 `z=tz((100−Af)^3*float64(1e−5)+0.5)`，`f=clamp(floor(k*z*10/100),0,100)`。race=15 且其 Q+0xfa 經 0x50b800 取得物件非空、物件+0x2e=0x20、0x506280(0x44) 為假時，f 在 clamp **之後**再加 5。實際比較是 `r>u32(100−f)` 且 r>5；須保留 unsigned 下溢，f>100 不能擅自當 100%。觸發後 R(k) 選新目標，令 forceHit=true、forcePen=false、high=false，清 G 的兩個突襲旗標，**重擲** `r'=1+R(100)`；命中不再比較 A，但穿甲比較 r'≤新目標 ARpen，即使 r'≤5 也沒有普通低骰的穿甲保證。來源：build/wiz8.asm:432844、build/wiz8.asm:439804。

  常數實讀：`build/Wiz8.exe@0x1ee7b0` 的 `00 00 00 00 00 00 08 40`＝f64 3；`@0x1ee7a8` 的 `f1 68 e3 88 b5 f8 e4 3e`＝f64 約 0.00001；`@0x1ebe80` 的 `00 00 00 00 00 00 e0 3f`＝f64 0.5。沒有特殊 race 分支時，原目標的 gate 可以合寫 `r≤5 或 (r≤95 且 r≤A 且 r≤ARpen 且 r≤100−f)`；這只是有傷害資格，還需扣吸收及處理中斷。

- [已驗證] **C09 每揮重擲，不是整個 attack 共用骰**：C07 函式尾端在 `0x53f489` 起遞減 T+0x68；尚有揮擊時重設動畫／T+0xa5 並返回 2，下一次真正結算再經 `0x53e50f` 的 RNG。動畫準備返回不擲命中骰；同一揮內 CTH/CTP 共骰，不同揮各自重擲。LS 仍是 P F03/F04 所述先抽一次旗標、一次追加 3 或 4 揮，追加的各揮再走同一結算。結算入口 `0x53e250`（前 16 bytes：`6a ff 68 9b 73 5e 00 64 a1 00 00 00 00 50 64 89`；`build/Wiz8.exe@0x13e250`）；caller `0x4e91fd` 呼叫 0x53e250。來源：build/wiz8.asm:312923、build/wiz8.asm:433920、build/decomp/p6.c:2090。

- [未取得] **C10 遠程命中不能套近戰百分式，完整幾何仍有缺口**：`0x544b60`（前 16 bytes：`81 ec 80 00 00 00 8b 15 6c 51 68 00 53 55 56 8b`；`build/Wiz8.exe@0x144b60`） 把 C04 的 ARhit 交 `0x544630`（前 16 bytes：`83 ec 20 8a 44 24 34 53 55 56 33 ed 84 c0 57 c6`；`build/Wiz8.exe@0x144630`），後者由 `0x5454c0`（前 16 bytes：`55 8b ec 83 e4 f8 83 ec 4c 53 8b 5d 08 56 57 8b`；`build/Wiz8.exe@0x1454c0`） 依距離、盲狀態、R(360) 方向及百分骰改變瞄準位置，再建立實體投射物。已讀到 0x545899；它不是單純 `r≤ARhit`。C07 的射程≥2 分支消費 G+0x8c4：=1 視為已命中，另擲 1+R(100) 做穿甲；feature 5、H+9=17、H+5=9、第一裝備 I+0x3f≠13 同時成立才強制穿甲。=2 是另一路已處理結果，不能當新骰命中；其他值為 miss。C+0xfa5=0x272 還有專用分支，直接依 G+0x8c4 跳到尾端。碰撞、遮擋、散布到命中機率、該特殊物品的全效果尚未串完；計算器須接收外部「已命中／已處理／miss」事件，不得以近戰 clamp 冒充完整遠程 CTH。來源：build/wiz8.asm:440925、build/wiz8.asm:440780、build/wiz8.asm:441715、build/decomp/p6.c:1370。

- [已驗證] **C11 法術模板來自 SPELLTABLES，不是 SPELLEFFECT**：載入入口 `0x4acc10`（前 16 bytes：`a1 1c be 65 00 83 ec 08 56 57 33 ff 3b c7 74 15`；`build/Wiz8.exe@0xacc10`） 讀 `Data\Databases\SpellTables.dbs`，檔頭兩個 u32 是記錄數 150、版本值 1；每筆先略過 0x101 bytes，再讀 0x1bf bytes。故磁碟 stride=`0x2c0`、第 i 筆起點 `r=8+0x2c0*i`、記憶體 F+o 對應檔偏移 `r+0x101+o`。檔長 105608，恰為 `8+150*704`；runtime stride 才是 0x1bf。`build/dbs/DATABASES/SPELLTABLES.DBS@0x0` 的前 16 bytes：`96 00 00 00 01 00 00 00 06 4e 6f 6e 65 00 00 00`。SPELLEFFECT.DBS 的開頭及檔長不符合此載入鏈，不能用它切 447-byte 模板；本票沒有解析它的全部格式。來源：build/wiz8.asm:231773、build/wiz8.asm:231850。

  | runtime 位移 | 磁碟記錄內位移 | 型別、語意 | exe 消費處 |
  |---|---|---|---|
  | +0x49 | +0x14a | i32，每 PL 的 SP 成本 | 0x4fa515／0x4fab7d；C12 |
  | +0x51..54 | +0x152..155 | i16 Z、u8 N、u8 S：Z+NdS | 0x4fbd07／0x517970 |
  | +0x56 | +0x157 | i32，法術等級；不是 PL 上限 | 0x4ff790；P2 強度式 |
  | +0x9b | +0x19c | UTF-16LE 顯示名，保留區 128 bytes | 0x4fa598 |
  | +0x12f | +0x230 | i32，射程枚舉，非 AoE 半徑 | 0x4fdc4f／0x51aa30；C15 |
  | +0x133 | +0x234 | u32，領域索引 | 0x4fa50a／0x54e950 |
  | +0x137 | +0x238 | i32，目標類型 | 0x4ac9d0；C15 |
  | +0x13b | +0x23c | i32，使用情境類欄位；不可當 PL cap | 0x5001e0／0x5dc022 |

- [已驗證] **C12 Power Cast 到一般直接施法的完整量加成**：`0x4fa4d0`（前 16 bytes：`83 ec 68 8b 0d 74 51 68 00 53 8b 5c 24 74 55 8b`；`build/Wiz8.exe@0xfa4d0`） 讀 C+0x6cf 啟用旗標，未啟用 b=0；啟用則 `b=1+floor(u32(C+0x6d5)/4)`，於 0x4fab67 作第五參數交 `0x4fb4c0`（前 16 bytes：`6a ff 68 48 6f 5e 00 64 a1 00 00 00 00 50 64 89`；`build/Wiz8.exe@0xfb4c0`）。這是 P3 的技能索引 35：`0x61e454+2*35=0x61e49a`，`build/Wiz8.exe@0x21e49a` 的 `05 03` 為文字 id 773，對照 `translations/stringdata.json:766` 的 Power Cast；因此此次也補上名稱證據。先以 P2「法術基礎強度」取得 S0（入口 `0x4ff790`（前 16 bytes：`8b 4c 24 08 8b 15 1c be 65 00 8d 04 cd 00 00 00`；`build/Wiz8.exe@0xff790`）），依 spell id 扣 d 再下限 0：id 0x46 扣 2、0x57 扣 4、0x5a/0x5d/0x5e 扣 6，其餘不扣；`S1=max(0,S0−d)`，`S=S1+floor(S1*floor(b/2)/100)`。E+0x24 保存完整 b；一般傷害骰為 `Z+((N*PL) mod 256)dS`，擲出 v 後量加成 `v'=v+floor(v*b/100)`（9999 保留），再用 C14 抗性削減。**技能 100 給 b=26，即量 +26%，強度 +13%，不是兩者都翻倍。** SP 扣量路徑讀同一 F+0x49 乘 PL，交 0x52b6d0 扣對應領域。來源：build/wiz8.asm:337211、build/wiz8.asm:337721、build/wiz8.asm:339134、build/wiz8.asm:343920。

  扣強度跳躍表已讀 `build/Wiz8.exe@0xfcfd0`／`@0xfcfe0`，不是憑 spell 名稱猜測。P2 的跨職 casterLevel 算法仍必須保留；此處 S0 的 L 是該有效施法等級，不是無條件人物總等級。傷害輸出入口 `0x54e950`（前 16 bytes：`83 ec 30 8b 15 1c be 65 00 53 55 56 57 8b 7c 24`；`build/Wiz8.exe@0x14e950`） 逐目標呼叫骰及抗性，AoE 不是全目標共用一份骰。

- [已驗證] **C13 三個攻擊法術的磁碟例子**：讀 C11 的磁碟映射，並由 C12/C14 消費鏈確認；骰入口 `0x517970`（前 16 bytes：`53 56 8b 74 24 0c 8a 46 02 33 db 84 c0 57 0f bf`；`build/Wiz8.exe@0x117970`）。以下 id 是零起算的記錄索引，成本是每 PL，不整表複製。PL7、無 PC、抗性前的 Energy Blast／Mind Stab／Fireball 分別為 7d7／1+7d7／14d4，SP 為 14／21／42；PC100 時各自對擲出總量加 floor(26v/100)，不是把基礎骰面數乘 1.26。來源：build/wiz8.asm:378002、build/wiz8.asm:339134、build/wiz8.asm:337721。

  | id／英文名 | 磁碟記錄起點／F 起點 | 骰 bytes 與 (Z,N,S) | SP/PL；法術等級 | realm／range／type |
  |---|---|---|---|---|
  | 5 Energy Blast | `build/dbs/DATABASES/SPELLTABLES.DBS@0xdc8`／0xec9 | +0x152：`00 00 01 07`＝(0,1,7) | 2；1 | 0／2／3 |
  | 10 Mind Stab | `build/dbs/DATABASES/SPELLTABLES.DBS@0x1b88`／0x1c89 | +0x152：`01 00 01 07`＝(1,1,7) | 3；1 | 4／2／3 |
  | 36 Fireball | `build/dbs/DATABASES/SPELLTABLES.DBS@0x6308`／0x6409 | +0x152：`00 00 02 04`＝(0,2,4) | 6；3 | 0／3／6 |

- [已驗證] **C14 六領域抗性與傷害的取整次序**：`0x552250`（前 16 bytes：`55 8b 6c 24 08 81 7d 00 0f 27 00 00 56 57 0f 84`；`build/Wiz8.exe@0x152250`），只補 P2 抗性式到 C12 的連接。j=F+0x133；怪物 `Rj=u8(D+0x176+j)+i8(M+0x217+j)`、`Lt=u8(D+0x24f)`，玩家用 P2 的 C+0xee1+16j 及 C+0x89。`e=Rj+3*(Lt−S)`；v'=9999 或 e≤0 時不削減。否則 `m=e+1`、`h=floor(m/2)`、`FR=min(100,h+R(m−h)+R(h))`，`damage=v'−floor(v'*FR/100)`。不能改成 floor(v'*(100−FR)/100)，低傷害有一點之差；此傷害削減也沒有狀態抵抗的 5–95 clamp 或 Difficulty 呼叫。六領域索引 0..5 為火、水、風、土、Mental、Divine，`build/Wiz8.exe@0x21e518` 的 u16 名稱 id 779..784 對應 `translations/stringdata.json:772` 起；模板與即時抗性必須分開。來源：build/wiz8.asm:459320、build/wiz8.asm:454463。

- [已驗證] **C15 目標／射程枚舉已可讀，射程不是半徑**：`0x4ac9d0`（前 16 bytes：`8b 44 24 04 8b 15 1c be 65 00 8d 0c c5 00 00 00`；`build/Wiz8.exe@0xac9d0`） 消費 F+0x137，參數旗標還可能將 type 1 正規化為 0；`build/Wiz8.exe@0x20d4b4` 的 u16 文字 id 798..807 與現有文字表對照，type 0..9 為施法者、單一友方、隊伍、單一敵人、敵群、錐形、範圍、所有敵人、定點、冒險。`0x51aa30`（前 16 bytes：`8b 44 24 04 83 ec 08 56 33 f6 40 83 f8 04 57 8b`；`build/Wiz8.exe@0x11aa30`） 以 range=−1/0/1/2/3 映射距離基數 0/2/4/25/50，乘 f32 500（`build/Wiz8.exe@0x1ebc40`：`00 00 fa 43`），再加施法者端的身形／碰撞尺度：人物取全域物件+0x174，怪物取 M 所連模型+0x18c，其他來源不加。0x4fdc4f 呼叫此射程 helper；PL 不是此 helper 的參數。距離是內部座標單位，尚不能直接標成公尺。來源：build/wiz8.asm:231534、build/wiz8.asm:382575、build/wiz8.asm:341510、translations/stringdata.json:791。

- [未取得] **C16 每法術 PL 上限欄位、範圍幾何與非直接施法**：C11 的資料沒有證據支持獨立「PL cap」欄位；不能把 F+0x56 法術等級或 F+0x13b 情境誤命名。已讀 `0x500357..0x50043b` 的選 PL 迴圈及最後 1..7 檢查，直接施法端也把參數 8 作自動選擇路徑；這支持一般選擇 PL1..7，不證明每筆都有 cap 欄位。該檢查點 `0x500433`（前 16 bytes：`83 fe 01 72 05 83 fe 07 76 1a 6a 00 68 bf 10 00`；`build/Wiz8.exe@0x100433`）。AoE 的圓錐角、球半徑、遮擋及逐目標名單生成停在 0x4fb4c0／0x4fdc4f 到 0x54e950 的交界，C15 不能代替它；計算器先接收命中名單。C12 只覆蓋角色直接施法，Music／Engineering／物品等呼叫端的 b、完整施法失敗率與反噬重導向仍沿用 P2 的未取得標記，不能套 PC 公式。來源：build/wiz8.asm:345027、build/wiz8.asm:337211、build/wiz8.asm:339134。

- [已驗證] **C17 SP 基礎回復讀最大 SP，不直接讀等級或屬性**：`0x502b50`（前 16 bytes：`83 ec 08 8b 54 24 0c 8b 82 0d 0b 00 00 53 33 db`；`build/Wiz8.exe@0x102b50`）。j=0..5，`MaxSPj=u32(C+0xb25+4j)`；MaxSPj=0 時 ratej=0，否則 `ratej=f32((MaxSPj*a+20)*c)`，若 byte C+0x17b4 非零再作 `ratej=f32(ratej*1.5)`，存 C+0xb79+8j。實讀常數：a 為 f32 `0.6499999761581421`（`build/Wiz8.exe@0x1ed7e8`：`66 66 26 3f`），20（`@0x1ed4f0`：`00 00 a0 41`），c 為 f32 `0.004166666883975267`（`@0x1ed7ec`：`89 88 88 3b`），1.5（`@0x1ec3b8`：`00 00 c0 3f`）。實數近似為 `(0.65*MaxSP+20)/240`，但精確模擬須在上述位置寫回 f32。本函式沒有額外 PIE／VIT／人物等級項；這些只能透過上游最大 SP 間接影響，最大 SP 的完整生成不是本式。來源：build/wiz8.asm:348550、build/wiz8.asm:348591。

- [已驗證] **C18 SP 倍率與分數累積（補 P4 的 0x1a／0x1b）**：`0x503100`（前 16 bytes：`83 ec 28 8b 0d 74 51 68 00 53 8b 5c 24 34 55 8b`；`build/Wiz8.exe@0x103100`）。先取 q0：byte[0x683fc5]≠0 為 1；否則 dword[0x687509]=3 且非戰鬥為 0.5；其餘 0。C+0xa09≠0 時 q0 乘 0.25；SP 用 `q=max(q0,0.5)`，所以這項降低 HP 回復的狀態不會使 SP 倍率小於 0.5。feature 0x1a 令 q 乘 **1.25**，feature 0x1b 再乘 **0.75**；常數 `build/Wiz8.exe@0x1ec340`：`00 00 a0 3f`，`@0x1ebccc`：`00 00 40 3f`。模式常數亦實讀：`build/Wiz8.exe@0x1ebb34`：`00 00 00 00`＝f32 0；`build/Wiz8.exe@0x1ebb38`：`00 00 80 3f`＝f32 1；`build/Wiz8.exe@0x1ebc7c`：`00 00 00 3f`＝f32 0.5；`build/Wiz8.exe@0x1ebcc8`：`00 00 80 3e`＝f32 0.25。兩 feature 都有時依序相乘；歸屬依 P4 種族表，不因職業等級而 scale。來源：build/wiz8.asm:348979、build/decomp/p6.c:4190。

  每次收到 dt 個 tick，當目前 SP<MaxSP，`accj=f32(dt*ratej*q+accj)`，`gain=tz(accj)`，增加後封頂 MaxSP，再 `accj=f32(accj−tz(accj))`；acc 在 C+0xb7d+8j。已滿時不增加，最後清 acc。C+0x177b 的 i8 裝備彙總回復另外逐領域處理：正值先加 `bonus*dt`；負值呼叫扣 SP helper 的量是 `−bonus`，這個分支沒有乘 dt，不能擅自對稱化。當前 SP 在 C+0xb45+4j。因此 MaxSP=100、普通戰鬥 q=0.5 時，基礎平均約 0.177083 SP/tick，而非每回合固定取整成 0。

- [已驗證] **C19 回復的時間單位：遊戲時間 tick，戰鬥一次結算推進一 tick**：`0x502010`（前 16 bytes：`8b 0d 4c 6a 68 00 83 ec 0c 53 56 8b 74 24 18 03`；`build/Wiz8.exe@0x102010`） 累加時間餘數，每 120000 內部單位送一 tick 給 `0x502d00`（前 16 bytes：`d9 05 05 75 68 00 83 ec 24 d8 05 01 75 68 00 d8`；`build/Wiz8.exe@0x102d00`），再對 Q+0 有效、且主狀態 C+0xb01<18 的角色呼叫 C18；主狀態≥18 僅 C+0xa49=0 且 0x5248a0(id,1) 成立才例外呼叫。byte[0x685071]≠0 可略過這次 tick 分派，必須保留上游暫停條件。`0x482a20`（前 16 bytes：`55 8b ec 83 e4 f8 83 ec 54 56 68 30 69 4d 00 6a`；`build/Wiz8.exe@0x82a20`） 以 86400000 內部單位折一天；`0x4e9c34` 的戰鬥結算固定送 120000。普通時間更新讀 IAT 0x5eb02c 的 `KERNEL32.DLL!GetTickCount`（名稱 `build/Wiz8.exe@0x2791ca`），乘 `build/Wiz8.exe@0x20a390` 的 f32 12（`00 00 40 41`）後推進。故通常未暫停、倍率未改的實時路徑約 10 秒為一 tick，遊戲曆為兩分鐘；戰鬥結算一回合推一 tick。這不是以「移動一步」為參數；休息／加速／暫停要使用實際傳入時間，不把 10 秒當所有情境的固定牆鐘時間。來源：build/wiz8.asm:347696、build/wiz8.asm:348680、build/wiz8.asm:174865、build/wiz8.asm:313728、build/wiz8.asm:174474。

- [已驗證] **C20 ITEMS.DBS 記錄格式與武器欄位**：載入器 `0x54a400`（前 16 bytes：`83 ec 40 53 56 57 68 a4 e0 61 00 68 9c e0 61 00`；`build/Wiz8.exe@0x14a400`） 讀 u32 count 後連續讀 0x10d-byte 記錄；原檔 819 筆、220315 bytes=`4+819*269`。第 i 筆在 `4+0x10d*i`，runtime I 直接對應該記錄，無 SPELLTABLES 的額外前綴。`build/dbs/DATABASES/ITEMS.DBS@0x0` 前 16 bytes：`33 03 00 00 44 00 61 00 67 00 67 00 65 00 72 00`。讀者包括 C03 的 0x4ee220、C02 的 0x519ac0、`0x542ca0`（前 16 bytes：`53 55 8b 2d f4 ba 5e 00 56 8b 74 24 14 57 8b 7c`；`build/Wiz8.exe@0x142ca0`）、`0x51d610`（前 16 bytes：`8b 44 24 08 8b 15 6c 51 68 00 53 8d 0c 40 55 8b`；`build/Wiz8.exe@0x11d610`）。來源：build/wiz8.asm:448671、build/wiz8.asm:319860、build/wiz8.asm:438300、build/wiz8.asm:386663。

  | I／磁碟記錄內位移 | 型別 | 欄位與讀取處 |
  |---|---|---|
  | +0x46 | u8 | 武器技能分流；0x4ee220 |
  | +0x47 | u8 | 射程枚舉；0x519ac0；影響近／遠程及背刺 |
  | +0x48 | i8 | 攻速修正，0x4ee66a 納入 H+0x1d；P F09 刷新先後仍须保留 |
  | +0x49 | i8 | 命中加成，0x4ee675 納入 H+0x21；特定武器／彈藥分支會相加 |
  | +0x4a | i16 Z,u8 N,u8 S | 傷害 Z+NdS；0x5420b0／0x544b60，沿用 F20 |
  | +0x4e | u16 mask | 允許攻擊模式；0x542ca0 的 0x542d24 讀取；bit 0..8 對應 C06 |
  | +0x76 | u16 mask | 需求／允許職業；0x51d610 以當前 C+0x69 索引測 bit，不是職業歷史聯集 |
  | +0x78 | u32 mask | 允許種族；同讀者 |
  | +0x7c | u8 mask | 允許性別；同讀者 |
  | +0x7d..80 | 兩組 u8 id,u8 requirement | 屬性需求；同讀者 |
  | +0x81..84 | 兩組 u8 id,u8 requirement | 技能需求；同讀者 |

  模式選擇先抽 R(9)，循環掃允許的 bit 計數到該序位；對非空、只用 bit0..8 的正常 mask 等價於選第 `R(9) mod popcount(mask)` 個 set bit（零起算）。因此不能預設所有允許模式等機率。裝備合法性還有其他條件，職業 mask 不是充分條件。

- [已驗證] **C21 MONSTERS.DBS 的 HP／AC／抗性與閃避相關資料**：載入器 `0x54a760`（前 16 bytes：`83 ec 40 53 55 56 57 68 a4 e0 61 00 68 b4 e0 61`；`build/Wiz8.exe@0x14a760`）；原檔 595 筆、394489 bytes=`4+595*0x297`，第 i 筆 `4+0x297*i`。`build/dbs/DATABASES/MONSTERS.DBS@0x0` 前 16 bytes：`53 02 00 00 4c 00 61 00 79 00 20 00 42 00 72 00`。建立實例入口 `0x4e3930`（前 16 bytes：`53 55 56 57 68 25 04 00 00 ff 15 a0 b1 5e 00 8b`；`build/Wiz8.exe@0xe3930`） 在 0x4e39d2 以模板 D+0xd6 呼叫骰函式，寫 M+0x27 最大 HP、M+0x2b 目前 HP；不是單一固定 HP 欄位。防禦、抗性讀者見 C05/C14。來源：build/wiz8.asm:449021、build/wiz8.asm:305451、build/wiz8.asm:438533、build/wiz8.asm:443370、build/wiz8.asm:459320。

  | D／磁碟記錄內位移 | 型別 | 意義與讀取 exe 位址 |
  |---|---|---|
  | +0xd1..d5 | u8[5] | 五項屬性基值；0x4e5d00；命中使用後四項，見 C22 |
  | +0xd6 | i16 Z,u8 N,u8 S | HP 骰；0x4e39d2 → 0x517970 |
  | +0xda | 同上 | stamina 骰；0x4e39de，寫 M+0x2f/+0x33 |
  | +0x15d | i8 | 基礎 AC 編碼；0x542ee0、0x542fc0 |
  | +0x15f | u8[7] | 被擊部位百分權重；0x53ea65 一帶累積抽樣 |
  | +0x166 | i8[7] | 部位 AC 修正；0x542fc0 |
  | +0x16d | i8[9] | 攻擊模式 AC 修正；0x542fc0 |
  | +0x176 | u8[6] | 六領域基礎抗性；0x552250 |
  | +0x24f | u8 | 模板等級；0x552250 |
  | +0xd0 mask 0x02 | bit | 突襲防禦及 miss 分類相關；0x5468d0／0x543110；不另命名成 Dodge 技能 |

  M+0x226／+0x1df（i8 命中防禦修正）、+0x1e0（i8 穿甲防禦修正）、+0x217+j（i8 抗性修正）、+0x242（疲勞量）、+0x220（狀態旗標）、+0x2e1（姿態）都是 **runtime**，不是 DBS 位移。尚未證明存在獨立「閃避技能」磁碟欄位；計算器應按 C04–C06 的 AC、屬性和即時修正計算。

- [已驗證] **C22 怪物四屬性的模板來源及即時修正已補得**：`0x4e5d00`（前 16 bytes：`53 55 56 57 33 ed 8b 74 24 14 85 f6 75 19 56 68`；`build/Wiz8.exe@0xe5d00`） 依 M+0x08 的模板 id 取得 D，迴圈 i=0..4 讀 `u8(D+0xd1+i)`，加 `i8(M+0x1e7+a[i])`，clamp 到 1..125，寫 `u8(M+0x247+i)`。`a=[0,1,4,5,6]` 由 `build/Wiz8.exe@0xe5e3c` 五個跳躍目標配正文確認；與 P3 的屬性順序相容。故 C04 的四項和可以直接寫 `Σ(i=1..4) clamp(u8(D+0xd1+i)+i8(M+0x1e7+a[i]),1,125)`：磁碟 **D+0xd2..0xd5** 才是這四項基值，動態修正對應 M+0x1e8/+0x1eb/+0x1ec/+0x1ed，不能讀 D+0x248..0x24b。其餘狀態及快取仍須從事件更新或顯式輸入。來源：build/wiz8.asm:308571、build/wiz8.asm:308613、build/wiz8.asm:437122。

## 計算器介面

承接 P「規格疑慮」第 5 條：先實作已驗證核心，未知量作顯式輸入或明確不支援；每個輸出附支持範圍，不將條件式結果標成全遊戲精確排名。

| 介面／輸入 | 已可精確的輸出 | 近似、待補或不支援 |
|---|---|---|
| `loadItem(id)`／`loadMonster(id)`／`loadSpell(id)`；三個只讀 DBS | C11/C20/C21 的磁碟解析、骰／成本／mask／AC／抗性 | 不解析未列欄位；怪物屬性按 C22 加即時修正；即時狀態不是磁碟值 |
| `scale(currentClassLevel,m)`、15 職業歷史及 feature | C02/C03 的整數門檻與加權等級；L28 LS 門檻12、成功追加主手4揮 | LS 資格、中斷沿用 F02/F04；實際抽中不是固定12%輸出 |
| `attackValues(attacker,hand,mode,originalTarget,actualTarget,loc,context)`；BARraw、J、H+0x25、四屬性、疲勞、T+0x94、盲／突襲／難度、C05 動態防禦 | C04–C06 的 ARhit／ARpen，取整與分支順序 | pow／x87 精度要依 C01；原始屬性到所有快取的刷新仍有缺口 |
| `resolveMeleeSwing(A,P,r,fumbleTargets,context)`；可注入 R(n) 序列 | C07–C09 的 miss／hit未穿甲／傷害資格／失手換目標；每揮重擲 | 精確重播需實際 RNG state 與全部呼叫次序；均勻機率只標近似 |
| `physicalDamage(handDice,mode,bonuses,targetAbsorption,backstabInputs)` | 沿用 P F20–F23，加 C02 的低等級背刺門檻 | 不把每回合攻擊預算直接當保證打出的次數；目標死亡、移動、狀態中斷外部輸入 |
| `spellDamage(id,PL,casterLevel,powerCastEnabled,powerCastValue,targets,R)`；每目標等級、六抗性含即時修正 | C12–C14 的一般直接攻擊法術單目標／給定名單傷害、每 PL 成本 | 成功施法條件式；施法成功率、非直接施法、完整 AoE 幾何及特殊法術分流不支援；PL一般域1..7，沒有偽造每筆 cap |
| `regenSP(maxSP[6],currentSP[6],fraction[6],ticks,flags,features,equipmentRegen)` | C17–C19 的 f32 基礎率、倍率、累積、封頂 | 不直接輸入「步數」；時間加速及暫停由上游換成 ticks；最大 SP 本身另算 |
| `rangedEvent(projectileResult,penetrationRoll,context)` | C10 已落點後的穿甲分流 | 幾何 CTH、彈道遮擋與特殊物品0x272，不用近戰機率代替 |

期望值模式應列「普通單次有效近戰揮擊」「成功直接施法、指定 PL／目標數」「N 個 tick 的可持續 SP」等可比較量；傷害抽樣與抗性逐次取整，不能只把骰期望代入非線性公式。若以 L28 比職業，還須共同指定敵人／抗性／AC、武器可取得集合、配點與技能、buff、難度、雙持、戰鬥長度、轉職歷史、施法成功及群體命中數；目前不足以無條件宣稱某職業全情境最高 DPS。

## 規格疑慮

1. 驗收腳本只檢查 `## 發現` 每個 `- ` 的三態標記與 `path:line`／URL，不驗證公式、raw bytes 或研究完整度；其來源正則不接受本票規格允許的單獨 `path@0x偏移`。本文同時附 asm 行號及原始檔偏移，未修改腳本。
2. 「完整 CTH」目前對普通近戰及其失手分支成立；遠程實體投射物的散布／碰撞全式仍未取得（C10）。個別法術 PL cap 欄位、AoE 幾何與非直接施法仍按允許格式標未取得（C16），未填猜值。
3. 原票要求「可直接實作的整數式」，但引擎本身含 x87、pow 與 f32 累積；本文保留實際浮點常數及截斷點，沒有假造全整數等價式。C02 的低等級小整數門檻可以有理整數式實作；CRT import 已取得，DLL 本體及執行時浮點環境未量測。P cc 補值把 f32 0.01 寫成精確 /100 導致 L20 門檻差一，本票已明確修正。

## cc 抽查（2026-10-08，本機一次性 python -I 讀 build/ 原檔，工具不入版）

- **C02 常數與 LS 門檻** [已驗證：cc 讀 Wiz8.exe@0x1ee800／@0x1ed490 為 f32 60／0.009999999776]：以 f32 常數算 `tz((2L+60)·12·c)` 得 L=1→7、10→9、19→11、20→11；L=20 是 11 不是 12，P 的 cc 補值（formulas.md「推測：Ftoi 視為向零截斷」條）照本條更正。函式前 16 bytes `0x5479b0` 與本文相同；exe SHA-256 前綴 `18a74ff61c65b8a2` 相同。
- **C03 職業倍率表** [已驗證：cc 讀 Wiz8.exe@0xee210]：15 bytes `0×7,1×3,2×5` 相同。
- **C11／C13 SPELLTABLES** [已驗證：cc 以 stride 704、名稱 +0x19c 解析]：檔長 105608＝8+150·704；id 5／10／36 名稱為 Energy Blast／Mind Stab／Fireball，骰 bytes、SP/PL、法術等級與表相同。
- **C17／C18 SP 常數** [已驗證：cc 讀 Wiz8.exe@0x1ed7e8／@0x1ed7ec／@0x1ec340／@0x1ebccc]：0.65、0.0041666669、1.25、0.75 相同。
- **C20 物品即死欄** [已驗證：cc 讀 ITEMS.DBS +0x55]：Cane of Corpus／Muramasa／Staff of Doom 15、Bundle of Peacemakers 20、Strong Bow 6、Bundle of Lightning Bolts 8、Rod of Sprites 0、Tripleshot Crossbow 0。
- C04–C10、C12、C14 的控制流 [推測：cc 未逐條重讀 asm]。
