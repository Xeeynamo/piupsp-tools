package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type entry struct {
	name    string
	unk00   uint32
	crc32_1 uint32
	unk08   uint32
	length  uint32
	crc32_2 uint32
	data    []byte
}

const nEntrySize = 0x12C

var (
	ErrInvalidRespack = errors.New("not a PIU RESPACK file")
)

func main() {
	var inPath string
	var outPath string
	var mode string

	args := os.Args[1:]
	switch len(args) {
	case 3:
		outPath = args[2]
		fallthrough
	case 2:
		inPath = args[1]
		mode = args[0]
	default:
		printHelp()
		os.Exit(1)
	}

	switch mode {
	case "x":
		if len(outPath) == 0 {
			name := strings.TrimSuffix(filepath.Base(inPath), filepath.Ext(inPath))
			outPath = path.Join(outPath, name)
		}
		if err := unpack(inPath, outPath); err != nil {
			panic(err)
		}
	case "p":
		if len(outPath) == 0 {
			outPath = filepath.Base(inPath) + ".DAT"
		}

		if err := pack(inPath, outPath); err != nil {
			panic(err)
		}
	default:
		fmt.Fprintf(os.Stderr, "mode '%s' not recognized\n", mode)
		printHelp()
		os.Exit(1)
	}

	os.Exit(0)
}

func printHelp() {
	fmt.Fprint(os.Stderr, "usage:\n")
	fmt.Fprint(os.Stderr, "   piu-unpack x file_name.DAT [output_path]\n")
	fmt.Fprint(os.Stderr, "   piu-unpack p input_path [out.DAT]\n")
}

func pack(inPath string, outFileName string) error {
	files, err := ioutil.ReadDir(inPath)
	if err != nil {
		return err
	}

	entries := make([]entry, 0)
	for _, file := range files {
		if len(file.Name()) >= 0x100 {
			return fmt.Errorf("file name '%s' is too long (256 char limit)", file.Name())
		}

		f, err := os.Open(path.Join(inPath, file.Name()))
		if err != nil {
			return err
		}
		defer f.Close()

		var buffer bytes.Buffer
		if _, err := deflate(f, &buffer); err != nil {
			return err
		}

		entries = append(entries, entry{
			name:   file.Name(),
			unk08:  1,
			length: uint32(file.Size()),
			data:   buffer.Bytes(),
		})
	}

	f, err := os.Create(outFileName)
	if err != nil {
		return err
	}
	defer f.Close()

	dataHeader := make([]byte, len(entries)*nEntrySize)
	var offset uint32
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		cursor := i * nEntrySize
		cmpLen := uint32(len(entry.data))
		setCString(dataHeader[cursor:], entry.name)
		setUint32(dataHeader[cursor+0x100:], entry.unk00)
		setUint32(dataHeader[cursor+0x104:], entry.crc32_1)
		setUint32(dataHeader[cursor+0x108:], entry.unk08)
		setUint32(dataHeader[cursor+0x10C:], entry.length)
		setUint32(dataHeader[cursor+0x110:], cmpLen)
		setUint32(dataHeader[cursor+0x114:], entry.crc32_2)
		setUint32(dataHeader[cursor+0x128:], offset)
		offset += cmpLen + 0x118
	}

	decryptHeader(dataHeader, uint32(len(entries)))

	f.Write([]byte{'R', 'E', 'S', 'P', 'A', 'C', 'K', 0x1A})
	binary.Write(f, binary.LittleEndian, uint32(0))
	binary.Write(f, binary.LittleEndian, uint32(len(entries)))
	binary.Write(f, binary.LittleEndian, uint32(0))
	binary.Write(f, binary.LittleEndian, uint32(0))
	f.Write(dataHeader)

	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		metadata := make([]byte, 0x118)
		setCString(metadata, entry.name)
		setUint32(metadata[0x100:], entry.unk00)
		setUint32(metadata[0x104:], entry.crc32_1)
		setUint32(metadata[0x108:], entry.unk08)
		setUint32(metadata[0x10C:], entry.length)
		setUint32(metadata[0x110:], uint32(len(entry.data)))
		setUint32(metadata[0x114:], entry.crc32_2)
		f.Write(metadata)
		f.Write(entry.data)
	}

	return nil
}

func unpack(inFileName string, dstPath string) error {
	data, err := os.ReadFile(inFileName)
	if err != nil {
		return err
	}

	if data[0] != 'R' || data[1] != 'E' || data[2] != 'S' || data[3] != 'P' ||
		data[4] != 'A' || data[5] != 'C' || data[6] != 'K' || data[7] != 0x1A {
		return ErrInvalidRespack
	}

	if _, err := os.Stat(dstPath); !os.IsExist(err) {
		os.MkdirAll(dstPath, 0755)
	}

	for _, e := range parse(data) {
		extractFileEntry(data, e, dstPath)
	}

	return nil
}

func decryptHeader(data []byte, n uint32) {
	n *= nEntrySize
	k := byte(0x5C)
	for i := uint32(0); i < n; i++ {
		data[i] ^= k
		k -= 0x3F
	}
}

func extractFileEntry(data []byte, e entry, dstPath string) error {
	outStream, err := os.Create(path.Join(dstPath, e.name))
	if err != nil {
		return err
	}
	defer outStream.Close()

	_, err = inflate(bytes.NewReader(e.data), outStream)
	if err != nil {
		return err
	}

	return err
}

func parse(data []byte) []entry {
	entries := make([]entry, 0)

	cursor := 0x18
	entryCount := binary.LittleEndian.Uint32(data[12:])
	decryptHeader(data[cursor:], entryCount)

	offData := cursor + int(entryCount)*nEntrySize
	for i := uint32(0); i < entryCount; i++ {
		offset := int(binary.LittleEndian.Uint32(data[cursor+0x128:])) + offData + 0x118
		cmpLen := binary.LittleEndian.Uint32(data[cursor+0x110:])
		entries = append(entries, entry{
			name:    getCString(data[cursor : cursor+0x100]),
			unk00:   binary.LittleEndian.Uint32(data[cursor+0x100:]),
			crc32_1: binary.LittleEndian.Uint32(data[cursor+0x104:]),
			unk08:   binary.LittleEndian.Uint32(data[cursor+0x108:]),
			length:  binary.LittleEndian.Uint32(data[cursor+0x10C:]),
			crc32_2: binary.LittleEndian.Uint32(data[cursor+0x114:]),
			data:    data[offset : offset+int(cmpLen)],
		})
		cursor += nEntrySize
	}

	return entries
}

func inflate(b *bytes.Reader, w io.Writer) (int64, error) {
	z, err := zlib.NewReader(b)
	if err != nil {
		return 0, err
	}
	defer z.Close()
	return io.Copy(w, z)
}

func deflate(r io.Reader, w io.Writer) (int64, error) {
	z, err := zlib.NewWriterLevel(w, zlib.DefaultCompression)
	if err != nil {
		return 0, err
	}
	defer z.Close()

	return io.Copy(z, r)
}

func getCString(data []byte) string {
	str := ""
	for i := 0; i < len(data) && data[i] != 0; i++ {
		str += string(rune(data[i]))
	}

	return str
}

func setCString(data []byte, s string) {
	len := len(s)
	for i := 0; i < len; i++ {
		data[i] = byte(s[i])
	}
	data[len] = 0
}

func setUint32(data []byte, n uint32) {
	data[0] = byte(n & 0xff)
	data[1] = byte((n >> 8) & 0xff)
	data[2] = byte((n >> 16) & 0xff)
	data[3] = byte((n >> 24) & 0xff)
}
