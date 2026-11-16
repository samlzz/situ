// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"fmt"
	"strings"
)

// checkComponents returns a message when the slash-separated components of
// rel (a path without its leading slash) are not all plain names.
func checkComponents(what, full, rel string) string {
	if strings.HasSuffix(rel, "/") {
		return fmt.Sprintf("%s %q must not end with \"/\"", what, full)
	}
	for c := range strings.SplitSeq(rel, "/") {
		switch c {
		case "":
			return fmt.Sprintf("%s %q must not contain empty components", what, full)
		case ".", "..":
			return fmt.Sprintf("%s %q must not contain %q components", what, full, c)
		}
	}
	return ""
}

func checkSourcePath(s string) string {
	if strings.HasPrefix(s, "/") {
		return fmt.Sprintf("source %q must be relative", s)
	}
	return checkComponents("source", s, s)
}

func checkDest(s string) string {
	if !strings.HasPrefix(s, "/") {
		return fmt.Sprintf("destination %q must be absolute", s)
	}
	if s == "/" {
		return "destination must not be \"/\""
	}
	return checkComponents("destination", s, s[1:])
}
