package stx

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"math"
	"os"
)

func WriteStx(w *os.File, step *Step) error {
	w.Write([]byte{'S', 'T', 'F', '4'})
	w.Seek(0x3C, io.SeekStart)
	if err := writeString(w, step.Title, 0x3F); err != nil {
		return err
	}
	w.Seek(0x7C, io.SeekStart)
	if err := writeString(w, step.Artist, 0x3F); err != nil {
		return err
	}
	w.Seek(0xBC, io.SeekStart)
	if err := writeString(w, step.Author, 0x3F); err != nil {
		return err
	}

	w.Seek(0xFC+9*4, io.SeekStart)

	buf := make([]byte, 4)
	w.Seek(0x120, io.SeekStart)
	offsets := make([]uint32, len(step.Charts))
	for i := 0; i < len(step.Charts); i++ {
		offset, _ := w.Seek(0, io.SeekCurrent)
		offsets[i] = uint32(offset)

		chart := step.Charts[i]
		setUint32(buf, chart.Difficulty)
		w.Write(buf[:4])
		setUint32(buf, compressedFlag)
		w.Write(buf[:4])
		w.Seek(0xC4, io.SeekCurrent)
		for _, block := range chart.Blocks {
			writeBlock(w, &block)
		}
	}

	w.Seek(0xFC, io.SeekStart)
	for i := 0; i < len(offsets); i++ {
		setUint32(buf, offsets[i])
		w.Write(buf[:4])
	}

	return nil
}

func writeBlock(w io.WriteSeeker, block *Block) {
	data := make([]byte, 0x84+len(block.Notes))
	setUint32(data[0x00:], math.Float32bits(block.Bpm))
	setUint32(data[0x04:], uint32(block.BeatPerMeasure))
	setUint32(data[0x08:], uint32(block.BeatSplit))
	setUint32(data[0x0C:], uint32(block.Delay))
	setDivisionSet(data[0x10:], block.SetPerfect)
	setDivisionSet(data[0x18:], block.SetGreat)
	setDivisionSet(data[0x20:], block.SetGood)
	setDivisionSet(data[0x28:], block.SetBad)
	setDivisionSet(data[0x30:], block.SetMiss)
	setDivisionSet(data[0x38:], block.SetStepG)
	setDivisionSet(data[0x40:], block.SetStepW)
	setDivisionSet(data[0x48:], block.SetStepA)
	setDivisionSet(data[0x50:], block.SetStepB)
	setDivisionSet(data[0x58:], block.SetStepC)
	setUint32(data[0x60:], uint32(block.Speed))
	setUint32(data[0x80:], uint32(len(block.Notes)/NotesPerRow))
	copy(data[0x84:], block.Notes)

	var cmpBuf bytes.Buffer
	deflate(bytes.NewBuffer(data), &cmpBuf)
	finalData := cmpBuf.Bytes()

	buf := make([]byte, 4)
	setUint32(buf, uint32(len(finalData)))
	w.Write(buf[:4])
	w.Write(finalData)
}

func writeString(w io.Writer, s string, maxLen int) error {
	data := []byte(s)
	if len(data) > maxLen {
		return fmt.Errorf("string '%s' cannot be longer than %d characters", s, maxLen)
	}

	_, err := w.Write(data)
	return err
}

func setUint32(data []byte, n uint32) {
	data[0] = byte(n & 0xff)
	data[1] = byte((n >> 8) & 0xff)
	data[2] = byte((n >> 16) & 0xff)
	data[3] = byte((n >> 24) & 0xff)
}

func setDivisionSet(data []byte, set DivisionSet) {
	setUint32(data, set.First)
	setUint32(data[8:], set.Second)
}

func deflate(r io.Reader, w io.Writer) (int64, error) {
	z, err := zlib.NewWriterLevel(w, zlib.DefaultCompression)
	if err != nil {
		return 0, err
	}
	defer z.Close()

	return io.Copy(z, r)
}
