// dbsname edits item, monster and spell names in fixed-size Wizardry 8 DBS records.
package main

import (
	"fmt"
	"io"
	"os"

	"wiz8-mod/internal/dbsname"
)

func run(args []string, out io.Writer) (int, error) {
	var err error
	switch {
	case len(args) == 3 && args[0] == "decode":
		err = dbsname.DecodeFile(args[1], args[2], out)
	case len(args) == 4 && args[0] == "encode":
		err = dbsname.EncodeFile(args[1], args[2], args[3])
	default:
		return 2, fmt.Errorf("usage: dbsname decode <items|monsters|spells> <in> | dbsname encode <items|monsters|spells> <in.json> <out>")
	}
	if err != nil {
		return 2, fmt.Errorf("%s: %w", args[2], err)
	}
	return 0, nil
}

func main() {
	code, err := run(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dbsname:", err)
	}
	os.Exit(code)
}
