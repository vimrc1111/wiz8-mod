// nsf: Wizardry 8 NPC 對白腳本（NPC SCRIPTS\*.NSF）的 decode／encode。
//
//	nsf decode <in>             stdout 輸出 JSON 物件（segments）
//	nsf encode <in.json> <out>
package main

import (
	"fmt"
	"io"
	"os"
	codec "wiz8-mod/internal/nsf"
)

func run(args []string, out io.Writer) (int, error) {
	var err error
	switch {
	case len(args) == 2 && args[0] == "decode":
		err = codec.DecodeFile(args[1], out)
	case len(args) == 3 && args[0] == "encode":
		err = codec.EncodeFile(args[1], args[2])
	default:
		return 2, fmt.Errorf("usage: nsf decode <in> | nsf encode <in.json> <out>")
	}
	if err != nil {
		return 2, fmt.Errorf("%s: %w", args[1], err)
	}
	return 0, nil
}

func main() {
	code, err := run(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nsf:", err)
	}
	os.Exit(code)
}
