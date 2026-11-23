// strlib: Wizardry 8 索引字串庫（描述 DBS 與 MSG）的 decode／encode。
//
//	strlib decode <in>             stdout 輸出 JSON 物件（header＋tail＋entries）
//	strlib encode <in.json> <out>
package main

import (
	"fmt"
	"io"
	"os"
	"wiz8-mod/internal/strlib"
)

func run(args []string, out io.Writer) (int, error) {
	var err error
	switch {
	case len(args) == 2 && args[0] == "decode":
		err = strlib.DecodeFile(args[1], out)
	case len(args) == 3 && args[0] == "encode":
		err = strlib.EncodeFile(args[1], args[2])
	default:
		return 2, fmt.Errorf("usage: strlib decode <in> | strlib encode <in.json> <out>")
	}
	if err != nil {
		return 2, fmt.Errorf("%s: %w", args[1], err)
	}
	return 0, nil
}

func main() {
	code, err := run(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "strlib:", err)
	}
	os.Exit(code)
}
