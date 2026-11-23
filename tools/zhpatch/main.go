// zhpatch: 玩家端安裝用。從玩家自有的原版（遊戲目錄）＋本專案譯文產生 Patches/PATCH.002，等同 scripts/build-zh-patch.sh 的產物（不跑機檢），單一執行檔、不需 bash／jq／Go。
//
//	zhpatch <遊戲目錄> <translations 目錄> <輸出 PATCH 檔>
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: zhpatch <遊戲目錄> <translations 目錄> <輸出 PATCH 檔>")
		os.Exit(2)
	}
	if err := build(os.Args[1], os.Args[2], os.Args[3]); err != nil {
		fmt.Fprintln(os.Stderr, "zhpatch:", err)
		os.Exit(2)
	}
}
