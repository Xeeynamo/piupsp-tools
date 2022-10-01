package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/xeeynamo/piupsp-tools/piu-makestx/stx"
)

func main() {
	var inFileName string
	var outFileName string
	var difficulty string

	args := os.Args[1:]
	switch len(args) {
	case 3:
		inFileName = args[0]
		outFileName = args[1]
		difficulty = args[2]
	default:
		printHelp()
		os.Exit(1)
	}

	err := convertSccToStx(inFileName, outFileName, difficulty)
	if err != nil {
		panic(err)
	}

	os.Exit(0)
}

func printHelp() {
	fmt.Fprint(os.Stderr, "usage:\n")
	fmt.Fprint(os.Stderr, "   piu-makestx step.scc step.STX S10\n")
}

func convertSccToStx(inFileName string, outFileName string, difficulty string) error {
	switch {
	case len(difficulty) > 0 && difficulty[0] == 'S':
	case len(difficulty) > 0 && difficulty[0] == 'D':
	default:
		return fmt.Errorf("unsure if it is a single or double chart: %s", difficulty)
	}

	level, err := strconv.Atoi(difficulty[1:])
	if err != nil {
		return fmt.Errorf("difficulty format invalid: %s", difficulty)
	}

	step, err := parseAsStx(inFileName)
	if err != nil {
		return err
	}

	var mainChart *stx.Chart = nil
	for _, b := range step.Charts {
		if b.Difficulty == uint32(level) {
			mainChart = &b
			break
		}
	}

	if mainChart == nil {
		return fmt.Errorf("no %s chart found in the specified SCC", difficulty)
	}

	f, err := os.Create(outFileName)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	return stx.WriteStx(f, &stx.Step{
		Title:  step.Title,
		Artist: step.Artist,
		Author: step.Author,
		Charts: []stx.Chart{
			// wow, this is UGLY.
			*mainChart,
			*mainChart,
			*mainChart,
			*mainChart,
			*mainChart,
			*mainChart,
			*mainChart,
			*mainChart,
			*mainChart,
		},
	})
}
