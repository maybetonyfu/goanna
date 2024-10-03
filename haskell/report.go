package haskell

import (
	mapset "github.com/deckarep/golang-set/v2"
	"mil/inventory"
	"mil/marco"
	prolog_tool "mil/prolog-tool"
	"slices"
	"strings"
)

type Tag string

const (
	tagNormal   Tag = "normal"
	tagCritical Tag = "critical"
	tagError    Tag = "error"
)

type Span struct {
	Tag  Tag
	Text string
	From int
	To   int
	Node int
	Type string
}

type Line struct {
	LineNumber int
	Spans      []Span
}

type Fix struct {
	CriticalNodes []int
	LocalType     map[int]string
	GlobalType    map[string]string
	Snapshot      []Line
}

type TypeError struct {
	CriticalNodes []int
	Fixes         []Fix
}

func shrinkRangeOnLine(loc inventory.Range, lineNum int, lineLength int) (int, int) {
	if loc.FromLine == lineNum && loc.ToLine == lineNum {
		return loc.FromCol, loc.ToCol
	}

	if loc.FromLine == lineNum {
		return loc.FromCol, lineLength
	}

	if loc.ToLine == lineNum {
		return 0, loc.ToCol
	}

	return 0, lineLength
}

func createSnapshot(criticalNodes []int, mcsNodes []int, nodeRange map[int]inventory.Range, file string) []Line {
	lineHasNode := make(map[int][]int)
	for _, node := range criticalNodes {
		loc := nodeRange[node]

		lineHasNode[node] = make([]int, 0)
		for i := loc.FromLine; i <= loc.ToLine; i++ {
			lineHasNode[i] = append(lineHasNode[i], node)
		}
	}

	lines := make([]Line, 0)
	for lineNum, line := range strings.Split(file, "\n") {

		if lineHasNode[lineNum] == nil || len(lineHasNode[lineNum]) == 0 {
			continue
		}

		var snapshotLine Line
		lineLength := len(line)
		snapshotLine = Line{
			LineNumber: lineNum,
			Spans:      nil,
		}
		criticalSpans := make([]Span, len(lineHasNode[lineNum]))
		for i, node := range lineHasNode[lineNum] {
			loc := nodeRange[node]
			fromPos, toPos := shrinkRangeOnLine(loc, lineNum, lineLength)
			var tag Tag
			if slices.Contains(mcsNodes, node) {
				tag = tagError
			} else {
				tag = tagCritical
			}
			criticalSpans[i] = Span{
				Tag:  tag,
				Text: line[fromPos:toPos],
				From: fromPos,
				To:   toPos,
				Node: node,
			}
		}
		slices.SortFunc(criticalSpans, func(a Span, b Span) int { return a.From - b.From })
		middleSpans := make([]Span, 0)
		for i := 0; i < len(criticalSpans)-1; i++ {
			a := criticalSpans[i]
			b := criticalSpans[i+1]
			if a.To < b.From {
				middleSpans = append(middleSpans, Span{
					Tag:  tagNormal,
					Text: line[a.To:b.From],
					From: a.To,
					To:   b.From,
				})
			}
		}
		if criticalSpans[0].From > 0 {
			middleSpans = append(middleSpans, Span{
				Tag:  tagNormal,
				Text: line[0:criticalSpans[0].From],
				From: 0,
				To:   criticalSpans[0].From,
			})
		}

		if criticalSpans[len(criticalSpans)-1].To < lineLength {
			middleSpans = append(middleSpans, Span{
				Tag:  tagNormal,
				Text: line[criticalSpans[len(criticalSpans)-1].To:lineLength],
				From: criticalSpans[len(criticalSpans)-1].To,
				To:   lineLength,
			})
		}
		spans := slices.Concat(middleSpans, criticalSpans)
		slices.SortFunc(spans, func(a Span, b Span) int { return a.From - b.From })
		snapshotLine.Spans = spans
		lines = append(lines, snapshotLine)

	}
	return lines
}

func ReportTypeError(rawError marco.Error, otherMSS mapset.Set[int], inv inventory.Inventory, file string) TypeError {
	fixes := make([]Fix, len(rawError.Causes))
	for i, cause := range rawError.Causes {
		printer := NewPrinter()
		completeMSS := otherMSS.Union(cause.MSS)
		prologResult := inv.QueryTypes(completeMSS.ToSlice(), rawError.CriticalNodes)
		globals := prologResult["G"]
		locals := prologResult["L"]

		globalTypes, err := prolog_tool.ParseTerm(globals)
		localTypes, err := prolog_tool.ParseTerm(locals)

		globalTypeMapping := make(map[string]string)
		for i, v := range globalTypes.(prolog_tool.List).Values {
			decl := inv.Declarations[i]
			globalTypeMapping[decl] = printer.GetType(v)
		}

		localTypeMapping := make(map[int]string)
		for i, v := range localTypes.(prolog_tool.List).Values {
			nodeId := rawError.CriticalNodes[i]
			localTypeMapping[nodeId] = printer.GetType(v)
		}

		if err != nil {
			panic("Error in parse types")
		}
		lines := createSnapshot(rawError.CriticalNodes, cause.MCS.ToSlice(), inv.NodeRange, file)
		fixes[i] = Fix{
			CriticalNodes: rawError.CriticalNodes,
			LocalType:     localTypeMapping,
			GlobalType:    globalTypeMapping,
			Snapshot:      lines,
		}
	}
	return TypeError{
		Fixes:         fixes,
		CriticalNodes: rawError.CriticalNodes,
	}
}
