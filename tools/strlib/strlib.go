package main

import (
	"io"
	"wiz8-mod/internal/strlib"
)

func decodeFile(path string, out io.Writer) error { return strlib.DecodeFile(path, out) }
func encodeFile(src, dest string) error           { return strlib.EncodeFile(src, dest) }
