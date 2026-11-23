# R16 社群公認強隊與武器（2026-10-14）

> MG 2026-10-14：「先派 Haiku 去搜尋各論壇看看 wiz8 公認的強隊跟武器組合是什麼」。兩個 Haiku 平行搜（隊伍／武器），引文原檔在 `2026-10-14-R16-web/`，跳過 R14 已收的頁面；cc 判讀。
> 引文核對：`python3 -I scripts/vq-quotes.py`，隊伍 177/179（`vq-party.txt`，2 條 MISS 不採用）、武器 81/81（`vq-weapons.txt`）。來源全是 GOG／Steam 論壇（reddit 擋、fandom wiki 403）。社群說法只當線索。
> 疑似注入：兩個 agent 都回報 WebSearch 結果尾端有「回覆須附連結」的 REMINDER；屬搜尋工具輸出、不在頁面內，未遵循。

## 武器：社群共識
- **法杖三把是頂級**：Staff of Doom、Zatoichi Bo、Cane of Corpus（weapons.md Zergs 2023-12-17 測試帖；HOK1970「TOP 3 weapons are all staffs」）。
  - Staff of Doom：多人說全遊戲最強、任何職業可用、必掉、早期可拿、附 Death Cloud（Zergs、HOK1970、mike.xulima）。
  - Zatoichi Bo：武僧／武士，「best overall weapon」前三（Zergs、Cyran）。
  - Cane of Corpus：妖精忍者，配 Rod of Sprites 雙持（party.md Skleedle「CoC and RoS dual wielded」）。
- **劍**：Light Sword 最好，Bloodlust 第二，Ivory／Muramasa／Fang 差不多（Zergs）；Muramasa 傷害不如 Ivory、不值得 S/L 刷（Horpner Sneed、Zergs）；Light Sword 全面勝過 The Avenger（Sonera）。
- **副手**：Diamond Eyes 是最好的副手（bevinator、drpetrov）；武士副手只有 Wakizashi 類（drpetrov）。
- **其他**：忍者不詛咒的最佳是 Dread Spear（CeterumCenseo）；Bloodlust 最好給詩人（Zergs），附 Berserk −30（HOK1970）；遊俠、工匠以遠程為主（drpetrov）。

## 隊伍：社群共識
- **沒有單一公認最強隊**；多人說合理的隊伍都能破關（PointBlank、bevinator）。
- 反覆出現的元素：妖精忍者拿 CoC；Bishop 包四系（Element／Soul Shield 時期需要）；1 priest＋1 mage 法書就夠全 buff（Slooge）；純 hybrid 隊偏弱（Zergs）；女武神 Cheat Death 防禦強（CeterumCenseo）。
- 「最 OP」單一說法：3 名 Mook 戰士拿 Giant's Sword＋3 Bishop（GameNerdVeggie，單人說法）。
- **NPC**：Vi、RFS-81、Saxx 最適合長期帶（Horpner Sneed）；RFS-81 是唯一願意去 Rapax／Rynjin 區的（同帖）；NPC 分走經驗，整局約少 1 級（mpnorman10）。

## 對照我方定稿（`notes/party-final.md`）
| 項目 | 社群 | 我方模型 | 判讀 |
|---|---|---|---|
| 妖精忍者 CoC＋RoS | 頂級 | .430，定稿採用 | 一致 |
| Zatoichi Bo | 前三 | 僧侶 .325，定稿採用 | 一致 |
| Light Sword 最好的劍 | 是 | 盜賊 LS＋Stiletto .440（全隊最高） | 一致 |
| Muramasa 不如 Ivory | 是 | 武士村正＋EW .398 | 職業不同不可直比，未衝突 |
| Diamond Eyes 最佳副手 | 是 | 遊俠／Vi 用 Ivory＋DE | 一致 |
| NPC 選 Vi／RFS-81 | 是（加 Saxx） | B 隊、任選結論都含 RFS-81 | 一致；Saxx 是詩人，MG 已排除 |
| **Staff of Doom 全遊戲最強** | 多人 | 只給 T'rang 忍者 .284 | **最大落差**：模型只算即死，SoD 的麻痺 25、槽 3＝10、Death Cloud 都沒算（I9c「即死有建模，其他特效沒有」）。社群是實玩測試 |
| Bloodlust 第二好的劍 | 是 | 未進定稿（Attacks +1、Sword +5、Backstab 未建模；MG「上限低」不開票） | 未建模造成的落差 |
| Mook 戰士 Giant's Sword | 一人說最 OP | MG 現隊 Mook 遊俠 GS .202 | 單一說法，不採 |
