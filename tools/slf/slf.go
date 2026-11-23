package main

import (
	"io"
	"wiz8-mod/internal/slf"
)

func listArchive(path string, out io.Writer) error { return slf.ListArchive(path, out) }
func extractArchive(path, dest string) error       { return slf.ExtractArchive(path, dest) }
func packArchive(src, dest string) error           { return slf.PackArchive(src, dest) }
