// trcheck：譯文表機檢（stage5 5.2）與術語代號展開。規格見 tickets/E-trcheck.md。
package main

import (
	"io"
	"os"
)

func run(args []string, out io.Writer) int {
	return runTrcheck(args, out)
}

func main() { os.Exit(run(os.Args[1:], os.Stdout)) }
