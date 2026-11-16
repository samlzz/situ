// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import "fmt"

// Expand returns the entries with variables replaced in Dest.
// Rejects two entries with the same expanded destination.
func Expand(sf *Situfile, vars map[string]string) ([]Entry, error) {
	var errs Errors
	out := make([]Entry, 0, len(sf.Entries))
	seen := map[string]int{}

	for _, e := range sf.Entries {
		orig := e.Dest
		dest, msg := substVars(orig, vars)
		if msg != "" {
			errs.add(sf.Path, e.Line, "%s", msg)
			continue
		}
		e.Dest = dest
		note := ""
		if dest != orig {
			note = fmt.Sprintf(" (expanded from %q)", orig)
		}
		if msg := checkDest(dest); msg != "" {
			errs.add(sf.Path, e.Line, "%s%s", msg, note)
			continue
		}
		if first, dup := seen[dest]; dup {
			errs.add(sf.Path, e.Line, "duplicate destination %q%s, first used at line %d", dest, note, first)
			continue
		}
		seen[dest] = e.Line
		out = append(out, e)
	}
	if len(errs) > 0 {
		return nil, errs
	}
	return out, nil
}
