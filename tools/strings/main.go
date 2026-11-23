// strings: Wizardry 8 字串庫（Data\Strings\StringData.DAT）的 decode／encode。格式見 tickets/C-strings-tool.md。
//
//	strings decode <in.dat>            stdout 輸出 JSON 陣列（每筆一個字串，不含結尾 NUL）
//	strings encode <in.json> <out.dat>
package main

import (
	"fmt"
	"io"
	"os"
	"wiz8-mod/internal/stringdata"
)

func run(args []string, out io.Writer) (int, error) {
	var err error
	switch {
	case len(args) == 2 && args[0] == "decode":
		err = stringdata.DecodeFile(args[1], out)
	case len(args) == 3 && args[0] == "encode":
		err = stringdata.EncodeFile(args[1], args[2])
	default:
		return 2, fmt.Errorf("usage: strings decode <in.dat> | strings encode <in.json> <out.dat>")
	}
	if err != nil {
		return 2, fmt.Errorf("%s: %w", args[1], err)
	}
	return 0, nil
}

func main() {
	code, err := run(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "strings:", err)
	}
	os.Exit(code)
}
