// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"fmt"
	"strings"
)

// Expand returns the entries with variables replaced in Dest.
// Rejects two entries with the same expanded destination.
func Expand(sf *Situfile, vars map[string]string) ([]Entry, error) {
	var errs Errors
	addErr := func(line int, format string, a ...any) {
		errs = append(errs, LineError{Path: sf.Path, Line: line, Msg: fmt.Sprintf(format, a...)})
	}
	out := make([]Entry, 0, len(sf.Entries))
	seen := map[string]int{}

	for _, e := range sf.Entries {
		var b strings.Builder
		orig := e.Dest
		rest := e.Dest
		msg := checkVars(e.Dest, func(name string) bool { _, ok := vars[name]; return ok })
		if msg != "" {
			addErr(e.Line, "%s", msg)
			continue
		}
		for {
			i := strings.Index(rest, "${")
			if i < 0 {
				b.WriteString(rest)
				break
			}
			end := strings.IndexByte(rest[i:], '}')
			b.WriteString(rest[:i])
			b.WriteString(vars[rest[i+2:i+end]])
			rest = rest[i+end+1:]
		}
		e.Dest = b.String()
		if msg := checkDest(e.Dest); msg != "" {
			addErr(e.Line, "%s (expanded from %q)", msg, orig)
			continue
		}
		if first, dup := seen[e.Dest]; dup {
			addErr(e.Line, "duplicate destination %q after expansion (first used at line %d)", e.Dest, first)
			continue
		}
		seen[e.Dest] = e.Line
		out = append(out, e)
	}
	if len(errs) > 0 {
		return nil, errs
	}
	return out, nil
}
