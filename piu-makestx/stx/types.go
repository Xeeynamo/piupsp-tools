package stx

type Step struct {
	Title  string
	Artist string
	Author string
	Charts []Chart
}

type Chart struct {
	Difficulty uint32
	Blocks     []Block
}

type DivisionSet struct {
	First  uint32
	Second uint32
}

type Block struct {
	Bpm            float32
	BeatPerMeasure uint32
	BeatSplit      uint32
	Delay          uint32
	SetPerfect     DivisionSet
	SetGreat       DivisionSet
	SetGood        DivisionSet
	SetBad         DivisionSet
	SetMiss        DivisionSet
	SetStepG       DivisionSet
	SetStepW       DivisionSet
	SetStepA       DivisionSet
	SetStepB       DivisionSet
	SetStepC       DivisionSet
	Speed          uint32
	Notes          []byte
}

const (
	NoteEmpty = iota
	NoteTap
	NoteG
	NoteW
	NoteA
	NoteB
	NoteC
	NoteUnk7
	NoteUnk8
	NoteUnk9
	NoteHoldStart
	NoteHold
	NoteHoldEnd
)

const nChartMax = 9
const NotesPerRow = 13
const compressedFlag = 1
