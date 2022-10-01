package stx

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"math"
)

func ParseStx(data []byte) (Step, error) {
	if data[0] != 'S' || data[1] != 'T' || data[2] != 'F' || data[3] != '4' {
		return Step{}, errors.New("not a STX/STF4 file")
	}

	step := Step{
		Title:  getCString(data[0x3C:]),
		Artist: getCString(data[0x7C:]),
		Author: getCString(data[0xBC:]),
		Charts: make([]Chart, nChartMax),
	}

	offsets := readUint32Array(data[0xFC:], nChartMax)
	for i := 0; i < nChartMax; i++ {
		difficulty := binary.LittleEndian.Uint32(data[offsets[i]:])
		offset := int(offsets[i] + 0xCC)
		blocks := make([]Block, 0)
		for offset < len(data) {
			size := int(binary.LittleEndian.Uint32(data[offset:]))
			offset += 4
			if size == 0 {
				break
			}

			var buffer bytes.Buffer
			if _, err := inflate(bytes.NewReader(data[offset:offset+size]), &buffer); err != nil {
				return Step{}, err
			}

			blocks = append(blocks, parseBlock(buffer.Bytes()))
			offset += size
		}

		step.Charts[i] = Chart{
			Difficulty: difficulty,
			Blocks:     blocks,
		}
	}

	return step, nil
}

func parseBlock(data []byte) Block {
	nRow := binary.LittleEndian.Uint32(data[0x80:])

	return Block{
		Bpm:            math.Float32frombits(binary.LittleEndian.Uint32(data)),
		BeatPerMeasure: binary.LittleEndian.Uint32(data[0x04:]),
		BeatSplit:      binary.LittleEndian.Uint32(data[0x08:]),
		Delay:          binary.LittleEndian.Uint32(data[0x0C:]),
		SetPerfect:     parseDivisionSet(data[0x10:]),
		SetGreat:       parseDivisionSet(data[0x18:]),
		SetGood:        parseDivisionSet(data[0x20:]),
		SetBad:         parseDivisionSet(data[0x28:]),
		SetMiss:        parseDivisionSet(data[0x30:]),
		SetStepG:       parseDivisionSet(data[0x38:]),
		SetStepW:       parseDivisionSet(data[0x40:]),
		SetStepA:       parseDivisionSet(data[0x48:]),
		SetStepB:       parseDivisionSet(data[0x50:]),
		SetStepC:       parseDivisionSet(data[0x58:]),
		Speed:          binary.LittleEndian.Uint32(data[0x60:]),
		Notes:          data[0x84 : 0x84+nRow*NotesPerRow],
	}
}

func parseDivisionSet(data []byte) DivisionSet {
	return DivisionSet{
		First:  binary.LittleEndian.Uint32(data[0:]),
		Second: binary.LittleEndian.Uint32(data[4:]),
	}
}

func inflate(b *bytes.Reader, w io.Writer) (int64, error) {
	z, err := zlib.NewReader(b)
	if err != nil {
		return 0, err
	}
	defer z.Close()
	return io.Copy(w, z)
}

func getCString(data []byte) string {
	str := ""
	for i := 0; i < len(data) && data[i] != 0; i++ {
		str += string(rune(data[i]))
	}

	return str
}

func readUint32Array(data []byte, n int) []uint32 {
	a := make([]uint32, n)
	for i := 0; i < n; i++ {
		a[i] = binary.LittleEndian.Uint32(data[i*4:])
	}

	return a
}
