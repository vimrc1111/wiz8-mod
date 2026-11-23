package main

import (
	"io"
	"wiz8-mod/internal/stringdata"
)

func decodeFile(path string, out io.Writer) error { return stringdata.DecodeFile(path, out) }
func encodeFile(src, dest string) error           { return stringdata.EncodeFile(src, dest) }
