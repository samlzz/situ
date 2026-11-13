// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"fmt"
	"strings"

	"github.com/samlzz/situ/internal/exitcode"
)

type LineError struct {
	Path string
	Line int
	Msg  string
}

func (e *LineError) Error() string {
	return fmt.Sprintf("%s:%d: %s", e.Path, e.Line, e.Msg)
}

// Errors is every error found in one Situfile, sorted by line.
type Errors []LineError

func (e Errors) Error() string {
	lines := make([]string, len(e))
	for i, le := range e {
		lines[i] = le.Error()
	}
	return strings.Join(lines, "\n")
}

func (Errors) ExitCode() int { return exitcode.Usage }
