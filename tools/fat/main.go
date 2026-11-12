// Command fat joins Mach-O files into a universal binary.
// Format: <mach-o/fat.h>, all fields big-endian uint32.
package main

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
	"fmt"
	"os"
)

const (
	fatMagic = 0xcafebabe
	// 16 KB pages, as lipo uses.
	alignBits = 14
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: fat <out> <mach-o>...")
		os.Exit(2)
	}
	if err := join(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "fat:", err)
		os.Exit(1)
	}
}

func join(out string, inputs []string) error {
	type slice struct {
		cpu, sub uint32
		data     []byte
	}
	var slices []slice
	for _, name := range inputs {
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		file, err := macho.NewFile(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		slices = append(slices, slice{uint32(file.Cpu), file.SubCpu, data})
	}

	const align = 1 << alignBits
	var header, body bytes.Buffer
	put := func(values ...uint32) {
		for _, v := range values {
			_ = binary.Write(&header, binary.BigEndian, v)
		}
	}
	put(fatMagic, uint32(len(slices)))
	offset := uint32(align)
	for _, s := range slices {
		put(s.cpu, s.sub, offset, uint32(len(s.data)), alignBits)
		body.Write(s.data)
		padded := (uint32(len(s.data)) + align - 1) &^ (align - 1)
		body.Write(make([]byte, padded-uint32(len(s.data))))
		offset += padded
	}
	if header.Len() > align {
		return fmt.Errorf("header is %d bytes, over %d", header.Len(), align)
	}
	header.Write(make([]byte, align-header.Len()))
	return os.WriteFile(out, append(header.Bytes(), body.Bytes()...), 0o755)
}
