# calc 題庫（goals/stage10.md B3「不用再反組譯」）

每題一個 `calc-qa` 區塊，`bash scripts/calc-qa.test.sh` 逐字重跑比對（CI 跑 `--progress`，完成模式由 check 定義 3 驗）。
`$EXE`、`$DBS` 由測試代換成玩家自有 Wiz8.exe 與 DATABASES 目錄。情境檔在 `notes/calc-qa/`。
答不出的題目寫 `未取得: <節點>`（節點必須已登記為未取得）；節點還在做的寫 `待節點: <節點>`，完成前要換成真答案。

## MG 問過的

### Q1 領主 L28 持 Excaliber，對 Dark Savant（DefHit 32）每回合期望傷害？
```calc-qa
$ calc build -exe $EXE -dbs $DBS notes/calc-qa/l28-c1.json
屬性         STR   INT        PIE  VIT  DEX  SPD  SEN
最終裸值       100   45         55   93   100  99   45
手部         Item  武器         模式   K    M    A    V   BARraw  HitMod  DmgPct  ARhit  ARpen  期望揮擊數     有效揮擊數     期望傷害
主手         34    Excaliber  0    100  0    103  87  180     17      100     133    133    4.500000  4.275000  235.125000
合計期望揮擊數    4.500000
合計每回合期望傷害  235.125000
build 未支援：負重（E、e 取 0）、裝備屬性加成、特殊物品傷害分支（F28 10×Z）、練技成長、背刺、Kill 抗性。
未支援：遠程、AoE 名單、施法成功率、Kill 抗性、狀態效果、F01 精確 RNG；近戰不含失手換目標及行動中斷。
```

### Q2 女武神 L28 持 Dread Spear（長柄主技能 125），同一目標？
```calc-qa
$ calc build -exe $EXE -dbs $DBS notes/calc-qa/l28-c2.json
屬性         STR   INT          PIE  VIT  DEX  SPD  SEN
最終裸值       100   45           55   94   100  98   45
手部         Item  武器           模式   K    M    A    V    BARraw  HitMod  DmgPct  ARhit  ARpen  期望揮擊數     有效揮擊數     期望傷害
主手         815   Dread Spear  0    116  0    109  115  191     15      100     134    134    6.000000  5.700000  193.800000
合計期望揮擊數    6.000000
合計每回合期望傷害  193.800000
build 未支援：負重（E、e 取 0）、裝備屬性加成、特殊物品傷害分支（F28 10×Z）、練技成長、背刺、Kill 抗性。
未支援：遠程、AoE 名單、施法成功率、Kill 抗性、狀態效果、F01 精確 RNG；近戰不含失手換目標及行動中斷。
```

結論：Q1 235.1 對 Q2 193.8，領主勝；女武神揮擊數多（6 對 4.5）但 K 加成抵不過 Excaliber 的骰。

### Q3 盜賊背刺：L20 Rogue 持匕首對一般怪每下多幾組骰、命中加多少？
旗標：L20 職業列 7（Rogue）、匕首（item 0，技能 4、射程 0、模式遮罩 0x2）、貼身一般怪（情境取 re/emu/cases/0053d870.json 邊界列 36 改欄）；輸出 flag9a5=1、mode0=1（遮罩 0x2 必選模式 1）。
```calc-qa
$ calc f 0053d870 0 -1 0 0 2 2 0 1 1 1 1 4 1 3 3 2 1 0 1 0 7 0 20 1 0 3 4 0 -1 0 0 0 0 2 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1 10 0 0 0 20 0 0 0 1 1 0 0 1 100 0 1 1 1 1 0 0 1092616192 0 1092616192 1093664768 0 1092616192 1065353216 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
dist
1,1,6,4294967295,0,0,0,1,2,0,0,1,1,77,1,0,0,0,0,0,0,0,0,0,1,77,301,302,303,304,101,102,103,536936448,1,0,0,0: 0.333374023438
1,2,6,4294967295,0,0,0,1,2,0,0,1,1,77,1,0,0,0,0,0,0,0,0,0,1,77,301,302,303,304,101,102,103,536936448,1,0,0,0: 0.333312988281
1,3,6,4294967295,0,0,0,1,2,0,0,1,1,77,1,0,0,0,0,0,0,0,0,0,1,77,301,302,303,304,101,102,103,536936448,1,0,0,0: 0.333312988281
mean ret: 1
mean T68: 1.99993896484
mean T70: 6
mean T74: 4294967295
mean T78: 0
mean T7c: 0
mean Ta5: 0
mean Tswings0: 1
mean Tswings1: 2
mean Tls: 0
mean flag9a4: 0
mean flag9a5: 1
mean mode0: 1
mean mode1: 77
mean mcntOut: 1
mean f3o0: 0
mean f3o1: 0
mean f3o2: 0
mean f3o3: 0
mean f3o4: 0
mean f3o5: 0
mean f3o6: 0
mean f3o7: 0
mean c_mainAction: 0
mean c_mainSpell: 1
mean c_mainParameter: 77
mean c_mainTargetWord0: 301
mean c_mainTargetWord1: 302
mean c_mainTargetWord2: 303
mean c_mainTargetWord3: 304
mean c_mainTargetWord4: 101
mean c_mainTargetWord5: 102
mean c_mainTargetWord6: 103
mean c_mainTargetWord7: 536936448
mean c_actionCompleted: 1
mean c_deferredDefense: 0
mean c_refreshMask: 0
mean c_interceptAttempts: 0
in: who, mode, flag510c, hand, swings0, swings1, lsFlag, kind0, kind1, enabled0, enabled1, hmode0, hmode1, dice0, dice1, umask0, umask1, item6, item7, a15, cls, race, level, inParty, mainAction, ttype, tchar, tmon, tgrp, it0cls, it0sub, it0range, it0bonus, it0wmask, it1cls, it1sub, it1range, it1bonus, it1wmask, it2cls, it2sub, it2range, it2bonus, it2wmask, it3cls, it3sub, it3range, it3bonus, it3wmask, tpresent, thp, tstatus, tmainA, toffA, tcur, tinterf, tatt, mid, mhp, m33, m107, m87, m2e1, m24b, mexcl, mactive, men, mphys, mspell, mcnt, combat, px, py, pz, mx, my, mz, mrad, prad, mhead, f0_0, f0_1, f0_2, f0_3, f0_4, f0_5, f0_6, f0_7, f3_0, f3_1, f3_2, f3_3, f3_4, f3_5, f3_6, f3_7
out: ret, T68, T70, T74, T78, T7c, Ta5, Tswings0, Tswings1, Tls, flag9a4, flag9a5, mode0, mode1, mcntOut, f3o0, f3o1, f3o2, f3o3, f3o4, f3o5, f3o6, f3o7, c_mainAction, c_mainSpell, c_mainParameter, c_mainTargetWord0, c_mainTargetWord1, c_mainTargetWord2, c_mainTargetWord3, c_mainTargetWord4, c_mainTargetWord5, c_mainTargetWord6, c_mainTargetWord7, c_actionCompleted, c_deferredDefense, c_refreshMask, c_interceptAttempts
```
命中：同一份 re/emu/cases/00541c00.json 邊界列 0 改欄（mode 1、職業 7、等級 20），只差 surprise（G+0x9a5）；絕對值是 fixture 數，只看差。
```calc-qa
$ calc f 00541c00 3 1 4 1 0 0 0 0 0 0 0 0 4 0 0 0 0 2 256 1 0 0 0 0 123 0 -83490646 -45 0 0 0 40 20 3 2 1 0 0 0 0 60 50 50 50 0 7 0 1 0 40 40 40 40 40 40 40 40 10 10 10 10 10 10 10 10 20 10 10 10 10 10 10 10
dist
-205: 1
mean arhit: -205
in: targetType, mode, characterIndex, combat, ambush, surprise, speedDefense, shieldDefense, characterDefenseSuppressed, monsterDefense, monsterHitDefense, fatigue, monsterPosture, monsterDefenseSuppressed, ambushDefenseMask, completedActionSelector, completedPosture, pendingPosture, sceneFlags, HP, status, actionCompleted, mainAction, offAction, monsterID, templateIndex, characterDefHit, monsterAC, who, hand, skipDefense, handB0, handB1, handJ0, handJ1, handKind0, handKind1, skillBonus, fatigueCategory, battleRecordT94, speed, attackerStat0, attackerStat1, attackerStat2, levelCapState, currentClass, race, difficulty, reverseDifficultyState, targetCharStat0, targetCharStat1, targetCharSpeed, targetCharStat3, monsterStat0, monsterStat1, monsterStat2, monsterStat3, classLevel-1, classLevel0, classLevel1, classLevel2, classLevel3, classLevel4, classLevel5, classLevel6, classLevel7, classLevel8, classLevel9, classLevel10, classLevel11, classLevel12, classLevel13, classLevel14
out: arhit
```
```calc-qa
$ calc f 00541c00 3 1 4 1 0 1 0 0 0 0 0 0 4 0 0 0 0 2 256 1 0 0 0 0 123 0 -83490646 -45 0 0 0 40 20 3 2 1 0 0 0 0 60 50 50 50 0 7 0 1 0 40 40 40 40 40 40 40 40 10 10 10 10 10 10 10 10 20 10 10 10 10 10 10 10
dist
-195: 1
mean arhit: -195
in: targetType, mode, characterIndex, combat, ambush, surprise, speedDefense, shieldDefense, characterDefenseSuppressed, monsterDefense, monsterHitDefense, fatigue, monsterPosture, monsterDefenseSuppressed, ambushDefenseMask, completedActionSelector, completedPosture, pendingPosture, sceneFlags, HP, status, actionCompleted, mainAction, offAction, monsterID, templateIndex, characterDefHit, monsterAC, who, hand, skipDefense, handB0, handB1, handJ0, handJ1, handKind0, handKind1, skillBonus, fatigueCategory, battleRecordT94, speed, attackerStat0, attackerStat1, attackerStat2, levelCapState, currentClass, race, difficulty, reverseDifficultyState, targetCharStat0, targetCharStat1, targetCharSpeed, targetCharStat3, monsterStat0, monsterStat1, monsterStat2, monsterStat3, classLevel-1, classLevel0, classLevel1, classLevel2, classLevel3, classLevel4, classLevel5, classLevel6, classLevel7, classLevel8, classLevel9, classLevel10, classLevel11, classLevel12, classLevel13, classLevel14
out: arhit
```
額外骰：傷害端 0x54226b 在 0x5420b0（weapon_damage，已 D4）。情境取 re/emu/cases/005420b0.json 邊界列 0 改欄：mode 0→1（匕首遮罩 0x2 必選模式 1）、tgt 0→3（打怪；mK=0、mA=0 為一般怪，不觸發扣一組的分支）、lvi 0→7（目前職業列 7）、lv7 12→20（L20）、idice6 → 67174400＝0x04010000，即 Dagger 1d4（ITEMS.DBS id 0 記錄 +0x4a：Z=0、N=1、S=4，與 internal/calc/load.go readDice 同欄）。兩區塊只差 f9a5（G+0x9a5 背刺旗標）0→1。其餘欄（pwr 0、hb0 0、ib* 255 無剋星、f9a4 0 非背後）是 fixture 值，傷害絕對值只看差。gout＝這一下擲的骰組數；c52e 是 0x52e690（排除）的呼叫機率，其回傳不被讀。
```calc-qa
$ calc f 005420b0 0 0 1 3 1 7 0 0 0 0 0 6 -1 7 4 5 6 7 8 9 10 11 20 13 14 15 16 17 18 19 0 1 2 3 4 5 6 7 0 1 2 3 1 1 0 0 67239937 50528258 0 0 0 0 0 0 0 0 0 0 33751041 33751042 33751043 33751044 33751045 33751046 67174400 33751048 255 255 255 255 255 255 255 255 4660 536936448 10 11 12 13 14 15 16 17 0 0 0 0 0 0 0 0 0 0 0 1 0 0 0 0
dist
1,1,2779096485,1515870810,0,195,210,225,1,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.250030517578
2,1,2779096485,1515870810,0,195,210,225,1,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.25
3,1,2779096485,1515870810,0,195,210,225,1,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.25
4,1,2779096485,1515870810,1,195,210,225,1,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.249969482422
mean ret: 2.49990844727
mean gout: 1
mean gpad0: 2779096485
mean gpad1: 1515870810
mean fout: 0.249969482422
mean fpad0: 195
mean fpad1: 210
mean fpad2: 225
mean c517: 1
mean c51c: 1
mean c4ed: 0
mean c553: 0
mean c52e: 0
mean a4ed1_0: 0
mean a4ed1_1: 0
mean a4ed2_0: 0
mean a4ed2_1: 0
mean a4ed3_0: 0
mean a4ed3_1: 0
mean a553_1: 0
mean a553_2: 0
mean a553_3: 0
mean a553_4: 0
mean a52e_1: 0
mean a52e_2: 0
mean a52e_3: 0
mean a52e_4: 0
mean a52e_5: 0
mean a51c_1: 6
in: who, hand, mode, tgt, tchar, mid, g90c, g910, g918, f9a4, f9a5, s70, s74, lvi, lvm1, lv0, lv1, lv2, lv3, lv4, lv5, lv6, lv7, lv8, lv9, lv10, lv11, lv12, lv13, lv14, it0, it1, it2, it3, it4, it5, it6, it7, it8, it9, it10, it11, hf0, hf1, hb0, hb1, hd0, hd1, pwr, tA, tDA, tS, mtidx, mA, mK, mD, tcb, tcf, idice0, idice1, idice2, idice3, idice4, idice5, idice6, idice7, ib0, ib1, ib2, ib3, ib4, ib5, ib6, ib7, g68c510, cbase, dd0, dd1, dd2, dd3, dd4, dd5, dd6, dd7, r51, r4d0, r4d1, fcls0, fcls1, fcls2, fcls3, fcls4, fcls5, fcls6, fcls7, fhp, fst, fmain, foff, fdone
out: ret, gout, gpad0, gpad1, fout, fpad0, fpad1, fpad2, c517, c51c, c4ed, c553, c52e, a4ed1_0, a4ed1_1, a4ed2_0, a4ed2_1, a4ed3_0, a4ed3_1, a553_1, a553_2, a553_3, a553_4, a52e_1, a52e_2, a52e_3, a52e_4, a52e_5, a51c_1
```
```calc-qa
$ calc f 005420b0 0 0 1 3 1 7 0 0 0 0 1 6 -1 7 4 5 6 7 8 9 10 11 20 13 14 15 16 17 18 19 0 1 2 3 4 5 6 7 0 1 2 3 1 1 0 0 67239937 50528258 0 0 0 0 0 0 0 0 0 0 33751041 33751042 33751043 33751044 33751045 33751046 67174400 33751048 255 255 255 255 255 255 255 255 4660 536936448 10 11 12 13 14 15 16 17 0 0 0 0 0 0 0 0 0 0 0 1 0 0 0 0
dist
1,1,2779096485,1515870810,0,195,210,225,1,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0399981681257
2,1,2779096485,1515870810,0,195,210,225,1,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0399932861328
2,2,2779096485,1515870810,0,195,210,225,2,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0300045011658
2,2,2779096485,1515870810,0,195,210,225,2,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.00750312841498
3,1,2779096485,1515870810,0,195,210,225,1,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0399932861328
3,2,2779096485,1515870810,0,195,210,225,2,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.060001677908
3,2,2779096485,1515870810,0,195,210,225,2,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.0150044252351
3,3,2779096485,1515870810,0,195,210,225,3,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.00187572488562
3,3,2779096485,1515870810,0,195,210,225,3,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.00125061046799
4,1,2779096485,1515870810,1,195,210,225,1,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0399884041399
4,2,2779096485,1515870810,0,195,210,225,2,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0899988550972
4,2,2779096485,1515870810,0,195,210,225,2,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.022505722167
4,3,2779096485,1515870810,0,195,210,225,3,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.00562648782972
4,3,2779096485,1515870810,0,195,210,225,3,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.00375137347263
4,4,2779096485,1515870810,0,195,210,225,4,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 6.25858576995e-05
4,4,2779096485,1515870810,0,195,210,225,4,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 9.38931126269e-05
5,2,2779096485,1515870810,0,195,210,225,2,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.119988707863
5,2,2779096485,1515870810,0,195,210,225,2,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.030005187504
5,3,2779096485,1515870810,0,195,210,225,3,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0112522889161
5,3,2779096485,1515870810,0,195,210,225,3,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.00750228906983
5,4,2779096485,1515870810,0,195,210,225,4,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.000250312875027
5,4,2779096485,1515870810,0,195,210,225,4,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.000375526609857
6,2,2779096485,1515870810,0,195,210,225,2,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0899842080381
6,2,2779096485,1515870810,0,195,210,225,2,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.0225020594243
6,3,2779096485,1515870810,0,195,210,225,3,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0187524413178
6,3,2779096485,1515870810,0,195,210,225,3,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.0125028993282
6,4,2779096485,1515870810,0,195,210,225,4,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.000625736359506
6,4,2779096485,1515870810,0,195,210,225,4,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.000938747772059
7,2,2779096485,1515870810,0,195,210,225,2,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0599870308488
7,2,2779096485,1515870810,0,195,210,225,2,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.0150007624924
7,3,2779096485,1515870810,0,195,210,225,3,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0225011441996
7,3,2779096485,1515870810,0,195,210,225,3,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.0150022888183
7,4,2779096485,1515870810,0,195,210,225,4,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.00125138106289
7,4,2779096485,1515870810,0,195,210,225,4,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.00187735803895
8,2,2779096485,1515870810,1,195,210,225,2,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0299898541066
8,2,2779096485,1515870810,1,195,210,225,2,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.00749946567233
8,3,2779096485,1515870810,0,195,210,225,3,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0224997708807
8,3,2779096485,1515870810,0,195,210,225,3,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.0150013731792
8,4,2779096485,1515870810,0,195,210,225,4,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.00193948942885
8,4,2779096485,1515870810,0,195,210,225,4,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.00290967809783
9,3,2779096485,1515870810,0,195,210,225,3,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0187483213609
9,3,2779096485,1515870810,0,195,210,225,3,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.0125001524109
9,4,2779096485,1515870810,0,195,210,225,4,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.00250239554603
9,4,2779096485,1515870810,0,195,210,225,4,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.00375416612437
10,3,2779096485,1515870810,0,195,210,225,3,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0112481689593
10,3,2779096485,1515870810,0,195,210,225,3,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.00749954215251
10,4,2779096485,1515870810,0,195,210,225,4,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.00275243349743
10,4,2779096485,1515870810,0,195,210,225,4,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.00412928028586
11,3,2779096485,1515870810,0,195,210,225,3,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.00562374119182
11,3,2779096485,1515870810,0,195,210,225,3,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.00374954219442
11,4,2779096485,1515870810,0,195,210,225,4,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.00250202901102
11,4,2779096485,1515870810,0,195,210,225,4,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.00375361623796
12,3,2779096485,1515870810,1,195,210,225,3,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.00187435156666
12,3,2779096485,1515870810,1,195,210,225,3,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.00124969482888
12,4,2779096485,1515870810,0,195,210,225,4,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.00193893962635
12,4,2779096485,1515870810,0,195,210,225,4,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.00290885326822
13,4,2779096485,1515870810,0,195,210,225,4,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.00125083126038
13,4,2779096485,1515870810,0,195,210,225,4,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.00187653320934
14,4,2779096485,1515870810,0,195,210,225,4,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.0006253698245
14,4,2779096485,1515870810,0,195,210,225,4,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.000938197885649
15,4,2779096485,1515870810,0,195,210,225,4,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 0.000250129607524
15,4,2779096485,1515870810,0,195,210,225,4,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 0.000375251666651
16,4,2779096485,1515870810,1,195,210,225,4,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,6: 6.25247685311e-05
16,4,2779096485,1515870810,1,195,210,225,4,1,0,0,1,0,0,0,0,0,0,0,0,0,0,536936448,4660,0,0,127,6: 9.38014648908e-05
mean ret: 5.30009886622
mean gout: 2.1201171875
mean gpad0: 2779096485
mean gpad1: 1515870810
mean fout: 0.0807580965478
mean fpad0: 195
mean fpad1: 210
mean fpad2: 225
mean c517: 2.1201171875
mean c51c: 1
mean c4ed: 0
mean c553: 0
mean c52e: 0.224055420607
mean a4ed1_0: 0
mean a4ed1_1: 0
mean a4ed2_0: 0
mean a4ed2_1: 0
mean a4ed3_0: 0
mean a4ed3_1: 0
mean a553_1: 0
mean a553_2: 0
mean a553_3: 0
mean a553_4: 0
mean a52e_1: 120303521.696
mean a52e_2: 1044.09826003
mean a52e_3: 0
mean a52e_4: 0
mean a52e_5: 28.4550384171
mean a51c_1: 6
in: who, hand, mode, tgt, tchar, mid, g90c, g910, g918, f9a4, f9a5, s70, s74, lvi, lvm1, lv0, lv1, lv2, lv3, lv4, lv5, lv6, lv7, lv8, lv9, lv10, lv11, lv12, lv13, lv14, it0, it1, it2, it3, it4, it5, it6, it7, it8, it9, it10, it11, hf0, hf1, hb0, hb1, hd0, hd1, pwr, tA, tDA, tS, mtidx, mA, mK, mD, tcb, tcf, idice0, idice1, idice2, idice3, idice4, idice5, idice6, idice7, ib0, ib1, ib2, ib3, ib4, ib5, ib6, ib7, g68c510, cbase, dd0, dd1, dd2, dd3, dd4, dd5, dd6, dd7, r51, r4d0, r4d1, fcls0, fcls1, fcls2, fcls3, fcls4, fcls5, fcls6, fcls7, fhp, fst, fmain, foff, fdone
out: ret, gout, gpad0, gpad1, fout, fpad0, fpad1, fpad2, c517, c51c, c4ed, c553, c52e, a4ed1_0, a4ed1_1, a4ed2_0, a4ed2_1, a4ed3_0, a4ed3_1, a553_1, a553_2, a553_3, a553_4, a52e_1, a52e_2, a52e_3, a52e_4, a52e_5, a51c_1
```

背刺時骰組數分布（由第二區塊 dist 的 gout 欄加總）：1 組 0.16、2 組 0.60、3 組 0.20、4 組 0.04，平均 2.12 組，對照無旗標固定 1 組；L20 的門檻 t=99（P(≥+1)=(t−15)/100=0.84、P(≥+2)=0.24、P(≥+3)=0.04）。Dagger 1d4 每下期望 2.50→5.30。

結論：L20 盜賊持匕首背刺一般怪，命中 +10（-205→-195），每下平均多 1.12 組武器骰（+0 組 16%、+1 組 60%、+2 組 20%、+3 組 4%），Dagger 每下期望傷害由 2.50 升到 5.30。

### Q4 即死：L28 Samurai 持村正對 Mad Marten（id 411）每回合即死機率？
即死揮擊 0x543270 槽 5：R(100) < chance5 才觸發，觸發後呼叫 0x551ba0（狀態 0x12、realm 5）；目標過 0x5520d0 抗性骰失敗才死，計入 stats_50 並發 0x52e690 事件。chance5＝武器 +0x55＋⌊Critical Strike（技能 19）/25⌋（近戰手，0x53e250）：村正＝ITEMS.DBS id 41，+0x55＝15；Critical Strike 100 → +4 → 19。抗性 a＝3×(怪物等級−強度)＋bonus[0x12]＋Rj，夾 5..95；強度＝L＋min(L,15)＝43（L28）。Mad Marten＝MONSTERS.DBS id 411：等級 22、六系抗性 100、+0xcb＝20、+0x1be＝0（記錄 4＋id×0x297）。情境取 re/emu/cases/00543270.json 列 81 改上列欄位、難度 1（怪物 pct 100）。
```calc-qa
$ calc f 00543270 0 1 536871168 1 0 2513873324 3 0 3958627719 0 156 184 3 68 81 1 0 2 150 34 9276 1 -21241 214 159 0 0 0 0 0 19 0 0 0 0 0 0 0 0 0 0 43 1 50 -4 11 0 2 7 2402464987 842 472 632 931 765 518 124 465 145 839 822 864 27 661 669 833 756 63 337 606 589 652 79 805 3391413659 71 26 0 1 0 1 0 1 1 0 0 1 1 0 1 20 1 9 16 6 0 5 20 16 20 11 0 7 9 1 5 7 14 11 8 9 12 0 1 10 13 12 10 5 10 3 14 11 5 13 2 4 7 6 0 0 0 0 8 13 46 1230137050 4086375950 893252631 2775586159 3402614605 927822497 0 0 -1 -1 0 20 0 0 1 1 10 0 0 0 0 0 0 0 0 100 100 100 100 100 100 22 0 0 0 0 0 0 0 20 1 5 65 66 0 1 7 65 66 0 0 50 10 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 9 100 5 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1 1 1 1 1 396 20 19 0 48 0 0 48 0 0 48 0 0 48 0 0 48 0 0 48 0 0 48 0 0 48 0 0 48 0 0 52 0 0 0 52 0 0 0 52 0 0 0 52 0 0 0 52 0 0 0 52 0 0 0 0 10 5 536870912 536940544 0 0 0 0 0 537133056 0 0 0 537137152 0 0 0 537138176 0 1 1 1 1 1 1 1 1 1 -1 -1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 536956928 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 255 255 255 255 255 255 255 255 255 255 255 255 255 255 255 0 0 0 0 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 255 255 255 255 255 255 255 255 255 255 255 255 255 255 0 0 0 0 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 255 255 255 255 255 255 255 255 255 255 255 255 255 255 0 0 0 0 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 4660 100 10 1000 0 -1 3 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 10 0 1 7 8 9 1 2 3 536870912 536940544 0 0 0 0 0 537133056 0 0 0 537137152 0 0 0 537138176 0 1 1 1 1 1 1 1 1 1 -1 -1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 536956928 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 255 255 255 255 255 255 255 255 255 255 255 255 255 255 255 0 0 0 0 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 255 255 255 255 255 255 255 255 255 255 255 255 255 255 0 0 0 0 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 255 255 255 255 255 255 255 255 255 255 255 255 255 255 0 0 0 0 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 100 100 0 0 0 0
dist
44295,214,159,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,64,0,0,0,0,79,805,34,1,1,18,0,0,0,0,0,0,0,0,0,0,0,0,5,0,0,0,0,0,0,0,0,0,0,0,0,43,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,9999,0,0,0,0,0,0,0,0,0,0,0,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1,536936448,56,0,0,127,1,8,0,3391413659,0,0,0,0,0,0,0: 0.0370526575193
44295,214,159,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,64,0,0,0,0,79,805,34,1,1,18,0,0,0,0,0,0,0,0,0,0,0,0,5,0,0,0,0,0,0,0,0,0,0,0,0,43,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,9999,0,0,0,0,0,0,0,0,0,0,0,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1,536936448,2402464987,0,0,127,1,8,0,3391413659,0,0,0,0,0,0,0: 0.0370571808249
44295,214,159,842,472,632,931,765,518,124,465,145,839,822,864,27,661,669,833,756,63,337,606,589,652,79,805,34,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0: 0.809967041016
44295,214,159,842,472,632,931,765,518,124,465,145,839,822,864,27,661,669,833,756,63,337,606,589,652,79,805,34,1,1,18,0,0,0,0,0,0,0,0,0,0,0,0,5,0,0,0,0,0,0,0,0,0,0,0,0,43,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,9999,0,0,0,0,0,0,0,0,0,0,0,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0: 0.11592312064
mean quantityBase: 44295
mean quantityDiceCount: 214
mean quantityDieSides: 159
mean stats_0c: 779.599516114
mean stats_10: 437.020156302
mean stats_14: 585.162582166
mean stats_18: 862.003740502
mean stats_1c: 708.305973667
mean stats_20: 479.611103738
mean stats_24: 114.810380045
mean stats_28: 430.53892517
mean stats_2c: 134.25407344
mean stats_30: 776.821845629
mean stats_34: 761.081712881
mean stats_38: 799.969099671
mean stats_3c: 24.9990343647
mean stats_40: 612.013396855
mean stats_44: 619.420518148
mean stats_48: 771.266504659
mean stats_4c: 699.972962212
mean stats_50: 63.0741098383
mean stats_54: 312.024984478
mean stats_68: 561.089437963
mean stats_6c: 545.349305215
mean stats_70: 603.6803854
mean event_30: 79
mean event_34: 805
mean monsterStamina: 34
mean monsterFatigue: 1
mean c551ba0: 0.190032958984
mean a551ba0_3_0: 3.42059326172
mean a551ba0_3_1: 0
mean a551ba0_3_2: 0
mean a551ba0_3_3: 0
mean a551ba0_3_4: 0
mean a551ba0_3_5: 0
mean a551ba0_3_6: 0
mean a551ba0_3_7: 0
mean a551ba0_3_8: 0
mean a551ba0_3_9: 0
mean a551ba0_3_10: 0
mean a551ba0_3_11: 0
mean a551ba0_3_12: 0
mean a551ba0_4_0: 0.950164794922
mean a551ba0_4_1: 0
mean a551ba0_4_2: 0
mean a551ba0_4_3: 0
mean a551ba0_4_4: 0
mean a551ba0_4_5: 0
mean a551ba0_4_6: 0
mean a551ba0_4_7: 0
mean a551ba0_4_8: 0
mean a551ba0_4_9: 0
mean a551ba0_4_10: 0
mean a551ba0_4_11: 0
mean a551ba0_4_12: 0
mean a551ba0_5_0: 8.17141723633
mean a551ba0_5_1: 0
mean a551ba0_5_2: 0
mean a551ba0_5_3: 0
mean a551ba0_5_4: 0
mean a551ba0_5_5: 0
mean a551ba0_5_6: 0
mean a551ba0_5_7: 0
mean a551ba0_5_8: 0
mean a551ba0_5_9: 0
mean a551ba0_5_10: 0
mean a551ba0_5_11: 0
mean a551ba0_5_12: 0
mean a551ba0_6_0: 0
mean a551ba0_6_1: 0
mean a551ba0_6_2: 0
mean a551ba0_6_3: 0
mean a551ba0_6_4: 0
mean a551ba0_6_5: 0
mean a551ba0_6_6: 0
mean a551ba0_6_7: 0
mean a551ba0_6_8: 0
mean a551ba0_6_9: 0
mean a551ba0_6_10: 0
mean a551ba0_6_11: 0
mean a551ba0_6_12: 0
mean a551ba0_7_0: 1900.13955688
mean a551ba0_7_1: 0
mean a551ba0_7_2: 0
mean a551ba0_7_3: 0
mean a551ba0_7_4: 0
mean a551ba0_7_5: 0
mean a551ba0_7_6: 0
mean a551ba0_7_7: 0
mean a551ba0_7_8: 0
mean a551ba0_7_9: 0
mean a551ba0_7_10: 0
mean a551ba0_7_11: 0
mean a551ba0_7_12: 0
mean a551ba0_8_0: 0.190032958984
mean a551ba0_8_1: 0
mean a551ba0_8_2: 0
mean a551ba0_8_3: 0
mean a551ba0_8_4: 0
mean a551ba0_8_5: 0
mean a551ba0_8_6: 0
mean a551ba0_8_7: 0
mean a551ba0_8_8: 0
mean a551ba0_8_9: 0
mean a551ba0_8_10: 0
mean a551ba0_8_11: 0
mean a551ba0_8_12: 0
mean a551ba0_9_0: 0
mean a551ba0_9_1: 0
mean a551ba0_9_2: 0
mean a551ba0_9_3: 0
mean a551ba0_9_4: 0
mean a551ba0_9_5: 0
mean a551ba0_9_6: 0
mean a551ba0_9_7: 0
mean a551ba0_9_8: 0
mean a551ba0_9_9: 0
mean a551ba0_9_10: 0
mean a551ba0_9_11: 0
mean a551ba0_9_12: 0
mean a551ba0_10_0: 0
mean a551ba0_10_1: 0
mean a551ba0_10_2: 0
mean a551ba0_10_3: 0
mean a551ba0_10_4: 0
mean a551ba0_10_5: 0
mean a551ba0_10_6: 0
mean a551ba0_10_7: 0
mean a551ba0_10_8: 0
mean a551ba0_10_9: 0
mean a551ba0_10_10: 0
mean a551ba0_10_11: 0
mean a551ba0_10_12: 0
mean c5519c0: 0.190032958984
mean c551a20: 0
mean c5520d0: 0
mean a5520d0_2_0: 0
mean a5520d0_2_1: 0
mean a5520d0_2_2: 0
mean a5520d0_3_0: 0
mean a5520d0_3_1: 0
mean a5520d0_3_2: 0
mean a5520d0_4_0: 0
mean a5520d0_4_1: 0
mean a5520d0_4_2: 0
mean c552250: 0
mean a552250_3_0: 0
mean a552250_3_1: 0
mean a552250_3_2: 0
mean a552250_4_0: 0
mean a552250_4_1: 0
mean a552250_4_2: 0
mean c52b7e0: 0
mean a52b7e0_1: 0
mean a52b7e0_2: 0
mean a52b7e0_3: 0
mean c52af50: 0
mean a52af50_1: 0
mean a52af50_2: 0
mean a52af50_3: 0
mean a52af50_4: 0
mean c52b590: 0
mean a52b590_1: 0
mean a52b590_2: 0
mean a52b590_3: 0
mean c524780: 0
mean a524780_1: 0
mean a524780_2: 0
mean a524780_3: 0
mean c523c00: 0
mean a523c00_1: 0
mean a523c00_2: 0
mean a523c00_3: 0
mean a523c00_4: 0
mean a523c00_6: 0
mean c52e690: 0.0741098383442
mean a52e690_1: 39792273.3624
mean a52e690_2: 89028581.5236
mean a52e690_3: 0
mean a52e690_4: 0
mean a52e690_5: 9.41194946971
mean c58aad0: 0.0741098383442
mean a58aad0_1_0: 0.592878706753
mean a58aad0_1_1: 0
mean a58aad0_2_0: 251337118.027
mean a58aad0_2_1: 0
mean c58b410: 0
mean a58b410_1: 0
mean a58b410_2: 0
mean a58b410_3: 0
mean a58b410_4: 0
mean c593330: 0
in: durationMessage, message, statsOut, sourceKind, sourceIndex, sourceID, targetKind, targetIndex, monsterID, matchSlot, presence, memberNameColor, characterPrimaryState, characterName0, characterName1, templateIndex, monsterPrimaryState, monsterHP, monsterMaxStamina, monsterStamina, monsterCondition17, monsterFatigue, quantityBase, quantityDiceCount, quantityDieSides, chance0, chance1, chance2, chance3, chance4, chance5, chance6, chance7, chance8, chance9, chance10, chance11, chance12, chance13, chance14, chance15, effectStrength, effectPenalty, effectPower, durationBonusPercent, durationBase, durationPerPower, hostileMonsterCount, activeCharacterIndex, damageEventCookie, stats_0c, stats_10, stats_14, stats_18, stats_1c, stats_20, stats_24, stats_28, stats_2c, stats_30, stats_34, stats_38, stats_3c, stats_40, stats_44, stats_48, stats_4c, stats_50, stats_54, stats_68, stats_6c, stats_70, event_30, event_34, text383Ptr, text913_0, text913_1, r551ba0_0, r551ba0_1, r551ba0_2, r551ba0_3, r551ba0_4, r551ba0_5, r551ba0_6, r551ba0_7, r551ba0_8, r551ba0_9, r551ba0_10, r551ba0_11, r551ba0_12, r5519c0_0, r5519c0_1, r5519c0_2, r5519c0_3, r5519c0_4, r5519c0_5, r5519c0_6, r5519c0_7, r5519c0_8, r5519c0_9, r5519c0_10, r5519c0_11, r5519c0_12, r5519c0_13, r5519c0_14, r5519c0_15, r5519c0_16, r5519c0_17, r5519c0_18, r5519c0_19, r551a20_0, r551a20_1, r551a20_2, r551a20_3, r551a20_4, r551a20_5, r551a20_6, r551a20_7, r551a20_8, r551a20_9, r551a20_10, r551a20_11, r551a20_12, r551a20_13, r551a20_14, r551a20_15, r551a20_16, r551a20_17, r551a20_18, r551a20_19, r5520d0_0, r5520d0_1, r5520d0_2, w552250_0, w552250_1, w552250_2, r552250, r52b7e0, r52af50, r52b590, r524780, r523c00, r52e690, f.characterStatus13Block, f.characterClass, f.characterRace, f.templateStatus12Immune, f.templateCategory, f.resistResult, f.durationResult, f.messagesEnabled, f.difficulty, f.characterLevel, f.characterResistance_fire, f.characterResistance_water, f.characterResistance_air, f.characterResistance_earth, f.characterResistance_mental, f.characterResistance_divine, f.characterSkillGrowthFlag, f.characterReverseDifficulty, f.templateResistance_fire, f.templateResistance_water, f.templateResistance_air, f.templateResistance_earth, f.templateResistance_mental, f.templateResistance_divine, f.templateLevel, f.monsterResistanceModifier_fire, f.monsterResistanceModifier_water, f.monsterResistanceModifier_air, f.monsterResistanceModifier_earth, f.monsterResistanceModifier_mental, f.monsterResistanceModifier_divine, f.monsterDifficultyClass, f.sdr.r552250, f.sdr.r522fe0, f.sdr.C.buff5.dur, f.sdr.C.name0, f.sdr.C.name1, f.sdr.C.name2, f.sdr.P.present, f.sdr.P.f1, f.sdr.text0, f.sdr.text1, f.sdr.M.group, f.sdr.M.inCombat, f.sdr.M.maxHP, f.sdr.M.maxSP, f.sdr.M.cond0, f.sdr.M.cond1, f.sdr.M.cond2, f.sdr.M.cond3, f.sdr.M.cond4, f.sdr.M.cond5, f.sdr.M.cond6, f.sdr.M.cond7, f.sdr.M.cond8, f.sdr.M.cond9, f.sdr.M.cond10, f.sdr.M.cond11, f.sdr.M.cond12, f.sdr.M.cond13, f.sdr.M.cond14, f.sdr.M.cond15, f.sdr.M.cond16, f.sdr.M.cond17, f.sdr.M.cond18, f.sdr.M.buff0.power, f.sdr.M.buff0.mult, f.sdr.M.buff0.dur, f.sdr.M.buff1.power, f.sdr.M.buff1.mult, f.sdr.M.buff1.dur, f.sdr.M.buff2.power, f.sdr.M.buff2.mult, f.sdr.M.buff2.dur, f.sdr.M.buff3.power, f.sdr.M.buff3.mult, f.sdr.M.buff3.dur, f.sdr.M.buff4.power, f.sdr.M.buff4.mult, f.sdr.M.buff4.dur, f.sdr.M.buff5.power, f.sdr.M.buff5.mult, f.sdr.M.buff5.dur, f.sdr.M.buff6.power, f.sdr.M.buff6.mult, f.sdr.M.buff6.dur, f.sdr.M.buff7.power, f.sdr.M.buff7.mult, f.sdr.M.buff7.dur, f.sdr.M.attackPenalty, f.sdr.M.eff0.active, f.sdr.M.eff0.type, f.sdr.M.eff0.power, f.sdr.M.eff0.mult, f.sdr.M.eff1.active, f.sdr.M.eff1.type, f.sdr.M.eff1.power, f.sdr.M.eff1.mult, f.sdr.M.eff2.active, f.sdr.M.eff2.type, f.sdr.M.eff2.power, f.sdr.M.eff2.mult, f.sdr.M.eff3.active, f.sdr.M.eff3.type, f.sdr.M.eff3.power, f.sdr.M.eff3.mult, f.sdr.M.eff4.active, f.sdr.M.eff4.type, f.sdr.M.eff4.power, f.sdr.M.eff4.mult, f.sdr.M.eff5.active, f.sdr.M.eff5.type, f.sdr.M.eff5.power, f.sdr.M.eff5.mult, f.sdr.M.eff6.active, f.sdr.M.eff6.type, f.sdr.M.eff6.power, f.sdr.M.eff6.mult, f.sdr.M.eff7.active, f.sdr.M.eff7.type, f.sdr.M.eff7.power, f.sdr.M.eff7.mult, f.sdr.M.eff8.active, f.sdr.M.eff8.type, f.sdr.M.eff8.power, f.sdr.M.eff8.mult, f.sdr.M.eff9.active, f.sdr.M.eff9.type, f.sdr.M.eff9.power, f.sdr.M.eff9.mult, f.sdr.M.eff10.active, f.sdr.M.eff10.type, f.sdr.M.eff10.power, f.sdr.M.eff10.mult, f.sdr.M.eff11.active, f.sdr.M.eff11.type, f.sdr.M.eff11.power, f.sdr.M.eff11.mult, f.sdr.tpl.attr0, f.sdr.tpl.attr1, f.sdr.tpl.attr2, f.sdr.tpl.attr3, f.sdr.tpl.attr4, f.sdr.tpl.species, f.sdr.actor.fa4, f.sdr.actor.fa7, f.sdr.T.sp0.active, f.sdr.T.sp0.code, f.sdr.T.sp0.value, f.sdr.T.sp1.active, f.sdr.T.sp1.code, f.sdr.T.sp1.value, f.sdr.T.sp2.active, f.sdr.T.sp2.code, f.sdr.T.sp2.value, f.sdr.T.sp3.active, f.sdr.T.sp3.code, f.sdr.T.sp3.value, f.sdr.T.sp4.active, f.sdr.T.sp4.code, f.sdr.T.sp4.value, f.sdr.T.sp5.active, f.sdr.T.sp5.code, f.sdr.T.sp5.value, f.sdr.T.sp6.active, f.sdr.T.sp6.code, f.sdr.T.sp6.value, f.sdr.T.sp7.active, f.sdr.T.sp7.code, f.sdr.T.sp7.value, f.sdr.T.sp8.active, f.sdr.T.sp8.code, f.sdr.T.sp8.value, f.sdr.T.aura0.active, f.sdr.T.aura0.code, f.sdr.T.aura0.power, f.sdr.T.aura0.mult, f.sdr.T.aura1.active, f.sdr.T.aura1.code, f.sdr.T.aura1.power, f.sdr.T.aura1.mult, f.sdr.T.aura2.active, f.sdr.T.aura2.code, f.sdr.T.aura2.power, f.sdr.T.aura2.mult, f.sdr.T.aura3.active, f.sdr.T.aura3.code, f.sdr.T.aura3.power, f.sdr.T.aura3.mult, f.sdr.T.aura4.active, f.sdr.T.aura4.code, f.sdr.T.aura4.power, f.sdr.T.aura4.mult, f.sdr.T.aura5.active, f.sdr.T.aura5.code, f.sdr.T.aura5.power, f.sdr.T.aura5.mult, f.sdr.grpA.f2c, f.sdr.grpB.f2c, f.sdr.fRj, f.sdr.fLt, f.sdr.ft0, f.sdr.ft1, f.sdr.ft2, f.sdr.ft3, f.sdr.ft4, f.sdr.ft5, f.sdr.ft6, f.sdr.ft7, f.sdr.ft8, f.sdr.ft9, f.sdr.ft10, f.sdr.ft11, f.sdr.ft12, f.sdr.ft13, f.sdr.ft14, f.sdr.ft15, f.sdr.ft16, f.sdr.ft17, f.sdr.ft18, f.sdr.ft19, f.sdr.ft20, f.sdr.ft21, f.sdr.ft22, f.sdr.ft23, f.sdr.ft24, f.sdr.ft25, f.sdr.ft26, f.sdr.ft27, f.sdr.ft28, f.sdr.ft29, f.sdr.ft30, f.sdr.ft31, f.sdr.ft32, f.sdr.ft33, f.sdr.ft34, f.sdr.ft35, f.sdr.ft36, f.sdr.ft37, f.sdr.ft38, f.sdr.ft39, f.sdr.ft40, f.sdr.ft41, f.sdr.ft42, f.sdr.ft43, f.sdr.ft44, f.sdr.ft45, f.sdr.ft46, f.sdr.ft47, f.sdr.ft48, f.sdr.ft49, f.sdr.ft50, f.sdr.ft51, f.sdr.ft52, f.sdr.ft53, f.sdr.ft54, f.sdr.ft55, f.sdr.ft56, f.sdr.ft57, f.sdr.ft58, f.sdr.ft59, f.sdr.ft60, f.sdr.ft61, f.sdr.ft62, f.sdr.ft63, f.sdr.ft64, f.sdr.ft65, f.sdr.ft66, f.sdr.ft67, f.sdr.ft68, f.sdr.ft69, f.sdr.ft70, f.sdr.ft71, f.sdr.ft72, f.sdr.ft73, f.sdr.ft74, f.sdr.ft75, f.sdr.ft76, f.sdr.ft77, f.sdr.ft78, f.sdr.ft79, f.sdr.ft80, f.sdr.ft81, f.sdr.ft82, f.sdr.ft83, f.sdr.ft84, f.sdr.ft85, f.sdr.ft86, f.sdr.ft87, f.sdr.ft88, f.sdr.ft89, f.sdr.ft90, f.sdr.ft91, f.sdr.ft92, f.sdr.ft93, f.sdr.ft94, f.sdr.ft95, f.sdr.ft96, f.sdr.ft97, f.sdr.ft98, f.sdr.ft99, f.sdr.ft100, f.sdr.ft101, f.sdr.ft102, f.sdr.ft103, f.sdr.ft104, f.sdr.ft105, f.sdr.ft106, f.sdr.ft107, f.sdr.ft108, f.sdr.ft109, f.sdr.ft110, f.sdr.ft111, f.sdr.ft112, f.sdr.ft113, f.sdr.ft114, f.sdr.ft115, f.sdr.ft116, f.sdr.ft117, f.sdr.ft118, f.sdr.ft119, f.sdr.ft120, f.sdr.ft121, f.sdr.ft122, f.sdr.ft123, f.sdr.ft124, f.sdr.ft125, f.sdr.ft126, f.sdr.ft127, f.sdr.ft128, f.sdr.ft129, f.sdr.ft130, f.sdr.ft131, f.sdr.ft132, f.sdr.ft133, f.sdr.ft134, f.sdr.ft135, f.sdr.ft136, f.sdr.ft137, f.sdr.ft138, f.sdr.ft139, f.sdr.ft140, f.sdr.ft141, f.sdr.ft142, f.sdr.ft143, f.sdr.ft144, f.sdr.ft145, f.sdr.ft146, f.sdr.ft147, f.sdr.ft148, f.sdr.ft149, f.sdr.ft150, f.sdr.ft151, f.sdr.ft152, f.sdr.ft153, f.sdr.ft154, f.sdr.ft155, f.sdr.ft156, f.sdr.ft157, f.sdr.ft158, f.sdr.ft159, f.sdr.ft160, f.sdr.ft161, f.sdr.ft162, f.sdr.ft163, f.sdr.ft164, f.sdr.ft165, f.sdr.ft166, f.sdr.ft167, f.sdr.ft168, f.sdr.ft169, f.sdr.ft170, f.sdr.ft171, f.sdr.ft172, f.sdr.ft173, f.sdr.ft174, f.sdr.ft175, f.sdr.ft176, f.sdr.ft177, f.sdr.ft178, f.sdr.ft179, f.sdr.ft180, f.sdr.ft181, f.sdr.ft182, f.sdr.ft183, f.sdr.ft184, f.sdr.ft185, f.sdr.ft186, f.sdr.ft187, f.sdr.ft188, f.sdr.ft189, f.sdr.ft190, f.sdr.ft191, f.sdr.ft192, f.sdr.ft193, f.sdr.ft194, f.sdr.ft195, f.sdr.ft196, f.sdr.ft197, f.sdr.ft198, f.sdr.ft199, f.sdr.ft200, f.sdr.ft201, f.sdr.ft202, f.sdr.ft203, f.sdr.ft204, f.sdr.ft205, f.sdr.ft206, f.sdr.ft207, f.sdr.ft208, f.sdr.ft209, f.sdr.ft210, f.sdr.ft211, f.sdr.ft212, f.sdr.ft213, f.sdr.ft214, f.sdr.ft215, f.sdr.ft216, f.sdr.ft217, f.sdr.ft218, f.sdr.ft219, f.sdr.ft220, f.sdr.ft221, f.sdr.ft222, f.sdr.ft223, f.sdr.ft224, f.sdr.ft225, f.sdr.ft226, f.sdr.ft227, f.sdr.ft228, f.sdr.ft229, f.sdr.ft230, f.sdr.ft231, f.sdr.ft232, f.sdr.ft233, f.sdr.ft234, f.sdr.ft235, f.sdr.ft236, f.sdr.ft237, f.sdr.ft238, f.sdr.ft239, f.sdr.ft240, f.sdr.ft241, f.sdr.ft242, f.sdr.ft243, f.sdr.ft244, f.sdr.ft245, f.sdr.ft246, f.sdr.ft247, f.sdr.ft248, f.sdr.ft249, f.sdr.ft250, f.sdr.ft251, f.sdr.ft252, f.sdr.ft253, f.sdr.ft254, f.sdr.ft255, f.sdr.ft256, f.sdr.ft257, f.sdr.ft258, f.sdr.ft259, f.sdr.ft260, f.sdr.ft261, f.sdr.ft262, f.sdr.ft263, f.sdr.ft264, f.sdr.ft265, f.sdr.ft266, f.sdr.ft267, f.sdr.ft268, f.sdr.ft269, f.sdr.ft270, f.sdr.ft271, f.sdr.ft272, f.sdr.ft273, f.sdr.ft274, f.sdr.ft275, f.sdr.ft276, f.sdr.ft277, f.sdr.ft278, f.sdr.ft279, f.sdr.ft280, f.sdr.ft281, f.sdr.ft282, f.sdr.ft283, f.sdr.ft284, f.sdr.ft285, f.sdr.ft286, f.sdr.ft287, f.sdr.ft288, f.sdr.ft289, f.sdr.ft290, f.sdr.ft291, f.sdr.ft292, f.sdr.ft293, f.sdr.ft294, f.sdr.ft295, f.sdr.ft296, f.sdr.ft297, f.sdr.ft298, f.sdr.ft299, f.sdr.ft300, f.sdr.ft301, f.sdr.ft302, f.sdr.ft303, f.sdr.ft304, g.textPtr, g.b0d, g.b11, g.b15, g.a05, g.f69, g.f6d, g.e0, g.e1, g.e2, g.e3, g.e4, g.e5, g.e6, g.e7, g.e8, g.e9, g.e10, g.e11, g.e12, g.e13, g.e14, g.lvl, g.vt2, g.vt3, g.g544, g.g550, g.g51c, g.ret52af, g.ret522a, g.ret522b, g.ft0, g.ft1, g.ft2, g.ft3, g.ft4, g.ft5, g.ft6, g.ft7, g.ft8, g.ft9, g.ft10, g.ft11, g.ft12, g.ft13, g.ft14, g.ft15, g.ft16, g.ft17, g.ft18, g.ft19, g.ft20, g.ft21, g.ft22, g.ft23, g.ft24, g.ft25, g.ft26, g.ft27, g.ft28, g.ft29, g.ft30, g.ft31, g.ft32, g.ft33, g.ft34, g.ft35, g.ft36, g.ft37, g.ft38, g.ft39, g.ft40, g.ft41, g.ft42, g.ft43, g.ft44, g.ft45, g.ft46, g.ft47, g.ft48, g.ft49, g.ft50, g.ft51, g.ft52, g.ft53, g.ft54, g.ft55, g.ft56, g.ft57, g.ft58, g.ft59, g.ft60, g.ft61, g.ft62, g.ft63, g.ft64, g.ft65, g.ft66, g.ft67, g.ft68, g.ft69, g.ft70, g.ft71, g.ft72, g.ft73, g.ft74, g.ft75, g.ft76, g.ft77, g.ft78, g.ft79, g.ft80, g.ft81, g.ft82, g.ft83, g.ft84, g.ft85, g.ft86, g.ft87, g.ft88, g.ft89, g.ft90, g.ft91, g.ft92, g.ft93, g.ft94, g.ft95, g.ft96, g.ft97, g.ft98, g.ft99, g.ft100, g.ft101, g.ft102, g.ft103, g.ft104, g.ft105, g.ft106, g.ft107, g.ft108, g.ft109, g.ft110, g.ft111, g.ft112, g.ft113, g.ft114, g.ft115, g.ft116, g.ft117, g.ft118, g.ft119, g.ft120, g.ft121, g.ft122, g.ft123, g.ft124, g.ft125, g.ft126, g.ft127, g.ft128, g.ft129, g.ft130, g.ft131, g.ft132, g.ft133, g.ft134, g.ft135, g.ft136, g.ft137, g.ft138, g.ft139, g.ft140, g.ft141, g.ft142, g.ft143, g.ft144, g.ft145, g.ft146, g.ft147, g.ft148, g.ft149, g.ft150, g.ft151, g.ft152, g.ft153, g.ft154, g.ft155, g.ft156, g.ft157, g.ft158, g.ft159, g.ft160, g.ft161, g.ft162, g.ft163, g.ft164, g.ft165, g.ft166, g.ft167, g.ft168, g.ft169, g.ft170, g.ft171, g.ft172, g.ft173, g.ft174, g.ft175, g.ft176, g.ft177, g.ft178, g.ft179, g.ft180, g.ft181, g.ft182, g.ft183, g.ft184, g.ft185, g.ft186, g.ft187, g.ft188, g.ft189, g.ft190, g.ft191, g.ft192, g.ft193, g.ft194, g.ft195, g.ft196, g.ft197, g.ft198, g.ft199, g.ft200, g.ft201, g.ft202, g.ft203, g.ft204, g.ft205, g.ft206, g.ft207, g.ft208, g.ft209, g.ft210, g.ft211, g.ft212, g.ft213, g.ft214, g.ft215, g.ft216, g.ft217, g.ft218, g.ft219, g.ft220, g.ft221, g.ft222, g.ft223, g.ft224, g.ft225, g.ft226, g.ft227, g.ft228, g.ft229, g.ft230, g.ft231, g.ft232, g.ft233, g.ft234, g.ft235, g.ft236, g.ft237, g.ft238, g.ft239, g.ft240, g.ft241, g.ft242, g.ft243, g.ft244, g.ft245, g.ft246, g.ft247, g.ft248, g.ft249, g.ft250, g.ft251, g.ft252, g.ft253, g.ft254, g.ft255, g.ft256, g.ft257, g.ft258, g.ft259, g.ft260, g.ft261, g.ft262, g.ft263, g.ft264, g.ft265, g.ft266, g.ft267, g.ft268, g.ft269, g.ft270, g.ft271, g.ft272, g.ft273, g.ft274, g.ft275, g.ft276, g.ft277, g.ft278, g.ft279, g.ft280, g.ft281, g.ft282, g.ft283, g.ft284, g.ft285, g.ft286, g.ft287, g.ft288, g.ft289, g.ft290, g.ft291, g.ft292, g.ft293, g.ft294, g.ft295, g.ft296, g.ft297, g.ft298, g.ft299, g.ft300, g.ft301, g.ft302, g.ft303, g.ft304, g.fstamina, g.fstaMax, g.fenc, g.faa9, g.flevel, g.fflag
out: quantityBase, quantityDiceCount, quantityDieSides, stats_0c, stats_10, stats_14, stats_18, stats_1c, stats_20, stats_24, stats_28, stats_2c, stats_30, stats_34, stats_38, stats_3c, stats_40, stats_44, stats_48, stats_4c, stats_50, stats_54, stats_68, stats_6c, stats_70, event_30, event_34, monsterStamina, monsterFatigue, c551ba0, a551ba0_3_0, a551ba0_3_1, a551ba0_3_2, a551ba0_3_3, a551ba0_3_4, a551ba0_3_5, a551ba0_3_6, a551ba0_3_7, a551ba0_3_8, a551ba0_3_9, a551ba0_3_10, a551ba0_3_11, a551ba0_3_12, a551ba0_4_0, a551ba0_4_1, a551ba0_4_2, a551ba0_4_3, a551ba0_4_4, a551ba0_4_5, a551ba0_4_6, a551ba0_4_7, a551ba0_4_8, a551ba0_4_9, a551ba0_4_10, a551ba0_4_11, a551ba0_4_12, a551ba0_5_0, a551ba0_5_1, a551ba0_5_2, a551ba0_5_3, a551ba0_5_4, a551ba0_5_5, a551ba0_5_6, a551ba0_5_7, a551ba0_5_8, a551ba0_5_9, a551ba0_5_10, a551ba0_5_11, a551ba0_5_12, a551ba0_6_0, a551ba0_6_1, a551ba0_6_2, a551ba0_6_3, a551ba0_6_4, a551ba0_6_5, a551ba0_6_6, a551ba0_6_7, a551ba0_6_8, a551ba0_6_9, a551ba0_6_10, a551ba0_6_11, a551ba0_6_12, a551ba0_7_0, a551ba0_7_1, a551ba0_7_2, a551ba0_7_3, a551ba0_7_4, a551ba0_7_5, a551ba0_7_6, a551ba0_7_7, a551ba0_7_8, a551ba0_7_9, a551ba0_7_10, a551ba0_7_11, a551ba0_7_12, a551ba0_8_0, a551ba0_8_1, a551ba0_8_2, a551ba0_8_3, a551ba0_8_4, a551ba0_8_5, a551ba0_8_6, a551ba0_8_7, a551ba0_8_8, a551ba0_8_9, a551ba0_8_10, a551ba0_8_11, a551ba0_8_12, a551ba0_9_0, a551ba0_9_1, a551ba0_9_2, a551ba0_9_3, a551ba0_9_4, a551ba0_9_5, a551ba0_9_6, a551ba0_9_7, a551ba0_9_8, a551ba0_9_9, a551ba0_9_10, a551ba0_9_11, a551ba0_9_12, a551ba0_10_0, a551ba0_10_1, a551ba0_10_2, a551ba0_10_3, a551ba0_10_4, a551ba0_10_5, a551ba0_10_6, a551ba0_10_7, a551ba0_10_8, a551ba0_10_9, a551ba0_10_10, a551ba0_10_11, a551ba0_10_12, c5519c0, c551a20, c5520d0, a5520d0_2_0, a5520d0_2_1, a5520d0_2_2, a5520d0_3_0, a5520d0_3_1, a5520d0_3_2, a5520d0_4_0, a5520d0_4_1, a5520d0_4_2, c552250, a552250_3_0, a552250_3_1, a552250_3_2, a552250_4_0, a552250_4_1, a552250_4_2, c52b7e0, a52b7e0_1, a52b7e0_2, a52b7e0_3, c52af50, a52af50_1, a52af50_2, a52af50_3, a52af50_4, c52b590, a52b590_1, a52b590_2, a52b590_3, c524780, a524780_1, a524780_2, a524780_3, c523c00, a523c00_1, a523c00_2, a523c00_3, a523c00_4, a523c00_6, c52e690, a52e690_1, a52e690_2, a52e690_3, a52e690_4, a52e690_5, c58aad0, a58aad0_1_0, a58aad0_1_1, a58aad0_2_0, a58aad0_2_1, c58b410, a58b410_1, a58b410_2, a58b410_3, a58b410_4, c593330
```

結論：村正每下命中觸發即死 19.0%（c551ba0），其中 Marten 擋下 61.0%、過 39.0%，每下命中即死 7.41%（c52e690＝stats_50 增量 0.0741）。每回合機率要再乘每回合命中下數，L28 Samurai＋村正的建檔情境不在 notes/calc-qa/（只有 l28-c1／l28-c2），每回合值未取得。

### Q5 怪物攻擊：Dark Savant 每回合對 AC 10 前排的期望揮擊數與命中？
怪物揮擊數 0x53fea0：每個攻擊單位揮 R(template +0xe6)＋1 下，消耗戰鬥 +9 預算（0x5323f0 由 template +0xe5 拷入）。Dark Savant 讀值 +0xe5＝3、+0xe6＝2。情境取 re/emu/cases/0053fea0.json 列 0 改 budget=3、gateE5=3、swingRange=2。
```calc-qa
$ calc f 0053fea0 0 0 537108312 -1 0 -1 0 0 0 0 4 1 1 536936448 0 1 16 0 1 1 1 -1 -1 1 0 0 0 1 10 0 1 10 0 1 10 0 1 10 0 1 10 0 1 10 0 1 10 0 1 10 0 0 0 0 0 100 20 0 1 0 1 0 0 1 0 0 0 0 0 5 1 10 10 10 10 10 10 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 65535 4294967295 3 255 0 255 0 255 0 255 0 1 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 100 1 1 1 0 0 0 0 0 0 1 0 0 101 102 103 536936448 201 202 203 536936448 77 301 302 303 304 0 0 1 0 0 0 0 0 1 1 1 0 1 1 1 0 1 1 1 0 0 0 0 0 0 0 0 0 0 0 0 0 200 0 1084227584 0 1092616192 0 0 0 0 0 0 0 0 0 0 0 0 0 0 9 0 3 3 2 1 1 1 1 1 1 1 1 0 0 0 1 0 0 0 0 0 0 0 0 0 1148846080 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
dist
1,1,2,1,0,4294967295,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,9,0,0,1,1,1,0,0,0,4294967295,4294967295,4294967295,0,0,0,0,0,0,0: 0.500030517578
1,2,2,1,0,4294967295,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,9,0,0,1,1,1,0,0,0,4294967295,4294967295,4294967295,0,0,0,0,0,0,0: 0.499969482422
mean ret: 1
mean swingsOut: 1.49996948242
mean budgetOut: 2
mean mTarget0Out: 1
mean mTarget1Out: 0
mean mTarget2Out: 4294967295
mean mTarget3Out: 0
mean rearFlag: 0
mean specialFlag: 0
mean interceptAttemptsOut: 0
mean victimAttemptsOut: 0
mean face0Out: 0
mean face1Out: 0
mean face2Out: 0
mean face3Out: 0
mean face4Out: 0
mean face5Out: 0
mean face6Out: 0
mean face7Out: 0
mean rangeAfter: 0
mean rangedStateAfter: 0
mean animationAfter: 9
mean animationFrameAfter: 0
mean animationStartAfter: 0
mean animationPending: 1
mean animationPolicy: 1
mean animationCounterAfter: 1
mean animationAccessCounter: 0
mean animationTimeAfter: 0
mean targetWordAfter0: 0
mean targetWordAfter1: 4294967295
mean targetWordAfter2: 4294967295
mean targetWordAfter3: 4294967295
mean targetWordAfter4: 0
mean targetWordAfter5: 0
mean targetWordAfter6: 0
mean targetWordAfter7: 0
mean scratchHeadAfter: 0
mean scratchTailAfter: 0
mean scratchBeyondAfter: 0
in: characterSlot, interfaceMode, battle, battleAction, battleSpecial, selectedCharacter, combat, specialUsesSpell, specialUsesItem, mainAction, offAction, mainSpell, offSpell, itemInstance, itemIndex, itemSpell, itemType, spellTarget, spellRange, targetKind, targetCharacter, targetMonster, targetGroup, actionOverride, otherMode0, otherMode1, otherMode2, present0, hp0, status0, present1, hp1, status1, present2, hp2, status2, present3, hp3, status3, present4, hp4, status4, present5, hp5, status5, present6, hp6, status6, present7, hp7, status7, profession, race, gender, breathEffect, maxStamina, currentStamina, mainAttackKind, mainAttackAvailable, offAttackKind, offAttackAvailable, mainItemIndex, offItemIndex, weaponCharges, itemRange0, itemRange1, itemRange2, itemRange3, spellRealm, spellCost, spellLevel, realmSP0, realmSP1, realmSP2, realmSP3, realmSP4, realmSP5, learned1, learned2, learned3, learned4, learned5, learned6, learned7, learned8, learned9, learned10, learned11, learned12, learned13, learned14, learned15, learned16, learned17, learned18, learned19, learned20, learned21, learned22, learned23, learned24, learned25, learned26, learned27, learned28, learned29, learned30, learned31, learned32, learned33, learned34, learned35, learned36, learned37, learned38, learned39, learned40, learned41, learned42, learned43, learned44, learned45, learned46, learned47, learned48, learned49, learned50, learned51, learned52, learned53, learned54, learned55, learned56, learned57, learned58, learned59, learned60, learned61, learned62, learned63, learned64, learned65, learned66, learned67, learned68, learned69, learned70, learned71, learned72, learned73, learned74, learned75, learned76, learned77, learned78, learned79, learned80, learned81, learned82, learned83, learned84, learned85, learned86, learned87, learned88, learned89, learned90, learned91, learned92, learned93, learned94, learned95, learned96, learned97, learned98, learned99, learned100, learned101, learned102, learned103, learned104, learned105, learned106, learned107, learned108, learned109, learned110, learned111, learned112, learned113, professionMask, raceMask, genderMask, attributeId0, attributeMinimum0, attributeId1, attributeMinimum1, skillId0, skillMinimum0, skillId1, skillMinimum1, itemKind, instrumentLevel, deviceLevel, strength, intelligence, piety, vitality, dexterity, speed, senses, skill0, skill1, skill2, skill3, skill4, skill5, skill6, skill7, skill8, skill9, skill10, skill11, skill12, skill13, skill14, skill15, skill16, skill17, skill18, skill19, skill20, skill21, skill22, skill23, skill24, skill25, skill26, skill27, skill28, skill29, skill30, skill31, skill32, skill33, skill34, skill35, skill36, skill37, skill38, skill39, skill40, instanceCharges, instanceCount, gameStarted, itemFlags, weaponSubtype, weaponRechargeType, itemRechargeType, targetStamina, targetFatigue, actionCompleted, deferredDefense, refreshMask, mainTargetTail0, mainTargetTail1, mainTargetTail2, mainTargetTail3, offTargetTail0, offTargetTail1, offTargetTail2, offTargetTail3, mainParameter, mainTargetHead0, mainTargetHead1, mainTargetHead2, mainTargetHead3, interferenceStatus, interceptAttempts, targetType, monsterType, attackIndex, rangeType, templateClass, templateDisturb, attackE0k0, attackE1k0, attackE2k0, attackE3k0, attackE0k1, attackE1k1, attackE2k1, attackE3k1, attackE0k2, attackE1k2, attackE2k2, attackE3k2, spellRange0, spellRange1, spellRange2, spellRange3, spellRange4, spellRange5, swingCount, victimAlive, victimAwake, victimLevel, victimStun, victimTurnable, victimTurns, victimAttempts, attackerX, attackerZ, victimX, victimZ, partyX, partyZ, attackerFacing, victimFacing, partyHeading, face0, face1, face2, face3, face4, face5, face6, face7, notifyKind, verboseMode, budget, gateE5, swingRange, attackWord0, attackWord1, attackWord2, reachEnable, reach0, reach1, reach2, reach3, altFlag, altFlag2, monsterFlag15, monsterDisposition, combatFlag15, charm0, charm1, charm2, charm3, charm4, charm5, charm6, charm7, partyY, monsterY, monsterRadius, partyRadius, character0Quadrant, character1Quadrant, character2Quadrant, character3Quadrant, character4Quadrant, character5Quadrant, character6Quadrant, character7Quadrant, quadrant0Slot0, quadrant0Slot1, quadrant0Slot2, quadrant1Slot0, quadrant1Slot1, quadrant1Slot2, quadrant2Slot0, quadrant2Slot1, quadrant2Slot2, quadrant3Slot0, quadrant3Slot1, quadrant3Slot2, quadrant4Slot0, quadrant4Slot1, quadrant4Slot2, character0FormationUnavailable, character1FormationUnavailable, character2FormationUnavailable, character3FormationUnavailable, character4FormationUnavailable, character5FormationUnavailable, character6FormationUnavailable, character7FormationUnavailable, animationVariants, rangedState, sceneFlags, animationFrame, animationStart, animationTimeBits, animationCounter, targetWord0, targetWord1, targetWord2, targetWord3, targetWord4, targetWord5, targetWord6, targetWord7, scratchHead, scratchTail, scratchBeyond
out: ret, swingsOut, budgetOut, mTarget0Out, mTarget1Out, mTarget2Out, mTarget3Out, rearFlag, specialFlag, interceptAttemptsOut, victimAttemptsOut, face0Out, face1Out, face2Out, face3Out, face4Out, face5Out, face6Out, face7Out, rangeAfter, rangedStateAfter, animationAfter, animationFrameAfter, animationStartAfter, animationPending, animationPolicy, animationCounterAfter, animationAccessCounter, animationTimeAfter, targetWordAfter0, targetWordAfter1, targetWordAfter2, targetWordAfter3, targetWordAfter4, targetWordAfter5, targetWordAfter6, targetWordAfter7, scratchHeadAfter, scratchTailAfter, scratchBeyondAfter
```

結論：Dark Savant 每個攻擊單位揮 1 或 2 下各半，平均 1.50；預算 3 全用在近戰時每回合期望 4.5 下〔推測：三單位皆為揮擊〕。命中未取得：0x542720／0x540170 需要執行期輸入（怪物 +0x1dc／+0x1dd 架勢、疲勞、目標人物四維、AC 10 對應的 defhit），本題沒有可引用的情境；0x541630 是朝向／背面幾何，屬數值規劃排除範圍（decisions/2026-10-11-numeric-only-scope.md）。

## 各系統一題

### Q6 鑑定：Bishop 鑑定技能 60 鑑定 Excaliber 的成功機率？
技能鑑定 0x5208f0：讀人物 +0x49b（技能 0x14 的值，skill(j)=C+0x1a3+0x26j，notes/re/formulas.md:29）除以 6 取低 byte，對照物品定義 +0x85（有號 byte）；沒有亂數，也不讀職業。Excaliber＝ITEMS.DBS id 34（Q1 同一件），記錄 +0x85 讀值 17（`4+269×34+0x85`，loot.md:9／combat-2.md:85 的記錄位置）。情境取 re/emu/cases/005208f0.json 邊界列 0 改欄：itemId 1→34、itemRequiredSkill 10→17、charSkillScore 70→60（第二區塊 102）；技能池啟用 1、狀態 0 沿用。
```calc-qa
$ calc f 005208f0 0 536870912 34 17 1 0 60 0 0 0 0 0 0 0 0
dist
0,0,0,0,0,0,0,0,0,0,0,0,0: 1
mean flag6: 0
mean flag7: 0
mean flag8: 0
mean flag9: 0
mean flagA: 0
mean flagB: 0
mean flagC: 0
mean result: 0
mean trainCalls: 0
mean trainChar: 0
mean trainSkill: 0
mean trainRepeat: 0
mean trainQuiet: 0
in: slot, charAddr, itemId, itemRequiredSkill, charPoolsEnabled, charState, charSkillScore, flag6, flag7, flag8, flag9, flagA, flagB, flagC, trainRet
out: flag6, flag7, flag8, flag9, flagA, flagB, flagC, result, trainCalls, trainChar, trainSkill, trainRepeat, trainQuiet
```
```calc-qa
$ calc f 005208f0 0 536870912 34 17 1 0 102 0 0 0 0 0 0 0 0
dist
1,1,1,0,0,0,0,1,1,536870912,20,2,1: 1
mean flag6: 1
mean flag7: 1
mean flag8: 1
mean flag9: 0
mean flagA: 0
mean flagB: 0
mean flagC: 0
mean result: 1
mean trainCalls: 1
mean trainChar: 536870912
mean trainSkill: 20
mean trainRepeat: 2
mean trainQuiet: 1
in: slot, charAddr, itemId, itemRequiredSkill, charPoolsEnabled, charState, charSkillScore, flag6, flag7, flag8, flag9, flagA, flagB, flagC, trainRet
out: flag6, flag7, flag8, flag9, flagA, flagB, flagC, result, trainCalls, trainChar, trainSkill, trainRepeat, trainQuiet
```

法術路徑：0x520a70 已登記的呼叫者是自動 PL 估算（0x4fe480，notes/re/iface/4fe480.tsv，23 號法術取副表物品實例），算的是「建議用幾級施法」，不是成功率。re/emu/cases/00520a70.json 邊界列 3 改欄：itemIndex 0→34、identifyRequirement 0→17，bonusPercent 0 沿用（fixture 值，原版加成來源未查）。
```calc-qa
$ calc f 00520a70 34 17 0
dist
6: 1
mean identifyPowerRequired: 6
in: itemIndex, identifyRequirement, bonusPercent
out: identifyPowerRequired
```

施法本身的成功率在 0x4fa4d0（D0），法術鑑定效果的執行入口也未定位（abilities.md Bishop 列）；本題不估。

結論：技能鑑定是確定性門檻，技能 60（60/6=10 < 17）鑑定 Excaliber 成功率 0%；需技能 ≥102 才成功（102–119 另練技能 0x14，≥120 再設實例 +0xb 旗標），與 Bishop 職業無關；法術鑑定的自動 PL 是 6，成功率待 0x4fa4d0。

### Q7 扒竊：Rogue 扒竊技能 50 對一般 NPC 的成功率？
門檻 0x50baf0：讀扒竊技能 C+0x391（技能 13＝0xd，0x50bc90 成功時練的就是 0xd）。情境取 re/emu/cases/0050baf0.json 邊界列 1（職業 7）改欄：stealBaseRating 100→50、characterTotalLevel 10→3（與 NPC 連結怪物模板等級 3 相同，等級差項歸 0）；DEX 50、態度 50（中立，項為 0）、首次嘗試（npc+0x84=0）、難度 1（不縮放）、itemId −1（偷錢）沿用，均為情境值，不是特定 NPC 的資料。
```calc-qa
$ calc f 0050baf0 7 50 50 3 0 1 1 5 3 100 0 1 50 0 -1 0
dist
55,1,0,0,0: 1
mean chance: 55
mean attitudeCalls: 1
mean itemInitCalls: 0
mean itemInitId: 0
mean itemValueCalls: 0
in: characterClass, stealBaseRating, effectiveDexterity, characterTotalLevel, npcStealAttempts, monsterLinkEnabled, linkedMonsterAvailable, linkedMonsterIdentity, monsterTemplateLevel, unitWeight, reverseDifficultyState, difficulty, attitudeRet, itemValueRet, itemId, quantity
out: chance, attitudeCalls, itemInitCalls, itemInitId, itemValueCalls
```

結算 0x50bc90：re/emu/cases/0050bc90.json 邊界列 0 改欄：characterClass 0→7、stealBaseRating 40→50、effectiveDexterity 100→50、characterTotalLevel 10→3，npcMoney 50→0。NPC 無物品（40 槽皆 0xffff）＋錢 0，成功分支回 4（沒東西可偷），所以 result 4＝成功、2＝失手未被發現、3＝被抓。
```calc-qa
$ calc f 0050bc90 0 536875008 7 50 50 3 0 1 0 0 0 0 0 0 1000 1001 1002 1003 1004 1005 0 0 0 1 1 5 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 65535 3 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 50 100 536953344
dist
2,1515870810,0,1,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,0,1,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,2779096485,0,0,0,0,0,0,0,0,0: 0.410003662109
3,1515870810,0,1,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,0,1,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,2779096485,0,0,0,0,0,0,0,0,0: 0.0399780273438
4,1515870810,0,1,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,65535,0,1,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,2779096485,0,0,0,0,0,0,0,0,0: 0.550018310547
mean result: 3.14001464844
mean moneyOut: 1515870810
mean npcMoney: 0
mean npcStealAttempts: 1
mean npcItem0: 65535
mean npcItem1: 65535
mean npcItem2: 65535
mean npcItem3: 65535
mean npcItem4: 65535
mean npcItem5: 65535
mean npcItem6: 65535
mean npcItem7: 65535
mean npcItem8: 65535
mean npcItem9: 65535
mean npcItem10: 65535
mean npcItem11: 65535
mean npcItem12: 65535
mean npcItem13: 65535
mean npcItem14: 65535
mean npcItem15: 65535
mean npcItem16: 65535
mean npcItem17: 65535
mean npcItem18: 65535
mean npcItem19: 65535
mean npcItem20: 65535
mean npcItem21: 65535
mean npcItem22: 65535
mean npcItem23: 65535
mean npcItem24: 65535
mean npcItem25: 65535
mean npcItem26: 65535
mean npcItem27: 65535
mean npcItem28: 65535
mean npcItem29: 65535
mean npcItem30: 65535
mean npcItem31: 65535
mean npcItem32: 65535
mean npcItem33: 65535
mean npcItem34: 65535
mean npcItem35: 65535
mean npcItem36: 65535
mean npcItem37: 65535
mean npcItem38: 65535
mean npcItem39: 65535
mean seed: 0
mean seedCalls: 1
mean attitudeCalls: 1
mean valueCalls: 0
mean initCalls: 0
mean initId0: 0
mean initId1: 0
mean initId2: 0
mean initId3: 0
mean initQty0: 0
mean initQty1: 0
mean initQty2: 0
mean initQty3: 0
mean initA40: 0
mean initA41: 0
mean initA42: 0
mean initA43: 0
mean initA50: 0
mean initA51: 0
mean initA52: 0
mean initA53: 0
mean itemOutId: 2779096485
mean skillCalls: 0
mean skillChar0: 0
mean skillChar1: 0
mean skillId0: 0
mean skillId1: 0
mean skillA30: 0
mean skillA31: 0
mean skillA40: 0
mean skillA41: 0
in: slot, charPtr, characterClass, stealBaseRating, effectiveDexterity, characterTotalLevel, reverseDifficultyState, difficulty, seedPresent0, seedPresent1, seedPresent2, seedPresent3, seedPresent4, seedPresent5, seedValue0, seedValue1, seedValue2, seedValue3, seedValue4, seedValue5, npcByte2e, npcMoney, npcStealAttempts, monsterLinkEnabled, linkedMonsterAvailable, linkedMonsterIdentity, npcItem0, npcItem1, npcItem2, npcItem3, npcItem4, npcItem5, npcItem6, npcItem7, npcItem8, npcItem9, npcItem10, npcItem11, npcItem12, npcItem13, npcItem14, npcItem15, npcItem16, npcItem17, npcItem18, npcItem19, npcItem20, npcItem21, npcItem22, npcItem23, npcItem24, npcItem25, npcItem26, npcItem27, npcItem28, npcItem29, npcItem30, npcItem31, npcItem32, npcItem33, npcItem34, npcItem35, npcItem36, npcItem37, npcItem38, npcItem39, monsterTemplateLevel, unitWeight0, unitWeight1, unitWeight2, unitWeight3, unitWeight4, unitWeight5, unitWeight6, unitWeight7, unitWeight8, unitWeight9, unitWeight10, unitWeight11, unitWeight12, unitWeight13, unitWeight14, unitWeight15, attitudeRet, itemValueRet, itemOutPtr
out: result, moneyOut, npcMoney, npcStealAttempts, npcItem0, npcItem1, npcItem2, npcItem3, npcItem4, npcItem5, npcItem6, npcItem7, npcItem8, npcItem9, npcItem10, npcItem11, npcItem12, npcItem13, npcItem14, npcItem15, npcItem16, npcItem17, npcItem18, npcItem19, npcItem20, npcItem21, npcItem22, npcItem23, npcItem24, npcItem25, npcItem26, npcItem27, npcItem28, npcItem29, npcItem30, npcItem31, npcItem32, npcItem33, npcItem34, npcItem35, npcItem36, npcItem37, npcItem38, npcItem39, seed, seedCalls, attitudeCalls, valueCalls, initCalls, initId0, initId1, initId2, initId3, initQty0, initQty1, initQty2, initQty3, initA40, initA41, initA42, initA43, initA50, initA51, initA52, initA53, itemOutId, skillCalls, skillChar0, skillChar1, skillId0, skillId1, skillA30, skillA31, skillA40, skillA41
```

規則（n_50bc90.go 步驟 6）：roll=R(100)，roll>95 或 roll>2×門檻→被抓；roll≥門檻→失手；否則成功。故成功率＝min(門檻,96)%，被抓至少 4%。門檻各項：+DEX/10（Rogue；他職且技能>0 為 /20）、+(態度−50)/4、−(10..14)×已嘗試次數、+5×(人物總等級−NPC 怪物等級)、偷物品另扣 max(0, 重量×數量/10＋價值/100−技能/4)，難度 0／2 乘 140%／60%，最後夾 0..127。

結論：Rogue 扒竊 50、DEX 50、與 NPC 同等級、首次、中立態度、普通難度偷錢：門檻 55，成功 55%、失手 41%、被抓 4%；人物每比 NPC 高一級 +5%（低一級 −5%），每多試一次約 −12%，上限 96%。

### Q8 施法：Mage L10 Fire 技能 50 施 PL3 Fireball 的成功率？
施法失敗風險在 0x4ff7f0 與 0x4fb4c0 共用同一式（0x501910 進度縮放）：weighted＝(學派基礎＋4×realm 有效)/5，低於 threshold[費用/2＋等級]×PL/7 才有風險 risk＝70×(t−w)/t，再加等級不足項；risk＝0 時 0x4fb4c0 不擲骰（n_4fb4c0.go:347），成功率＝1−risk/100。0x4ff7f0 只在 risk＝0 時回傳可訓練學派（24），否則回最高學派（此處以 Divinity 60 當對照，25），故輸出 24＝無失敗風險。Fireball＝SPELLTABLES id 36：費用 6、等級 3、realm 0（Fire）；spellIndex 36（2026-10-11 F5 起節點域 0..149；原取 0，上限 15）。Mage＝職業 14，等級 10；戰鬥中、難度 2、進度 100；Wizardry 0（最壞情形）。第二區塊 Fire 39 對照。
```calc-qa
$ calc f 004ff7f0 36 1 1 3 0 0 1 1 0 6 3 0 14 1 2 100 1 1 0 0 0 60 60 0 0 0 0 0 0 50 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 10
dist
24: 1
mean skillIndex: 24
in: spellIndex, casting, training, powerLevel, verbalCastingBlocked, alchemyUsable, wizardryUsable, divinityUsable, psionicUsable, spellCost, spellLevel, realm, currentClass, battle, difficulty, actionProgress, characterPresent, school0Enabled, school0Effective, school0Base, school1Enabled, school1Effective, school1Base, school2Enabled, school2Effective, school2Base, school3Enabled, school3Effective, school3Base, realm0Effective, realm1Effective, realm2Effective, realm3Effective, realm4Effective, realm5Effective, classLevel0, classLevel1, classLevel2, classLevel3, classLevel4, classLevel5, classLevel6, classLevel7, classLevel8, classLevel9, classLevel10, classLevel11, classLevel12, classLevel13, classLevel14
out: skillIndex
```
```calc-qa
$ calc f 004ff7f0 36 1 1 3 0 0 1 1 0 6 3 0 14 1 2 100 1 1 0 0 0 60 60 0 0 0 0 0 0 39 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 10
dist
25: 1
mean skillIndex: 25
in: spellIndex, casting, training, powerLevel, verbalCastingBlocked, alchemyUsable, wizardryUsable, divinityUsable, psionicUsable, spellCost, spellLevel, realm, currentClass, battle, difficulty, actionProgress, characterPresent, school0Enabled, school0Effective, school0Base, school1Enabled, school1Effective, school1Base, school2Enabled, school2Effective, school2Base, school3Enabled, school3Effective, school3Base, realm0Effective, realm1Effective, realm2Effective, realm3Effective, realm4Effective, realm5Effective, classLevel0, classLevel1, classLevel2, classLevel3, classLevel4, classLevel5, classLevel6, classLevel7, classLevel8, classLevel9, classLevel10, classLevel11, classLevel12, classLevel13, classLevel14
out: skillIndex
```

結論：Mage L10、Fire 50 施 PL3 Fireball 失敗風險 0，成功率 100%（Wizardry 取 0 的最壞情形；Fire 40 仍為 0，39 起開始有風險）。0x4fa4d0 全流程組合跑不起來（0x4ff220 上限 spellIndex≤3、sourceMonsterID≤7，人物施法 +8＝0xffffffff 必 panic），本題以 0x4ff7f0 的 risk＝0 指示代答。

補（2026-10-11，F5 24bb55f 後）：0x4fa4d0 全流程已可跑：`bash scripts/nodes/f4-probe.sh` 產同情境輸入（728 欄，輸出 1643 行，不貼入），calc 結果所有分支 ret＝1（未失敗、未反噬），成功率 100%，與上方 0x4ff7f0 代答一致；上段「跑不起來」已過時。

## MG 玩法假說（2026-10-09／10，未入題庫；Q4–Q8 綠後再轉成 calc-qa 題）
MG 原話：
- 「法術拿來輔助比較有用」
- 「即死在高難度打折，固定傷害相對靠譜」
- 「遠程其實我自己開 saveeditor 實測，要不是距離一遠就不中就是中了打不穿」
- 「有個比較粗暴的玩法是每個人扣掉拿詛咒武器（無法切換副手），全部都放棄遠程，就拿近戰（short）跟近戰（extend）類別的武器面對敵人，戰鬥開始放完 haste 就要衝到近戰距離了」
- 「拿 muse 搜尋到…的 saveeditor 實際改了滿級 powercast 所有屬性全滿的角色轟出去依然是笑話。拿你建議的盜賊（種族可能還沒差），一樣屬性全滿，進階例如 powerstrike、powercast 等也全滿，拿光劍跟盜賊匕首一回合打最高等級的 ascention peak 的高級怪可以轟出 300 點以上的 HP」（2026-10-10 實測；對照 L28-ranking.md:45 Rogue 雙持 LS＋匕首對 Dark Savant 期望 424／回合）

可轉成的題：即死率對難度、遠程命中與穿透對距離、全近戰隊 haste 後幾回合進到 short／extend 距離（4f0af0 移動距離上限已 D4）、同隊近戰對遠程每回合期望傷害。
**公式完成後第一個試算（MG 2026-10-10：「這個之後公式完成後下可以先跑這個試算」）**：全近戰隊打法三項——①全隊 haste 後幾回合進到 short／extend 距離（4f0af0 已 D4）②同隊全近戰對保留遠程的每回合期望傷害差 ③詛咒武器持有者不能切副手，當固定配裝。
- 「法術輔助較有用」的既有數字：`notes/research/ranking-history.md:131`（傷害法術對抗性 100 只剩約兩成到四分之一）。三類分流：傷害法術過 552250 抗性削減（已 D4）；對隊伍 buff 不過怪物抗性；對怪 debuff 過 5520d0 狀態抵抗（乘難度，D0）。5520d0 D4 後一起轉題。
