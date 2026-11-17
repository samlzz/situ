// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/samlzz/situ/internal/exitcode"
)

// LineError is one line of Errors. It is not an error on its own.
type LineError struct {
	Path string
	Line int
	Msg  string
}

func (e LineError) String() string {
	return fmt.Sprintf("%s:%d: %s", e.Path, e.Line, e.Msg)
}

// Errors is every error found in one Situfile, sorted by line.
type Errors []LineError

func (e Errors) Error() string {
	lines := make([]string, len(e))
	for i, le := range e {
		lines[i] = le.String()
	}
	return strings.Join(lines, "\n")
}

func (Errors) ExitCode() int { return exitcode.Usage }

func (e *Errors) add(path string, line int, format string, a ...any) {
	*e = append(*e, LineError{Path: path, Line: line, Msg: fmt.Sprintf(format, a...)})
}

// Add appends an error; it is for callers that check several Situfiles.
func (e *Errors) Add(path string, line int, format string, a ...any) { e.add(path, line, format, a...) }

func (e Errors) sortByLine() {
	slices.SortStableFunc(e, func(a, b LineError) int { return cmp.Compare(a.Line, b.Line) })
}

// Sort orders the errors of several Situfiles by path, then line.
func (e Errors) Sort() {
	slices.SortStableFunc(e, func(a, b LineError) int {
		return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.Line, b.Line))
	})
}
