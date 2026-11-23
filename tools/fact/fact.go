package main

import (
	"io"
	codec "wiz8-mod/internal/fact"
)

func decodeFile(path string, out io.Writer) error { return codec.DecodeFile(path, out) }
func encodeFile(src, dest string) error           { return codec.EncodeFile(src, dest) }
