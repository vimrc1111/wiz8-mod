// slf: Wizardry 8 的 SLF／PATCH.00N 封存工具。格式見 notes/research/2026-10-07-A1-slf-patch-format.md。
//
//	slf list <檔>                 每筆一行：offset<TAB>size<TAB>name
//	slf extract <檔> <輸出目錄>
//	slf pack <extract 出的目錄> <輸出檔>
package main

import (
	"fmt"
	"io"
	"os"
	"wiz8-mod/internal/slf"
)

func run(args []string, out io.Writer) (int, error) {
	var err error
	switch {
	case len(args) == 2 && args[0] == "list":
		err = slf.ListArchive(args[1], out)
	case len(args) == 3 && args[0] == "extract":
		err = slf.ExtractArchive(args[1], args[2])
	case len(args) == 3 && args[0] == "pack":
		err = slf.PackArchive(args[1], args[2])
	default:
		return 2, fmt.Errorf("usage: slf list <檔> | slf extract <檔> <輸出目錄> | slf pack <目錄> <輸出檔>")
	}
	if err != nil {
		return 2, fmt.Errorf("%s: %w", args[1], err)
	}
	return 0, nil
}

func main() {
	code, err := run(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "slf:", err)
	}
	os.Exit(code)
}
