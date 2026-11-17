// SPDX-License-Identifier: GPL-3.0-or-later

package testutil

import "strings"

// Situfile returns the content of a Situfile made of the format header
// followed by lines, one statement per line.
func Situfile(lines ...string) string {
	if len(lines) == 0 {
		return "format 1\n"
	}
	return "format 1\n" + strings.Join(lines, "\n") + "\n"
}
