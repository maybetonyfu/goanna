package haskell

import (
	"fmt"
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
	LocalType  map[int]string
	GlobalType map[string]string
	MCS        []int
	Snapshot   []Line
}

type NodeDetail struct {
	DisplayName string
	Range       inventory.Range
}

type TypeError struct {
	CriticalNodes map[int]NodeDetail
	Fixes         []Fix
}

type Report struct {
	TypeErrors []TypeError
	NodeRange  map[int]inventory.Range
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

func getDisplayName(loc inventory.Range, file string) string {
	lines := strings.Split(file, "\n")
	if loc.FromLine != loc.ToLine {
		fromLine := lines[loc.FromLine]
		toLine := lines[loc.ToLine]
		start := fromLine[loc.FromCol : loc.FromCol+4]
		var end string
		if loc.ToCol < 4 {
			end = toLine[0:loc.ToCol]

		} else {
			end = toLine[loc.ToCol-4 : loc.ToCol]
		}

		return strings.Join([]string{start, end}, "...")

	} else {
		line := lines[loc.FromLine]
		if loc.ToCol-loc.FromCol > 8 {
			start := line[loc.FromCol : loc.FromCol+4]
			end := line[loc.ToCol-4 : loc.ToCol]
			return strings.Join([]string{start, end}, "...")
		}
		return line[loc.FromCol:loc.ToCol]

	}

}

func ReportTypeError(rawError marco.Error, inv inventory.Inventory, file string) TypeError {
	fixes := make([]Fix, len(rawError.Causes))
	for i, cause := range rawError.Causes {
		prologResult := inv.QueryTypes(cause.MSS.ToSlice(), rawError.CriticalNodes)
		globals := prologResult["G"]
		locals := prologResult["L"]
		fmt.Printf("Local Types: %v\nMCS: %v\n", locals, cause.MCS)

		globalTypes, err := prolog_tool.ParseTerm(globals)
		localTypes, err := prolog_tool.ParseTerm(locals)

		globalTypeMapping := make(map[string]string)
		for i, v := range globalTypes.(prolog_tool.List).Values {
			printer := NewPrinter()
			decl := inv.Declarations[i]
			globalTypeMapping[decl] = printer.GetType(v)
		}

		localTypeMapping := make(map[int]string)
		for i, v := range localTypes.(prolog_tool.List).Values {
			printer := NewPrinter()
			nodeId := rawError.CriticalNodes[i]
			localTypeMapping[nodeId] = printer.GetType(v)
		}

		if err != nil {
			panic("Error in parse types")
		}
		lines := createSnapshot(rawError.CriticalNodes, cause.MCS.ToSlice(), inv.NodeRange, file)
		fixes[i] = Fix{
			LocalType:  localTypeMapping,
			GlobalType: globalTypeMapping,
			Snapshot:   lines,
			MCS:        cause.MCS.ToSlice(),
		}
	}
	slices.SortFunc(fixes, func(a, b Fix) int {
		minMCS1 := slices.Min(a.MCS)
		minMCS2 := slices.Min(b.MCS)
		loc1 := inv.NodeRange[minMCS1]
		loc2 := inv.NodeRange[minMCS2]
		if loc1.FromLine < loc2.FromLine {
			return -1
		} else if loc1.FromLine > loc2.FromLine {
			return 1
		} else {
			return loc1.FromCol - loc2.FromCol
		}
	})
	nodeDetails := make(map[int]NodeDetail)
	for _, node := range rawError.CriticalNodes {
		nodeDetails[node] = NodeDetail{
			DisplayName: getDisplayName(inv.NodeRange[node], file),
			Range:       inv.NodeRange[node],
		}
	}
	return TypeError{
		Fixes:         fixes,
		CriticalNodes: nodeDetails,
	}
}

func MakeReport(errors []marco.Error, inv inventory.Inventory, file string) Report {
	tcErrors := make([]TypeError, len(errors))
	for i, e := range errors {
		tcErrors[i] = ReportTypeError(e, inv, file)
	}
	slices.SortFunc(tcErrors, func(a, b TypeError) int {
		var minA, minB int = -1, -1
		for k := range a.CriticalNodes {
			if minA == -1 || k <= minA {
				minA = k
			}
		}

		for k := range b.CriticalNodes {
			if minB == -1 || k <= minB {
				minB = k
			}
		}
		locA := inv.NodeRange[minA]
		locB := inv.NodeRange[minB]
		if locA.FromLine < locB.FromLine {
			return -1
		} else if locA.FromLine > locB.FromLine {
			return 1
		} else {
			return locA.FromCol - locB.FromCol
		}

	})
	return Report{
		TypeErrors: tcErrors,
		NodeRange:  inv.NodeRange,
	}
}
