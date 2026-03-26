package parser

// Loc represents a location/range in source code with line and column information
type Loc struct {
	fromLine int
	toLine   int
	fromCol  int
	toCol    int
}

// FromLine returns the starting line number
func (l Loc) FromLine() int { return l.fromLine }

// ToLine returns the ending line number
func (l Loc) ToLine() int { return l.toLine }

// FromCol returns the starting column number
func (l Loc) FromCol() int { return l.fromCol }

// ToCol returns the ending column number
func (l Loc) ToCol() int { return l.toCol }

// IsInside checks if this location is completely inside the other location
func (l Loc) IsInside(other Loc) bool {
	// Check if l starts after or at the start of other
	startsInside := l.fromLine > other.fromLine ||
		(l.fromLine == other.fromLine && l.fromCol >= other.fromCol)

	// Check if l ends before or at the end of other
	endsInside := l.toLine < other.toLine ||
		(l.toLine == other.toLine && l.toCol <= other.toCol)

	return startsInside && endsInside
}

// Envelopes checks if this location completely envelopes (contains) the other location
func (l Loc) Envelopes(other Loc) bool {
	// Check if l starts before or at the start of other
	startsBefore := l.fromLine < other.fromLine ||
		(l.fromLine == other.fromLine && l.fromCol <= other.fromCol)

	// Check if l ends after or at the end of other
	endsAfter := l.toLine > other.toLine ||
		(l.toLine == other.toLine && l.toCol >= other.toCol)

	return startsBefore && endsAfter
}

// Equal checks if two locations are exactly the same
func (l Loc) Equal(other Loc) bool {
	return l.fromLine == other.fromLine &&
		l.toLine == other.toLine &&
		l.fromCol == other.fromCol &&
		l.toCol == other.toCol
}

func mergeLoc(l1 Loc, l2 Loc) Loc {
	return Loc{
		fromLine: l1.fromLine,
		toLine:   l1.toLine,
		fromCol:  l2.fromCol,
		toCol:    l2.toCol,
	}
}

var noloc = Loc{
	0, 0, 0, 0,
}

// NewLoc creates a new location with the given line and column information
func NewLoc(fromLine, toLine, fromCol, toCol int) Loc {
	return Loc{
		fromLine: fromLine,
		toLine:   toLine,
		fromCol:  fromCol,
		toCol:    toCol,
	}
}
