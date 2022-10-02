package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/xeeynamo/piupsp-tools/piu-makestx/stx"
)

type desc struct {
	bpm            float32
	beatPerMeasure uint32
	beatSplit      uint32
}

var errTimeSignature = errors.New("invalid time signature")
var errDisplayBPM = errors.New("invalid display BPM")

func parseAsStx(fileName string) (stx.Step, error) {
	step := stx.Step{}
	f, err := os.Open(fileName)
	if err != nil {
		return step, err
	}
	defer f.Close()

	desc := desc{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case len(line) >= 1 && line[0] == '#':
			key, value := parseToken(line[1:])
			switch key {
			case "TITLE":
				step.Title = value
			case "ARTIST":
				step.Artist = value
			case "CREDIT":
				step.Author = value
			case "DISPLAYBPM":
				v, err := strconv.ParseFloat(value, 32)
				if err != nil || v <= 0 {
					return step, fmt.Errorf("%s: %s", errDisplayBPM.Error(), line)
				}
				desc.bpm = float32(v)
			case "TIMESIGNATURES":
				beatPerMeasure, beatSplit, err := parseTimeSignature(value)
				if err != nil {
					return step, fmt.Errorf("%s: %s", err.Error(), line)
				}
				desc.beatPerMeasure = beatPerMeasure
				desc.beatSplit = beatSplit
			case "NOTEDATA":
				chart, err := parseChart(scanner, desc)
				step.Charts = append(step.Charts, chart)
				if err != nil {
					return step, err
				}
			}
		}
	}

	return step, nil
}

func parseChart(scanner *bufio.Scanner, desc desc) (stx.Chart, error) {
	chart := stx.Chart{}
	block := stx.Block{
		Bpm:            desc.bpm / 2,
		BeatPerMeasure: desc.beatPerMeasure,
		BeatSplit:      desc.beatSplit,
		Speed:          1000,
	}
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case len(line) >= 1 && line[0] == '#':
			key, value := parseToken(line[1:])
			switch key {
			case "METER":
				difficulty, err := strconv.Atoi(value)
				if err != nil {
					return chart, err
				}
				chart.Difficulty = uint32(difficulty)
			case "TIMESIGNATURES":
				beatPerMeasure, beatSplit, err := parseTimeSignature(value)
				if err != nil {
					return chart, fmt.Errorf("%s: %s", err.Error(), line)
				}
				block.BeatPerMeasure = beatPerMeasure
				block.BeatSplit = beatSplit
			case "NOTES":
				block.Notes = readNotes(scanner)
				chart.Blocks = append(chart.Blocks, block)
				return chart, nil // natural end of chart
			}
		}
	}

	chart.Blocks = append(chart.Blocks, block)
	return chart, nil
}

func readNotes(scanner *bufio.Scanner) []byte {
	notes := make([]byte, 0)
	row := make([]byte, stx.NotesPerRow)
	hold := make([]bool, stx.NotesPerRow)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case len(line) == 0: // ignore
		case line[0] == ',': // end of block
		case line[0] == ';': // ends of chart
			return notes
		}

		for i := 0; i < len(line) && i < stx.NotesPerRow; i++ {
			switch line[i] {
			case '0':
				if hold[i] {
					row[i] = stx.NoteHold
				} else {
					row[i] = stx.NoteEmpty
				}
			case '1':
				row[i] = stx.NoteTap
			case '2':
				row[i] = stx.NoteHoldStart
				hold[i] = true
			case '3':
				row[i] = stx.NoteHoldEnd
				hold[i] = false
			}
		}

		notes = append(notes, row...)
	}

	return notes
}

func parseToken(s string) (string, string) {
	tokens := strings.Split(s, ":")
	switch len(tokens) {
	case 0:
		return "", ""
	case 1:
		return tokens[0], ""
	default:
		values := strings.Split(tokens[1], ";")
		if len(values) > 0 {
			return tokens[0], values[0]
		} else {
			return tokens[0], ""
		}
	}
}

func parseTimeSignature(s string) (uint32, uint32, error) {
	tokens := strings.Split(s, "=")
	if len(tokens) != 3 {
		return 0, 0, errTimeSignature
	}

	n, err := strconv.Atoi(tokens[1])
	if err != nil || n <= 0 {
		return 0, 0, fmt.Errorf("%s: %s", errTimeSignature.Error(), err.Error())
	}
	beatPerMeasure := uint32(n)

	n, err = strconv.Atoi(tokens[2])
	if err != nil || n <= 0 {
		return 0, 0, fmt.Errorf("%s: %s", errTimeSignature.Error(), err.Error())
	}
	beatSplit := uint32(n)

	return beatPerMeasure, beatSplit, nil
}
