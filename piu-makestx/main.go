package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xeeynamo/piupsp-tools/piu-makestx/stx"
)

func main() {
	var inFileName string
	var outFileName string

	args := os.Args[1:]
	switch len(args) {
	case 2:
		inFileName = args[0]
		outFileName = args[1]
	case 1:
		inFileName = args[0]
		outFileName = strings.TrimSuffix(filepath.Base(inFileName), filepath.Ext(inFileName)) + ".STX"
	default:
		printHelp()
		os.Exit(1)
	}

	err := convertSccToStx(inFileName, outFileName)
	if err != nil {
		panic(err)
	}

	os.Exit(0)
}

func printHelp() {
	fmt.Fprint(os.Stderr, "usage:\n")
	fmt.Fprint(os.Stderr, "   piu-makestx step.scc [step.STX]\n")
}

func convertSccToStx(inFileName string, outFileName string) error {
	step, err := parseAsStx(inFileName)
	if err != nil {
		return err
	}

	f, err := os.Create(outFileName)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	chart := stx.Chart{
		Difficulty: 0,
		Blocks: []stx.Block{
			{
				Bpm:            step.Charts[0].Blocks[0].Bpm,
				BeatPerMeasure: step.Charts[0].Blocks[0].BeatPerMeasure,
				BeatSplit:      step.Charts[0].Blocks[0].BeatSplit,
				Speed:          step.Charts[0].Blocks[0].Speed,
				Notes:          make([]byte, 64*stx.NotesPerRow),
			},
		},
	}

	mainChart := &step.Charts[8]
	// mainBlock := &mainChart.Blocks[0]
	// delay := make([]byte, stx.NotesPerRow*mainBlock.BeatPerMeasure*mainBlock.BeatSplit*4)
	// mainBlock.Notes = append(mainBlock.Notes, delay...)
	// mainBlock.Notes = append(mainBlock.Notes, delay...)
	// mainBlock.Notes = append(mainBlock.Notes, delay...)
	// mainBlock.Notes = append(mainBlock.Notes, delay...)

	return stx.WriteStx(f, &stx.Step{
		Title:  step.Title,
		Artist: step.Artist,
		Author: step.Author,
		Charts: []stx.Chart{
			chart,
			*mainChart,
			chart,
			chart,
			chart,
			chart,
			chart,
			chart,
			chart,
		},
	})
}
