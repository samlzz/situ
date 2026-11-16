// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import "strings"

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
		if msg := checkDest(dest); msg != "" {
			errs.add(sf.Path, e.Line, "%s (expanded from %q)", msg, orig)
			continue
		}
		if first, dup := seen[dest]; dup {
			if strings.ContainsRune(orig, '$') {
				errs.add(sf.Path, e.Line, "duplicate destination %q (first used at line %d) (expanded from %q)", dest, first, orig)
			} else {
				errs.add(sf.Path, e.Line, "duplicate destination %q (first used at line %d)", dest, first)
			}
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
