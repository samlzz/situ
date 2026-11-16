// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"fmt"
	"strings"
)

// VarHost is the name of the host profile directory.
const VarHost = "HOST"

// format1Vars lists the variables defined in format 1.
var format1Vars = []string{VarHost}

// scanVars walks the ${NAME} references of s. For each valid one it calls ref
// with the name and the byte range s[start:end] of it; a non-empty result stops
// the scan and is returned. A '$' that does not start a valid reference is an error.
func scanVars(s string, ref func(name string, start, end int) string) string {
	for i := 0; i < len(s); i++ {
		if s[i] != '$' {
			continue
		}
		bad := func() string { return fmt.Sprintf("invalid variable syntax in %q: \"$\" must start ${NAME}", s) }
		if i+2 >= len(s) || s[i+1] != '{' {
			return bad()
		}
		n := strings.IndexByte(s[i+2:], '}')
		if n < 0 {
			return bad()
		}
		name := s[i+2 : i+2+n]
		if !validVarName(name) {
			return bad()
		}
		end := i + 2 + n + 1
		if msg := ref(name, i, end); msg != "" {
			return msg
		}
		i = end - 1
	}
	return ""
}

func validVarName(s string) bool {
	if s == "" || s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}

func undefinedVar(name string) string {
	return fmt.Sprintf("undefined variable ${%s}", name)
}

// checkVars checks the variable syntax of s and that every name is defined.
func checkVars(s string, defined func(name string) bool) string {
	return scanVars(s, func(name string, _, _ int) string {
		if !defined(name) {
			return undefinedVar(name)
		}
		return ""
	})
}

// substVars replaces every ${NAME} of s. The second result is a message
// when the syntax is invalid or a name is undefined.
func substVars(s string, vars map[string]string) (string, string) {
	var b strings.Builder
	last := 0
	msg := scanVars(s, func(name string, start, end int) string {
		v, ok := vars[name]
		if !ok {
			return undefinedVar(name)
		}
		b.WriteString(s[last:start])
		b.WriteString(v)
		last = end
		return ""
	})
	if msg != "" {
		return "", msg
	}
	b.WriteString(s[last:])
	return b.String(), ""
}
