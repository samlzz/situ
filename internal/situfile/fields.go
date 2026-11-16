// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"
)

var unitSuffixes = []string{
	".service", ".socket", ".timer", ".target", ".path",
	".mount", ".automount", ".swap", ".slice",
}

// parseMode parses exactly four octal digits, setuid, setgid and sticky included.
func parseMode(s string) (fs.FileMode, bool) {
	if len(s) != 4 {
		return 0, false
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '7' {
			return 0, false
		}
	}
	m, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return 0, false
	}
	mode := fs.FileMode(m & 0o777)
	if m&0o4000 != 0 {
		mode |= fs.ModeSetuid
	}
	if m&0o2000 != 0 {
		mode |= fs.ModeSetgid
	}
	if m&0o1000 != 0 {
		mode |= fs.ModeSticky
	}
	return mode, true
}

// parseOwner parses <user>:<group>.
func parseOwner(s string) (Owner, bool) {
	user, group, found := strings.Cut(s, ":")
	if !found || !validIDPart(user) || !validIDPart(group) {
		return Owner{}, false
	}
	return Owner{User: user, Group: group}, true
}

// parseTarget checks the syntax of a requires target only.
func parseTarget(s string) (ModuleID, bool) {
	parts := strings.Split(s, "/")
	for _, p := range parts {
		if !validName(p) {
			return ModuleID{}, false
		}
	}
	switch len(parts) {
	case 1:
		return ModuleID{Component: parts[0]}, true
	case 2:
		return ModuleID{Category: parts[0], Component: parts[1]}, true
	}
	return ModuleID{}, false
}

// checkSourceField checks the source field of a typ entry and returns a
// message, or "" when it is valid.
func checkSourceField(typ EntryType, s string) string {
	switch typ {
	case Dir, Data:
		if s != "-" {
			return fmt.Sprintf("%s: source must be \"-\", got %q", typ, s)
		}
		return ""
	}
	if s == "-" {
		return fmt.Sprintf("%s: source is required (\"-\" means no source)", typ)
	}
	return checkSourcePath(s)
}

// validName reports whether s matches [a-z0-9][a-z0-9-]*, the name of a
// category or component.
func validName(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || i > 0 && c == '-') {
			return false
		}
	}
	return true
}

func isDigits(s string) bool {
	return s != "" && !strings.ContainsFunc(s, func(r rune) bool { return r < '0' || r > '9' })
}

// validIDPart reports whether s is a user or group name or a decimal ID
// that fits in 32 bits.
func validIDPart(s string) bool {
	if isDigits(s) {
		_, err := strconv.ParseUint(s, 10, 32)
		return err == nil
	}
	name := strings.TrimSuffix(s, "$")
	if name == "" {
		return false
	}
	for i := range len(name) {
		c := name[i]
		first := c >= 'a' && c <= 'z' || c == '_'
		if !(first || i > 0 && (c >= '0' && c <= '9' || c == '-')) {
			return false
		}
	}
	return true
}

func validUnit(s string) bool {
	for i := range len(s) {
		c := s[i]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' ||
			strings.IndexByte(":_.@\\-", c) >= 0) {
			return false
		}
	}
	return slices.ContainsFunc(unitSuffixes, func(suf string) bool {
		return len(s) > len(suf) && strings.HasSuffix(s, suf)
	})
}
