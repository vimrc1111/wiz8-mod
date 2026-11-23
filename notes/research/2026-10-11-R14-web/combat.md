# Wizardry 8 combat mechanics: web sources (rules.txt compliant)

注意：WebFetch 回傳的是小模型摘要後的文字。以下「引文」是 fetch 結果中以引號呈現的字句，不是原始 HTML 逐字。未加引號的內容一律未收。

## 來源清單
1. Steam「BASE ATTACK RATING FORMULA (updated)」
2. Steam「At last, I understand spell damage!」
3. GOG「w8 dual weapons skill」
4. GOG「wizardry 8 combat times」
5. GOG「wizardry 8 dexterity speed or senses」
6. GOG「party build guide」
7. GOG「good wizardry 8 party」page2
8. GOG「when to do class change」
9. Quarter to Three「Wizardry 8 Party Configs」page 2
10. Cyberly「wizardry 8 class guide」
11. RPG Codex 文章 7307
12. CrimsonTear「Wizardry 8」spells 頁
13. CrimsonTear「Wizardry 8」FAQs 頁
14. GameBanshee 文章（Rampant Coyote 系列引文）
15. Let's Play Archive「Wizardry 8, Part #1 - Crabs」

---

## BASE ATTACK RATING FORMULA (updated)
URL: https://steamcommunity.com/app/245450/discussions/0/2260187517774957003
Author/date if shown: peddroelm, 22 Apr 2020（編輯於 25 Apr 2020）
Kind: forum（玩家逆向公式帖）

- [combat]
  > "BAR SINGLE WEAPON = Floor( Floor(( CC_RC + 2 * WPN ) / 3 ) * 2 + Floor(DEX/2) + SUM(ProfessionLvLUpMul * Lvl) ) / 3) + 60"
- [combat]
  > "BAR DUAL WIELD MH = floor((floor((floor((CC_RC + 2*WPN1)/1.5) + floor(WPN1/2) + DW) * 10/35)*2 - 10 - floor((100-DW)/4 )"
- [combat]
  > "BAR DUAL WIELD OH = floor((floor((floor((CC_RC + 2*WPN2)/1.5) + floor(WPN2/2) + DW) * 10/35)*2 - 20 - floor((100-DW)/4 )"
- [combat] 共同尾項：
  > "+ floor(DEX/2) + SUM(ProfessionLvLUpMul * Lvl))/3) + 60."
- [combat]
  > "But at very low dual wield level it penalizes BOTH hads CTH/CTP heavily" （原文拼字照錄）
- [combat] 範例（Fighter lv20, DEX 105, CC 100, Mace 90, DW 81）：
  > "= 166 // UI shows 33"（單手）
  > "= 158 // UI Shows 32"（雙手主手）
  > "= 155 // UI shows 31"（雙手副手）
- [combat] 作者自註：
  > "will need to investigate a bit further"（主手 BAR 無說明下降後）

---

## At last, I understand spell damage!
URL: https://steamcommunity.com/app/245450/discussions/0/2139714324759466927
Author/date if shown: Uncle Al, 4 Apr 2020（編輯於 7 Apr 2020）
Kind: forum

- [spell] [stats]
  > "1/4 of the mana cost per power level of the spell +"
  > "1/2 of the level of the spell +"
  > "The power level you cast it at +"
  > "1/2 of your caster level"
- [spell] Power Cast 步驟：
  > "Then all of that is increased if you have power cast by a percentage equal to your power cast skill"
  > "(so double at 100% skill)."
- [spell] [crit/instadeath] 抗性公式：
  > "So Effective Resistance = (Target Level - SAS) * 3 + Base Resistance"
  > "If Effective Resistance is less than or equal to zero, you take full damage from the spell."
  > "Final Resistance = (Effective Resistance/2) + (Random Roll between 0 and Effective Resistance/2)"
  > "+ (Random Roll between 0 and Effective Resistance/2)"
  > "reduced by a percentage equal to the Final Resistance"
- [spell] 範例：
  > "0.5 + 0.5 + 1 + 0.5 = 2.5"（Energy Blast，level 1 mage，power 1）
  > "0.5 + 0.5 + 7 + 15 = 23"（Energy Blast，power 7，level 30 mage）
  > "4.5 + 3.5 + 7 + 15 = 30 then increase that by 100% for your 100 power cast = 60"（Nuclear Blast）

---

## w8 dual weapons skill
URL: https://gog.com/forum/wizardry_series/w8_dual_weapons_skill
Author/date if shown: drpetrov 2024-07-30、2024-08-12；RChu1982 2024-07-24；dtgreene 2024-07-30
Kind: forum

- [combat] [skill]
  > "When a character wields two weapons they get a penalty to their weapon skill when attacking with those weapons."
  > "With no points in Dual Weapons, the weapon in the primary hand attacks with only 50% of its skill."
  > "The weapon in the off-hand attacks with only 25% of its skill."
- [skill]
  > "Unlike most skills, there is no benefit to having the Dual Weapon skill at higher than 100"
- [skill] 反駁（未附來源）：
  > "My understanding is that this has been shown to be false; dual wielding past 100 does have an effect."
- [skill] 推測（作者自標 speculative）：
  > "your off-hand skill could get up to 56.5% of normal instead of only 50%"
- [class]
  > "The Lord does have a bonus to the Dual Weapons skill, but isn't going to be doing near the damage of the Fighter or Rogue dual-wielding."

---

## wizardry 8 combat times
URL: https://gog.com/forum/wizardry_series/wizardry_8_combat_times
Author/date if shown: abbayarra 2018-09-02；Immerhinque 2018-09-03；dtgreene 2018-09-03；ussnorway 2018-09-03
Kind: forum

- [combat]
  > "An HOUR later(55 minutes actually) I finally finish this fight!"
  > "What made this fight so tough was the rank weeds debuffing me."
- [combat]
  > "A fight vs Juggernauts at this stage takes very long, because they have high HPs and DR"
- [combat] [skill]
  > "get faster once your level increases."（dtgreene）
- [combat] 加速方法：
  > "You have your 2 Sorcerers call MIBL and press the enter button and not release it"
- [combat] 修正建議（Immerhinque）：
  > "put everything that speeds up combat to the max."
  > "turn the camera off"
  > "there are more options, to speed up movement. hit/miss animations, missile speed."
  > "lower the difficulty level. Gives more hits on your side and less to the monsters,"
  > "use less hybrids. Battles with hybrids take longer at this stage"
- [combat]
  > "getting some of the enemies to run away (via fear of blindness, for example)"（dtgreene）
- 註：此頁無人討論 initiative 或每回合攻擊次數。

---

## wizardry 8 dexterity speed or senses
URL: https://gog.com/forum/wizardry_series/wizardry_8_dexterity_speed_or_senses/page1
Author/date if shown: dtgreene 2019-11-20；abbayarra 2019-11-20；Zadok_Allen 2020-01-15；townltu 2020-01-22
Kind: forum

- [stats] [combat]
  > "what should be the *second* stat to increase, Dexterity, Speed, or Senses?"（dtgreene）
  > "It seems like Dexterity is the choice in the research I've done."（abbayarra）
  > "The only time I saw a different choice is speed for the samurai and senses for the Ninja and Ranger."（abbayarra）
  > "Dexterity is the right choice."（Zadok_Allen）
  > "has lower initiative and a bit more damage (14,5 DE & 19,5 tM vs 21 GS)"（Zadok_Allen）
- [combat] [equipment]（townltu）：
  > "In wiz8 the default number of attacks and swings in a single turn"
  > "is based on the characters (snake-)speed, dexterity, combat&weapon skill, and lvl"
  > "so dex & spd rise slower which affects attacks/swings/turn,"
  > "The dread spear weigs 6 lbs, lance and stun rod 18 lbs,"
  > "the +20 polarm will ensure that she gets 3 attacks with 3 swings each much earlier,"

---

## party build guide
URL: https://gog.com/forum/wizardry_series/party_build_guide
Author/date if shown: Charon121, bevinator, singbird, Nomad_Soul；2013-06-03
Kind: forum

- [party]
  > "You should have 2-3 melee fighters, at least 2 casters, and the others can be ranged/utility classes."
  > "You can't really go wrong, unless you do something dumb, like getting 6 characters of the same class."
- [party]
  > "bard and gadgeteer can work just fine here"（bevinator，補血角色）
- [equipment] [class]
  > "the heaviest armor and heaviest weapons can only be used by Fighters, Valkyries, and Lords."
- [combat]
  > "My alchemist does more damage with a sling and impaling stones than with spells."（singbird）
  > "if you want a character to deal damage in melee, max out strength as a priority."（Nomad_Soul）
  > "high STR and access to power strike \"shouldn't be overlooked.\""（Nomad_Soul）
- [party]
  > "If you really can't make it out of the Monastery (starting dungeon) despite the best tactical play"（Charon121）

---

## good wizardry 8 party（page 2）
URL: https://www.gog.com/forum/wizardry_series/good_wizardry_8_party/page2
Author/date if shown: Polly77、mndrew、deathknight1728、SacredPath、ussnorway、Charon121、potato_head；2013-06-15 至 2013-06-23
Kind: forum

- [party]
  > "Go for a balanced party. You have 6 slots and can add a further 2 members to your party once you reach Arnika."（Polly77，2013-06-15）
  > "make sure you train them to use either bow/crossbow or slingshots/throwing."（Polly77）
  > "I wouldn't advise Bishops for a newb."（SacredPath，2013-06-20）
- [stats]
  > "Prime stats as opposed to skills are the single easiest way to bugger up your game."（ussnorway，2013-06-20）
- [combat]
  > "The dynamics of combat don't rely on the role of the dice as they would do in D&D."（Polly77，2013-06-20）
  > "Melee only party are a challenge and can win the game,"（ussnorway，2013-06-20）
  > "BTW I usually play melee heavy parties for the very reason that melee is OP."（SacredPath，2013-06-21）
  > "From my three playthroughs I concluded that melee characters are the strongest."（Charon121，2013-06-23）
  > "A magic heavy party can still have a fighter or two, that can deal with those enemies."（potato_head，2013-06-23）

---

## when to do class change
URL: https://gog.com/forum/wizardry_series/when_to_do_class_change
Author/date if shown: abbayarra、dtgreene、Zadok_Allen；2016-02-26 至 2016-02-27
Kind: forum

- [class] [leveling]
  > "do it once leveling up starts to slow down significantly, and remember to continue to boost your spellbook skill."（dtgreene）
  > "Lord was somewhat noticeably worse than Valkyrie"（abbayarra）
  > "hybrids do a little better in the monastery"（abbayarra）
- [spell] [class]
  > "Mage->Samurai: Get Enchanted Blade and Missile Shield; those are the most important spells."（dtgreene）
  > "If I am not mistaken hybrids can't cast spells on levels 1-4."（Zadok_Allen）
- [skill] [leveling]
  > "that skill can be improved regardless of class"（dtgreene）
  > "If you do not qualify for the skill, you don't benefit from it"（dtgreene）

---

## Wizardry 8 Party Configs（page 2）
URL: https://forum.quartertothree.com/t/wizardry-8-party-configs/66021?page=2
Author/date if shown: zenblack、BoyFrance、Intangir、Fugitive、HumanTon、Razgon；2011-08-20
Kind: forum

- [combat] [class]
  > "a fighter of any flavor will constantly beat out any other melee class when it comes to damage."（zenblack，2011-08-20 9:43am）
  > "Fighters are awesome in Wiz8, they just aren't that versatile."（Intangir，12:46pm）
  > "In my last party, my fighter had the most kills."（Intangir，2:16pm）
- [class]
  > "Bishops aren't newbie friendly and take a lot more nurturing though"（Intangir，12:46pm）
  > "you have to be very careful about the order in which they raise skills."（Fugitive，12:54pm）
- [party]
  > "Instead of a Priest, I usually bring a Valkyrie (and/or recruit Vi) and have them double as my priest."（Intangir，12:46pm）
- [combat]
  > "Mobs won't attack you if they don't see you. Use the terrain to your advantage."（HumanTon，3:29pm）
  > "Fireball isn't really that good if I remember correctly after a short while, since it deals way to little damage"（Razgon，3:40pm）

---

## Cyberly: wizardry 8 class guide
URL: https://www.cyberly.org/game-cheats/pc-walkthroughs/wizardry-8-class-guide.html
Author/date if shown: 頁尾 "Copyright © 2025 Cyberly, James William Steven Parker"；無文章日期
Kind: guide

- [class] [combat]
  > Fighter 評級 "B+"；特性 "Berserk, Stamina Regeneration, May knock out opponents"
  > "The fighter gets the most hit points, and can hit the hardest, just as one would expect."
- [class] Valkyrie 評級 "A+"；特性 "Cheat death.....come again???? Yes, I did say cheat death."
  > "The Valkyrie is a great fighter, coming close to the fighter himself."
- [class] Lord 評級 "B- (B+ if you use the build in the tips section)"；特性 "Health Regeneration."
  > "Doesn't deal as much damage as a fighter, or even a valk."
- [class] [crit/instadeath] Samurai 評級 "B"；特性 "Critical Kill, Lightning Strike, immune to fear"
- [class] Ninja 評級 "B- (A- if Faerie)"；特性 "Auto Penetrate Thrown weapons, Thrown and melee Criticals, Able to mix potions and powders together"
- [class] Bishop 評級 "A-"
  > "turn undead is all but useless."

---

## RPG Codex 文章 7307（column）
URL: https://rpgcodex.net/article.php?id=7307
Author/date if shown: VentilatorOfDoom，2010-12-01
Kind: guide（專欄文章，非嚴格意義上的攻略）

- [combat]
  > "My number one complaint about Wizardry 8 [...] was the length of its combat."
- [combat]
  > "I love the little tactical mini-game of turn-based combat."
- [combat]
  > "Gold Box style: "signature tactical combat in the style of miniatures-based 1st edition Dungeons & Dragons games""
- 註：文中 Wizardry 8 機制描述僅此，無數值。

---

## CrimsonTear: wizardry-8 spells
URL: https://www.crimsontear.com/game-guides/wizardry-8/spells
Author/date if shown: 無
Kind: wiki（作者自述仍在撰寫中，另見搜尋摘要）

- [spell] [crit/instadeath]
  > "Spells fall into six realms: Air, Divine, Earth, Fire, Mental, and Water."
  > "Spells always hit their target once cast."
  > "They might fizzle and not go off, or sometimes backfire and hit your party or the caster instead, but they don't miss."
  > "How much damage a spell does depends on the target's resistance."
  > "Fire Realm spells do not work underwater."
- [crit/instadeath] 頁內列表：
  > Instant Death (Divine, Level 5): "Instantly Kills Single Enemy"
  > Death Wish (Divine, Level 7): "Chance of Instant Death to All Enemies"
  > Death Cloud (Air, Level 7): "Chance of Instant Death Each Turn"
  > Asphyxiation (Air, Level 7): "Instand Death to All Enemies"（頁面原拼字）
  > Quicksand (Earth, Level 6): "Instand Death to Enemies in Radius"（頁面原拼字）

---

## CrimsonTear: wizardry-8 FAQs
URL: https://www.crimsontear.com/game-guides/wizardry-8/faqs
Author/date if shown: 無
Kind: FAQ

- [combat] 注意：下列為 fetch 摘要中的引號句；括號內為摘要者的改寫註記，非原文。
  > "Combat in narrow passageways is safe enough to fight directly"
  > "Let them come to you instead of rushing in."
  > "place your weaker or squishier characters away from the reach of those you are fighting"
  > "enemies with long reach, ranged attacks, or magic can still target anyone in your group"
  > "Just because you are in the melee range of one enemy does not mean it should be your primary target."
  > "You can always swap your weapons to target something further away"
  > "enter combat mode by clicking the sword button at the bottom"
- 註：此頁無 critical hit、armor class 文字。

---

## GameBanshee 文章（Rampant Coyote 系列引文）
URL: https://www.gamebanshee.com/uyp9
Author/date if shown: BuckGB（引述 The Rampant Coyote "Storming the Castle"），2009-02-06
Kind: guide（引述文章）

- [combat]
  > "The main floor of the castle was largely the same story - infuriatingly long combats."
  > "Rapax are minotaur-looking beasts which have some of the most infuriatingly boring combat in the known universe."
  > "It can shave precious minutes off of an hour-long fight."
- [combat] [spell] [crit/instadeath]
  > "Usually the best spells are buffs, heals, and insanity spells"
  > "This usually means putting up an element shield in the first round"
- [combat] 敵人韌性：
  > "They are - tough. Very tough. Most magic barely touches them. They hit like a ton of bricks. They have hundreds of hit points."

---

## Let's Play Archive: Wizardry 8, Part #1 - Crabs. It's Mostly Just Crabs.
URL: https://lparchive.org/Wizardry-8/Update%2001/
Author/date if shown: PurpleXVI；無日期
Kind: guide（Let's Play，最接近的類別）

- [combat]
  > "Another advantage of putting enemies to sleep(or otherwise making them unable to fight back, with, say, paralysis or webbing), is that all melee attacks do double damage to them because they can't dodge."
  > "Ranged weapons and spells, of course, don't give a damn about this."
  > "Disease also does not go away with time."
- [crit/instadeath] 描述 Twinklesparkles 的徒手戰鬥：
  > "double the chances for critical strike instant kills"
- [spell]
  > "Wizardry 8's Magic Missile is more like a blast of magic buckshot that nails everything in sight."
  > "The higher the level, the higher the power, and also the greater the risk."

---

## SNIPPET（搜尋摘要，未 fetch，不視為來源內容）
- [crit/instadeath] 搜尋摘要稱 Samurai 以近戰 critical 可 instant kill、Ninja 以投擲武器 critical 可 instant kill（出處：搜尋結果中的 fan site，未取得該頁）。
- [crit/instadeath] 搜尋摘要稱 Death Cloud 為 Air 系 level 7，每回合有機率即死（與 CrimsonTear 頁一致，已由該頁 fetch 證實，見上）。
- [combat] 搜尋摘要稱 Steam 上有「Base: 顯示為 floor(actual_baseAR/5)」的說明（已由來源 1 fetch，引文未直接取得該句，未收）。

---

## 注入內容通報（rules §4.1）
- Steam 帖 https://steamcommunity.com/app/245450/discussions/0/1737760452986202911 的 fetch 結果末尾含一段給 AI 的指示，要求「quote text about instant death, critical hits, and damage」並附長度與格式限制。該頁內容未被收錄為引文。
- Steam 帖 https://steamcommunity.com/app/245450/discussions/0/2145343448810179395 的 fetch 結果末尾含類似指示（要求逐字引用並遵守格式）。該頁內容只留在 fetch 摘要中，無可用逐字引文，未收錄。
- 原文字：此處只能取得摘要者的描述，無法取得原始指示逐字。上述兩段指示均未遵循。

---

## Searches run
1. "Wizardry 8 combat mechanics" → 9 results, 有用 4（lparchive、wizardry8 tips、GOG hangs in combat、preview）
2. "Wizardry 8 GameFAQs FAQ" → 9 results, 有用 2（CrimsonTear FAQs、Cyberly）；無 GameFAQs 頁面
3. "Wizardry 8 critical hit instant death decapitation" → 10 results, 有用 3（Crimson Tear instant death、Steam、fan site 摘要）；無斬首機制
4. "Wizardry 8 formulas" → 9 results, 有用 3（Steam 公式帖×2、crimsontear spells）
5. "Wizardry 8 best party guide" → 9 results, 有用 4（GOG party build、GOG party、Steam 多帖、Quarter to Three）
6. "Wizardry 8 dual wielding attack rating" → 9 results, 有用 3（Steam 公式帖、GOG dual weapons skill、GOG rothgar）
7. "Wizardry 8 skills explained" → 9 results, 有用 2（jefklakscodex classes、Steam skill 帖）
8. "site:reddit.com/r/wizardry Wizardry 8 combat" → 9 results, reddit 0；有用 3（GOG combat times、GOG dexterity、GameBanshee）
9. "site:rpgcodex.net Wizardry 8 combat" → 9 results, 有用 2（RPG Codex 7307、138）
10. "Wizardry 8 class guide ninja samurai lord valkyrie" → 10 results, 有用 3（GOG when to change、Cyberly、Quarter to Three）

未執行的建議查詢：「Wizardry 8 party guide」（改以 best party guide 替代）、「Wizardry 8 skills explained」之外的 Celtic Minstrel／wiki 查詢、site:gamefaqs.gamespot.com、site:steamcommunity.com 單獨查詢、site:rpgcodex.net formulas。

## Not fetched
- https://git.brainbaking.com/wgroeneveld/jefklakscodex/src/branch/master/content/articles/wizardry8-tips.md（及 raw 版）：TLS 憑證主機名不符，WebFetch 拒絕。
- https://git.brainbaking.com/wgroeneveld/jefklakscodex/src/branch/master/content/articles/wizardry8-classes.md（及 raw 版）：同上。
- https://gog.com/forum/wizardry_series/wizardry_8_hangs_in_combat：未 fetch。
- https://www.mocagh.org/sir-tech/wiz8-hintbook.pdf：回應超過 10MB 上限，拒絕。
- https://www.gog.com/forum/wizardry_series/party_build_guide 之外的 cyberly 其他頁：未 fetch。
- https://steamcommunity.com/app/245450/discussions/0/1737760452986202911：fetch 成功但無可用引文（見注入通報）。
- https://steamcommunity.com/app/245450/discussions/0/2145343448810179395：fetch 成功但無逐字引文（見注入通報）。
- GameFAQs：搜尋未找到 Wizardry 8 頁面。
- reddit.com/r/wizardry：搜尋無結果。
- https://steamcommunity.com/app/245450/discussions/0/2626094636170792310、https://steamcommunity.com/app/245450/discussions/0/2805074491017891006 等搜尋結果：未 fetch。
- https://rpgcodex.net/article.php?id=138 與 https://www.crimsontear.com/game-guides/wizardry-8 主頁：fetch 成功，因來源數上限未收錄為獨立來源（138 的 "phase-based" 句與主頁的 "Gone is the grid-based movement" 句可在需要時補收）。
- 暴擊斬首（decapitation）、武器 ranged 彈藥數、armor class 數值、resistance 表：所有已取得頁面均未提供。
