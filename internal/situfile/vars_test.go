// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import "testing"

func TestValidVarName(t *testing.T) {
	for in, want := range map[string]bool{
		"HOST": true, "A1_B": true, "": false, "host": false, "1A": false, "_A": false, "A-B": false,
	} {
		if got := validVarName(in); got != want {
			t.Errorf("%q: %v", in, got)
		}
	}
}

func TestCheckVars(t *testing.T) {
	defined := func(n string) bool { return n == "HOST" }
	bad := func(s string) string {
		return `invalid variable syntax in "` + s + `": "$" must start ${NAME}`
	}
	tests := map[string]string{
		"/a":                 "",
		"/a/${HOST}/${HOST}": "",
		"/a/$HOST":           bad("/a/$HOST"),
		"/a/${HOST":          bad("/a/${HOST"),
		"/a/${host}":         bad("/a/${host}"),
		"/a$":                bad("/a$"),
		"/a/${FOO}":          "undefined variable ${FOO}",
	}
	for in, want := range tests {
		if got := checkVars(in, defined); got != want {
			t.Errorf("%q: got %q", in, got)
		}
	}
}

func TestSubstVars(t *testing.T) {
	vars := map[string]string{"HOST": "h1", "E": ""}
	tests := []struct {
		in, want, msg string
	}{
		{"/a", "/a", ""},
		{"/${HOST}/x/${HOST}", "/h1/x/h1", ""},
		{"/a/${E}/b", "/a//b", ""},
		{"${HOST}", "h1", ""},
		{"/a/${FOO}", "", "undefined variable ${FOO}"},
		{"/a$", "", `invalid variable syntax in "/a$": "$" must start ${NAME}`},
	}
	for _, tt := range tests {
		got, msg := substVars(tt.in, vars)
		if got != tt.want || msg != tt.msg {
			t.Errorf("%q: got %q, %q", tt.in, got, msg)
		}
	}
}
