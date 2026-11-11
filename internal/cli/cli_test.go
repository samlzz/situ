// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/samlzz/situ/internal/help"
)

type result struct {
	code        int
	out, errOut string
}

func run(args ...string) result {
	var out, errOut bytes.Buffer
	a := &App{Stdout: &out, Stderr: &errOut, Version: func() string { return "situ v9.9.9" }}
	code := a.Main(args)
	return result{code, out.String(), errOut.String()}
}

var cmdNames = []string{"install", "check", "adopt", "remove", "list", "version", "help"}

func TestHelpTopLevel(t *testing.T) {
	want := run("help")
	if want.code != 0 || want.errOut != "" {
		t.Fatalf("help: %+v", want)
	}
	for _, s := range slices.Concat(cmdNames, help.Topics()) {
		if !strings.Contains(want.out, s) {
			t.Errorf("help output lacks %q", s)
		}
	}
	if got := run("--help"); got != want {
		t.Fatalf("--help differs from help:\n%+v\n%+v", got, want)
	}
	// --help after global options.
	if r := run("-C", "x", "--root=/tmp", "--help"); r != want {
		t.Fatalf("got %+v", r)
	}
}

func TestHelpEachCommand(t *testing.T) {
	for _, c := range cmdNames {
		r := run(c, "--help")
		
		if w := fmt.Sprintf("usage: situ [global options] %s", c);
		r.code != 0 || r.errOut != "" || !strings.HasPrefix(r.out, w) {
			t.Errorf("%s --help: %+v", c, r)
		}
		if strings.Contains(r.out, "not implemented") {
			t.Errorf("%s --help: not implemented", c)
		}
	}
	// --help wins over other arguments, even invalid ones
	for _, args := range [][]string{
		{"install", "--force", "--help"}, {"install", "mod", "--help"},
		{"remove", "-r", "--help"}, {"list", "a", "b", "--help"}, {"check", "/p", "--help"},
	} {
		if r := run(args...); r.code != 0 || r.errOut != "" || !strings.HasPrefix(r.out, "usage: situ") {
			t.Errorf("%v: %+v", args, r)
		}
	}
}

func TestVersion(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"--version"}, {"-C", "d", "--version"}} {
		if r := run(args...); r != (result{0, "situ v9.9.9\n", ""}) {
			t.Errorf("%v: %+v", args, r)
		}
	}
	for _, args := range [][]string{{"version", "--version"}, {"install", "--version"}, {"version", "x"}} {
		if r := run(args...); r.code != 2 || r.out != "" {
			t.Errorf("%v: %+v", args, r)
		}
	}
}

func TestShortHRejected(t *testing.T) {
	for _, args := range []string{"-h", "install -h", "-C d -h"} {
		r := run(strings.Fields(args)...)
		if r.code != 2 || r.out != "" || !strings.Contains(r.errOut, "unknown option -h") {
			t.Errorf("%v: %+v", args, r)
		}
	}
}

func TestHelpTopics(t *testing.T) {
	for _, topic := range []string{"situfile", "exit-codes"} {
		text, ok := help.Topic(topic)
		if !ok || text == "" {
			t.Fatalf("topic %s missing", topic)
		}
		if r := run("help", topic); r != (result{0, text, ""}) {
			t.Errorf("help %s: %+v", topic, r)
		}
	}
	if r := run("help", "status"); r.code != 3 || r.out != "" || r.errOut != "situ: help topic status: not implemented\n" {
		t.Errorf("help status: %+v", r)
	}
	r := run("help", "nope")
	if r.code != 2 || r.out != "" || !strings.HasPrefix(r.errOut, "situ: no help topic \"nope\" (use situ help)\n") {
		t.Errorf("unknown topic: %+v", r)
	}
	r = run("help", "install")
	if r.code != 2 || !strings.HasPrefix(r.errOut, "situ: no help topic \"install\" (use situ install --help)\n") {
		t.Errorf("command as topic: %+v", r)
	}
	if r := run("help", "a", "b"); r.code != 2 {
		t.Errorf("help a b: %+v", r)
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		msg  string
	}{
		{"no arguments", nil, "missing command"},
		{"only global options", []string{"-C", "d"}, "missing command"},
		{"unknown command", []string{"frobnicate"}, "unknown command \"frobnicate\""},
		{"unknown global option", []string{"--bogus", "check"}, "unknown option --bogus"},
		{"-C without argument", []string{"-C"}, "option -C requires an argument"},
		{"--root without argument", []string{"--root"}, "option --root requires an argument"},
		{"global option after command", []string{"check", "--root", "x"}, "check: unknown option --root"},
		{"-C after command", []string{"check", "-C", "x"}, "check: unknown option -C"},
		{"unknown command option", []string{"check", "--force"}, "check: unknown option --force"},
		{"--force without module", []string{"install", "--force"}, "install: --force requires a module"},
		{"option after target", []string{"install", "foo", "--force"}, "install: option --force after a target (options must come before targets)"},
		{"-- not supported", []string{"install", "--", "foo"}, "install: unknown option --"},
		{"path to install", []string{"install", "/etc/a"}, "install: /etc/a is a path"},
		{"path to check", []string{"check", "/etc/a"}, "check: /etc/a is a path"},
		{"path to remove", []string{"remove", "/etc/a"}, "remove: /etc/a is a path"},
		{"remove no module", []string{"remove"}, "remove: expected at least 1 module, got 0"},
		{"remove flags no module", []string{"remove", "-r", "--purge"}, "remove: expected at least 1 module, got 0"},
		{"adopt no target", []string{"adopt"}, "adopt: expected at least 1 module, got 0"},
		{"list two targets", []string{"list", "a", "b"}, "list: expected at most 1 argument(s), got 2"},
		{"empty target", []string{"check", ""}, "check: empty target"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := run(tt.args...)
			if r.code != 2 || r.out != "" {
				t.Fatalf("code=%d out=%q", r.code, r.out)
			}
			if !strings.HasPrefix(r.errOut, "situ: "+tt.msg+"\n") {
				t.Fatalf("stderr = %q", r.errOut)
			}
			if !strings.Contains(r.errOut, "usage: situ") {
				t.Fatalf("no short usage: %q", r.errOut)
			}
		})
	}
}

func TestStubsExit3(t *testing.T) {
	tests := []struct {
		args []string
		cmd  string
	}{
		{[]string{"install"}, "install"},
		{[]string{"install", "--force", "a", "b/c"}, "install"},
		{[]string{"check"}, "check"},
		{[]string{"check", "a"}, "check"},
		{[]string{"remove", "-r", "--purge", "a"}, "remove"},
		{[]string{"remove", "--recursive", "a"}, "remove"},
		{[]string{"adopt", "/etc/a", "mod"}, "adopt"},
		{[]string{"list"}, "list"},
		{[]string{"list", "/etc/a"}, "list"},
		{[]string{"-C", "d", "--root", "/tmp/r", "list", "mod"}, "list"},
		{[]string{"--root=/tmp/r", "check"}, "check"},
	}
	for _, tt := range tests {
		r := run(tt.args...)
		if r.code != 3 || r.out != "" || r.errOut != "situ: "+tt.cmd+": not implemented\n" {
			t.Errorf("%v: %+v", tt.args, r)
		}
	}
}

func TestGlobalOptionsStored(t *testing.T) {
	var out, errOut bytes.Buffer
	a := &App{Stdout: &out, Stderr: &errOut, Version: func() string { return "" }}
	a.Main([]string{"-C", "one", "-C", "two", "--root", "/r1", "--root=/r2", "check"})
	if a.Opts != (Options{Dir: "two", Root: "/r2"}) {
		t.Fatalf("Opts = %+v", a.Opts)
	}
}
