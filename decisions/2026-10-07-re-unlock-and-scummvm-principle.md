# 2026-10-07 反組譯解禁＋ScummVM 式原則

## MG 原話
1. 「解禁，該做什麼做什麼，以目標導向，這個做完了也不會是商用所以不用擔心」（回應「Ghidra 要不要解禁」）
2. 「我們的設計原則模仿 ScummVm 的概念，使用需要配合原版軟體」

## 裁定（取代 goals/stage1.md:15「不開 Ghidra／反組譯器，不做任何 hook」，僅自第二階段起）
- 允許：Ghidra 等反組譯／反編譯工具分析自己這份 Wiz8.exe 與附帶 dll；寫 hook（ddraw 代理內 detour／vtable 掛勾）。目標導向：需要什麼就做什麼。
- 公開反組譯專案（agluszak/wizardry-8-decomp，A2／A3 引用）可讀作定位參考。
- ScummVM 原則：本 mod 必須搭配使用者自有的原版遊戲才能用。repo 與任何產出物不含原版 exe、資料、美術、音樂、文字；只含我們的程式、patch 規則、自製字型圖集、翻譯文字。
- 不變：原版遊戲目錄除實機測試暫放的我方檔案外只讀，放入前後 hashdir；原版資料一個 byte 不改（改動以 PATCH.00N／hook 疊上）。
