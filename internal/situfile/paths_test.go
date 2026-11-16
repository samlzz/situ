// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import "testing"

func TestCheckSourcePath(t *testing.T) {
	tests := map[string]string{
		"a":        "",
		"a/b.conf": "",
		"x#y":      "",
		"/a":       `source "/a" must be relative`,
		"./a":      `source "./a" must not contain "." components`,
		"a/../b":   `source "a/../b" must not contain ".." components`,
		"a//b":     `source "a//b" must not contain empty components`,
		"a/":       `source "a/" must not end with "/"`,
	}
	for in, want := range tests {
		if got := checkSourcePath(in); got != want {
			t.Errorf("%q: got %q", in, got)
		}
	}
}

func TestCheckDest(t *testing.T) {
	tests := map[string]string{
		"/a":      "",
		"/a/${H}": "",
		"etc/a":   `destination "etc/a" must be absolute`,
		"/":       `destination must not be "/"`,
		"/a/../b": `destination "/a/../b" must not contain ".." components`,
		"/a/./b":  `destination "/a/./b" must not contain "." components`,
		"/a//b":   `destination "/a//b" must not contain empty components`,
		"/a/":     `destination "/a/" must not end with "/"`,
	}
	for in, want := range tests {
		if got := checkDest(in); got != want {
			t.Errorf("%q: got %q", in, got)
		}
	}
}

func TestCheckComponents(t *testing.T) {
	tests := map[string]string{
		"a/b":  "",
		"a/":   `x "F" must not end with "/"`,
		"a//b": `x "F" must not contain empty components`,
		"a/.":  `x "F" must not contain "." components`,
		"..":   `x "F" must not contain ".." components`,
	}
	for in, want := range tests {
		if got := checkComponents("x", "F", in); got != want {
			t.Errorf("%q: got %q", in, got)
		}
	}
}
