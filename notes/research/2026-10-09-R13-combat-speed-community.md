# R13 研究：Wizardry 8 戰鬥加速的社群做法與遊戲內建設定（2026-10-09）

範圍：Steam 版 Wiz8.exe v1.2.4。只讀網頁與手冊文件；未下載、未執行任何加速工具、mod 或遊戲本體；未寫入遊戲目錄。取得日期除特別註明外均為 2026-10-09。

## 發現

- 內建「Combat Speed」滑桿：決定戰鬥節奏，作用是改變攻擊之間的延遲，見 https://mocagh.org/sir-tech/wiz8-alt-manual.pdf 。 [已驗證]
- 內建「Text Display Speed」：決定戰鬥中文字顯示速度，見 https://mocagh.org/sir-tech/wiz8-alt-manual.pdf 。 [已驗證]
- 內建「Monster Movement Speed（1x、3x、5x，預設約 1.5x）」：調整戰鬥中怪物移動動畫速率，見 http://wizardryarchives.com/downloads/w8manual.pdf 。 [已驗證]
- 內建「Verbose Combat Messages」開關：ON 時顯示最詳細戰鬥訊息（含命中部位 arms／legs 等），OFF 可減少訊息量從而加快流程，見 http://wizardryarchives.com/downloads/w8manual.pdf 。 [已驗證]
- 內建戰鬥模式分 Phased 與 Continuous：前者下完指令後按勾才開下一回合，後者自動連續開回合、可在回合進行中下下一輪指令，Continuous 整體速度可在 General 選項調，見 https://discmaster.textfiles.com/file/36655/Level_2001-11_cd1.bin/Demos/Wizardry_8/DATA1.CAB/Program_Executable_Files/Documentation/Wizardry_8_Demo_Readme.doc/Wizardry_8_Demo_Readme.pdf?contentType=application/pdf 與 https://rpgwatch.com/forum/threads/rpgs-with-turn-based-combat.1884/page-3 。 [已驗證]
- 舊作手冊另有「Delay 1（最快）到 18」訊息停留設定（出自 Wizardry 早期手冊用語），Wiz8 手冊對應項即 Text Display Speed，兩者不宜直接等同，見 http://wizardryarchives.com/downloads/archivesmanual.pdf 。 [推測]
- ini／登錄檔隱藏速度設定：所讀社群資料中無人提及 Wiz8 有隱藏 ini／registry 速度開關，唯一已知外部可調值是遊戲目錄下 `Wiz8.CFG` 內的怪物移動速度欄（遊戲內改選項即覆寫它），見 https://www.zimlab.com/wizardry/recovered/wolfie/Wiz8Fast.html 。 [未取得]
- Wiz8Fast 原理：不是改 exe，而是把 `Wiz8.CFG` 裡怪物移動滑桿值推到超過 UI 上限（玩家回報約 50x 可用、625x 近乎瞬移），只加速走路／位移動畫；須在遊戲啟動前執行，且遊戲內一改選項就會覆寫掉需重跑，見 https://www.zimlab.com/wizardry/recovered/wolfie/Wiz8Fast.html 與 https://rpgcodex.net/forums/threads/wizardry-8-first-time-playing.133152/ 。 [已驗證]
- Wiz8Fast 副作用：滑桿拉過約一半時怪物呈「pop／瞬移」而非走動，觀感詭異，見 https://www.zimlab.com/wizardry/recovered/wolfie/Wiz8Fast.html ；另有回報它主要加速 walk／transition，不動 attack／dodge／get_hit 段，見 https://www.gog.com/forum/wizardry_series/any_way_to_speed_up_combat_turns_in_wiz_8 。 [已驗證]
- Kaucukovnik Faster combat（Wiz8 Enhancements 內含）：提高全部怪物播UV格率、一般投射物速度、火箭 2 倍、Poison Spit／Stinger Thorn 加速，並由 Bloax 縮短戰鬥音效；法術未動、隊伍近戰未動（作者稱可能寫死），戰鬥者之間仍有殘留間隔（作者稱可能寫死），見 https://www.gog.com/forum/wizardry_series/any_way_to_speed_up_combat_turns_in_wiz_8 與 https://rpgcodex.net/forums/threads/wizardry-8-enhancements-a-call-to-arms.79581/page-7 。 [已驗證]
- Kaucukovnik 做法副作用：有玩家認為整體加速過頭，移動快到不自然、畫面滑稽，見 https://www.gog.com/forum/wizardry_series/any_way_to_speed_up_combat_turns_in_wiz_8 。 [已驗證]
- GameFAQs「Fast Animation Mod」（Cosmic Forge 製）：刪掉每個動畫約一半影格（重要 NPC 除外）並加倍飛彈速度，作者建議搭配 WizFast 使用，見 https://gamefaqs.gamespot.com/boards/374906-wizardry-8/66271587 。 [已驗證]
- Qusari Reforged mod v1.1：把全部飛彈加速（做法為複製全部 `*.mls` 檔改值），見 https://www.gog.com/forum/wizardry_series/any_way_to_speed_up_combat_turns_in_wiz_8 。 [已驗證]
- townltu「Even Faster」配方：每個 `.mon` 動畫檔只留一影格並設 fps 60、把 `sounds.slf` 內 combat 與 monstersound 音檔換成極短音、`mls` 檔 `[velocity]` 拉到至少 250，還不夠就再疊 Cheat Engine 全域 4x，見 https://www.gog.com/forum/wizardry_series/even_faster_wizardry_8/page1 。 [已驗證]
- Cheat Engine speedhack 對 Wiz8：有人建議全域 4x（上條同帖），但同串另有人回報 CE Speedhack 對本遊戲無效（僅精靈圖例外），兩說並存、前者需實測，見 https://www.gog.com/forum/wizardry_series/even_faster_wizardry_8/page1 。 [推測]
- DxWnd timestretch 做法：須先用 3dsetup.exe 切 DirectX7，再開 DxWnd expert 模式於 Timing 頁勾 Time Stretch；屬全域時間拉伸而非只針對戰鬥，見 https://www.gog.com/forum/wizardry_series/even_faster_wizardry_8/page1 。 [已驗證]
- 社群修改版 exe（直接改二進位的加速版）：所讀資料中無人提及此類成品，見 https://www.gog.com/forum/wizardry_series/any_way_to_speed_up_combat_turns_in_wiz_8 。 [未取得]
- Christian Coder mod：內容是職業／種族／物品／法術平衡（被評價為最接近官方升級感的 mod），無資料顯示它處理過戰鬥速度，見 https://www.gog.com/forum/wizardry_series/mods_for_wizardry_8 。 [未取得]
- dgVoodoo／nGlide 對速度影響：dgVoodoo2 在社群的角色是讓舊渲染跑在現代系統上（高解析、修 crash、修相容），無資料顯示它改變戰鬥節奏；另有獨立 low-frame-rate fix 是掛 DXPrimaryEmulation 相容旗而非加速，見 https://www.neogaf.com/threads/wizardry-viii.1650311/ 與 https://community.pcgamingwiki.com/files/file/348-wizardry-8-low-frame-rate-fix/ 。 [推測]
- 全域加速副作用：CE／DxWnd 類做法拉的是整個遊戲時鐘，探索中怪物移動、回復、遊戲時間會一起變快，屬此類工具原理上的必然連帶，見 https://www.gog.com/forum/wizardry_series/even_faster_wizardry_8/page1 ；另有 Wine 回報遊戲時鐘異常會連帶拖慢全遊戲移動，可佐證時鐘與移動共用時基，見 https://appdb.winehq.org/objectManager.php?sClass=version&iId=2849 。 [推測]
- 音效／語音副作用：加速 mod 需同步縮短戰鬥音效（Bloax 短音版、townltu 極短音替換），反證原長度音效在加速下會重疊或被截斷，見 https://rpgcodex.net/forums/threads/wizardry-8-enhancements-a-call-to-arms.79581/page-7 與 https://www.gog.com/forum/wizardry_series/even_faster_wizardry_8/page1 。 [推測]
- 訊息可讀性副作用：有玩家回報無論 Delay 怎麼設，戰鬥文字照樣一閃而過（例如看不清有無上到 Silence），加速只會更惡化，見 https://rpgcodex.net/forums/threads/bored-so-installed-wizardry-8.20982/page-5 。 [已驗證]
- 訊息延遲／動畫時間對應 exe 位址或常數：所讀社群資料無人給出具體位址；mod 作者僅以「probably hardcoded」描述隊伍近戰速度與行動間隔，未附位址，見 https://rpgcodex.net/forums/threads/wizardry-8-enhancements-a-call-to-arms.79581/page-7 ；反組譯專案存在但未見訊息延遲位址，見 https://github.com/agluszak/wizardry-8-decomp 。 [未取得]
- 玩家抱怨最多的慢段是怪物回合逐隻走位：小蟲群一人 10 秒、20 隻怪打 8 分鐘、大群 50 隻每隻等 30 秒等說法反覆出現，見 https://www.mobygames.com/game/5580/wizardry-8/reviews/windows/ 與 https://www.gamespot.com/wizardry-8/user-reviews/2200-120308/ 與 https://rpgcodex.net/forums/threads/142607/page-2 。 [已驗證]
- 次多抱怨是攻擊／閃避／受擊動畫本身：有玩家稱即使用動畫加速 mod 到滿仍嫌慢，見 https://www.neogaf.com/threads/wizardry-viii.1650311/ ；另有以最高速＋全部加速手段把一回合壓到約 7 秒的實測，見 https://steamcommunity.com/app/245450/discussions/0/4356745048752354759/?l=thai 。 [已驗證]
- 施法特效／投射物飛行被抱怨較少，且多數加速 mod 已涵蓋（加速飛彈、火箭 2x），優先順序低於怪物走位與攻擊動畫，見 https://rpgcodex.net/forums/threads/wizardry-8-enhancements-a-call-to-arms.79581/page-7 。 [推測]

## 規格疑慮

- 讀取工具不支援自訂 HTTP User-Agent，實際請求未能送出「wiz8-mod personal research (non-commercial)」字串；全程未下載可執行檔、未送出 email 或真實姓名。
