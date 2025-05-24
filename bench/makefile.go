package main

import (
	"fmt"
	"strings"
)

func makefile(loc int, numberOfErrors int) string {
	errorLines := makeErrorLines(numberOfErrors)
	if loc-numberOfErrors > 0 {
		correctLines := makeCorrectLines(loc - numberOfErrors)
		return strings.Join([]string{errorLines, correctLines}, "\n")
	} else {
		return errorLines
	}

}
func makeMultiParty(loc int, numberOfLocs int) string {
	errorLines := makeMultiPartyError(numberOfLocs / 2)
	if loc-numberOfLocs > 0 {
		correctLines := makeCorrectLines(loc - numberOfLocs)
		return strings.Join([]string{errorLines, correctLines}, "\n")
	} else {
		return errorLines
	}
}
func makeMultiPartyError(parties int) string {
	lines1 := make([]string, parties)
	types := make([]string, parties)
	for i := range parties {
		types[i] = fmt.Sprintf("data X%d = X%d", i, i)
	}
	for i := range parties {
		lines1[i] = fmt.Sprintf("x = X%d", i)
	}

	return strings.Join(types, "\n") + "\n" + strings.Join(lines1, "\n")
}

func makeMultiWitness(loc int, numberOfLocs int) string {
	errorLines := makeMultiWitnessError(numberOfLocs)
	if loc-numberOfLocs > 0 {
		correctLines := makeCorrectLines(loc - numberOfLocs)
		return strings.Join([]string{errorLines, correctLines}, "\n")
	} else {
		return errorLines
	}
}

func makeMultiWitnessError(numberOfLocs int) string {
	header := "x0 :: Char\n"
	lines := make([]string, numberOfLocs-1)
	for i := range numberOfLocs - 1 {
		lines[i] = "x0 = 0"
	}
	return header + strings.Join(lines, "\n")
}

func makeErrorLines(errorNumber int) string {
	header := "x0 :: Char\n"
	lines := make([]string, errorNumber-1)
	for i := range errorNumber - 2 {
		lines[i] = fmt.Sprintf("x%d = x%d", i, i+1)
	}
	lines[errorNumber-2] = fmt.Sprintf("x%d = 0", errorNumber-2)
	return header + strings.Join(lines, "\n")
}

func makeCorrectLines(loc int) string {
	lines := make([]string, loc)
	for i := range loc {
		if i%2 == 0 {
			lines[i] = fmt.Sprintf("--y%d = 0", i)
		} else {
			lines[i] = fmt.Sprintf("y%d = 0", i)

		}
	}
	return strings.Join(lines, "\n")
}
