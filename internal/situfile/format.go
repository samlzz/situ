// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"fmt"
	"io/fs"
	"strings"
)

// ValidName reports whether s matches [a-z0-9][a-z0-9-]*, the name of a
// category, a component or a host.
func ValidName(s string) bool { return validName(s) }

// modeString writes m as four octal digits, special bits included.
func modeString(m fs.FileMode) string {
	v := uint32(m.Perm())
	if m&fs.ModeSetuid != 0 {
		v |= 0o4000
	}
	if m&fs.ModeSetgid != 0 {
		v |= 0o2000
	}
	if m&fs.ModeSticky != 0 {
		v |= 0o1000
	}
	return fmt.Sprintf("%04o", v)
}

// Format returns the statement of e with its fields joined by two spaces.
// Dest is printed as it is in e, so variables stay unexpanded unless e
// comes from Expand.
func (e Entry) Format() string {
	src := e.Source
	if src == "" {
		src = "-"
	}
	return strings.Join([]string{
		e.Type.String(), modeString(e.Mode), e.Owner.User + ":" + e.Owner.Group, src, e.Dest,
	}, "  ")
}

// Format returns the statement of u with its fields joined by two spaces.
func (u Unit) Format() string { return u.Action.String() + "  " + u.Name }
