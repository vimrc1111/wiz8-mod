// fitcheck：譯文像素寬與明示換行數不得超過原文（stage5 5.3）。規格見 tickets/G-fitcheck.md。
package main

import (
	"os"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout)) }
