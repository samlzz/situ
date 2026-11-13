// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/samlzz/situ/internal/exitcode"
)

const hdr = "format 1\n"

func errLines(t *testing.T, err error) []string {
	t.Helper()
	errs, ok := errors.AsType[Errors](err)
	if !ok {
		t.Fatalf("error is %T, want Errors: %v", err, err)
	}
	if exitcode.CodeOf(err) != exitcode.Usage {
		t.Fatalf("code = %d, want 2", exitcode.CodeOf(err))
	}
	out := make([]string, len(errs))
	for i, e := range errs {
		out[i] = e.Error()
	}
	return out
}

func TestParseValid(t *testing.T) {
	src := `# comment before format
format    1
requires  backup-core   # trailing comment
requires  backup/etc-bundle
	file	0755	root:root	etc-bundle	/usr/local/sbin/etc-bundle
config    0644  root:root  files/a.conf  /etc/a.conf
dir       2775  _svc$:1000  -  /etc/a
data      0700  root:root  -  /var/backups/${HOST}/etc
enable    a.service
start     a.service
enable    foo@bar.service
enable    dev-sda\\x2d1.mount
file      0644  root:root  x#y  /etc/x#y
`
	sf, err := Parse("Situfile", []byte(src), ModuleID{Category: "backup", Component: "me"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantReq := []Require{{3, ModuleID{Component: "backup-core"}}, {4, ModuleID{Category: "backup", Component: "etc-bundle"}}}
	if !reflect.DeepEqual(sf.Requires, wantReq) {
		t.Errorf("Requires = %+v", sf.Requires)
	}
	if len(sf.Entries) != 5 || len(sf.Units) != 4 {
		t.Fatalf("entries=%d units=%d", len(sf.Entries), len(sf.Units))
	}
	e := sf.Entries[0]
	if e.Line != 5 || e.Type != File || e.Mode != 0o755 || e.Owner != (Owner{"root", "root"}) ||
		e.Source != "etc-bundle" || e.Dest != "/usr/local/sbin/etc-bundle" {
		t.Errorf("entry 0 = %+v", e)
	}
	if got := sf.Entries[2].Mode; got != 0o775|fs.ModeSetgid {
		t.Errorf("mode 2775 = %v", got)
	}
	if sf.Entries[2].Source != "" || sf.Entries[3].Source != "" {
		t.Errorf("dir/data Source = %q, %q, want empty", sf.Entries[2].Source, sf.Entries[3].Source)
	}
	if sf.Entries[2].Owner != (Owner{"_svc$", "1000"}) {
		t.Errorf("owner = %+v", sf.Entries[2].Owner)
	}
	if sf.Entries[3].Dest != "/var/backups/${HOST}/etc" {
		t.Errorf("dest = %q", sf.Entries[3].Dest)
	}
	if sf.Units[1].Action != Start || sf.Units[1].Name != "a.service" {
		t.Errorf("unit 1 = %+v", sf.Units[1])
	}
}

func TestParseModeBits(t *testing.T) {
	for in, want := range map[string]fs.FileMode{
		"0644": 0o644, "4755": 0o755 | fs.ModeSetuid, "1777": 0o777 | fs.ModeSticky,
		"7000": fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky,
	} {
		sf, err := Parse("S", []byte(hdr+"dir "+in+" root:root - /d\n"), ModuleID{})
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if sf.Entries[0].Mode != want {
			t.Errorf("%s: got %v want %v", in, sf.Entries[0].Mode, want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	const ok = "root:root"
	tests := []struct {
		name string
		src  string
		self ModuleID
		want []string
	}{
		{"empty file", "", ModuleID{}, []string{"S:1: missing format statement"}},
		{"comments only", "# a\n\n  \n", ModuleID{}, []string{"S:1: missing format statement"}},
		{"format missing, other stmt", "enable a.service\n", ModuleID{}, []string{"S:1: missing format statement"}},
		{"format not first", "enable a.service\nformat 1\n", ModuleID{}, []string{"S:1: format must be the first statement"}},
		{"format twice", hdr + "format 1\n", ModuleID{}, []string{"S:2: duplicate format statement"}},
		{"format version 2", "format 2\nfile bad\n", ModuleID{}, []string{"S:1: unsupported format version \"2\" (this situ reads version 1)"}},
		{"format version 01", "format 01\n", ModuleID{}, []string{"S:1: unsupported format version \"01\" (this situ reads version 1)"}},
		{"format no field", "format\n", ModuleID{}, []string{"S:1: format: expected 1 field, got 0"}},
		{"format two fields", "format 1 2\n", ModuleID{}, []string{"S:1: format: expected 1 field, got 2"}},
		{"CR", "format 1\r\nenable a.service\n", ModuleID{}, []string{
			"S:1: carriage return is not allowed (LF line endings only)", "S:2: missing format statement"}},
		{"CR in comment", hdr + "# x\r\n", ModuleID{}, []string{"S:2: carriage return is not allowed (LF line endings only)"}},
		{"invalid UTF-8", hdr + "# \xff\n", ModuleID{}, []string{"S:2: invalid UTF-8"}},
		{"unknown keyword", hdr + "bogus x\n", ModuleID{}, []string{"S:2: unknown keyword \"bogus\""}},
		{"keyword case", hdr + "File 0644 root:root a /a\n", ModuleID{}, []string{"S:2: unknown keyword \"File\""}},
		{"file too few", hdr + "file 0644 root:root a\n", ModuleID{}, []string{"S:2: file: expected 4 fields after the keyword, got 3"}},
		{"file too many", hdr + "file 0644 root:root a /a extra\n", ModuleID{}, []string{"S:2: file: expected 4 fields after the keyword, got 5"}},
		{"enable too many", hdr + "enable a.service b.service\n", ModuleID{}, []string{"S:2: enable: expected 1 fields after the keyword, got 2"}},
		{"requires none", hdr + "requires\n", ModuleID{}, []string{"S:2: requires: expected 1 fields after the keyword, got 0"}},
		{"mode 3 digits", hdr + "file 644 " + ok + " a /a\n", ModuleID{}, []string{"S:2: invalid mode \"644\" (want four octal digits)"}},
		{"mode 8", hdr + "file 0948 " + ok + " a /a\n", ModuleID{}, []string{"S:2: invalid mode \"0948\" (want four octal digits)"}},
		{"mode 5 digits", hdr + "file 00644 " + ok + " a /a\n", ModuleID{}, []string{"S:2: invalid mode \"00644\" (want four octal digits)"}},
		{"owner no group", hdr + "file 0644 root a /a\n", ModuleID{}, []string{"S:2: invalid owner \"root\" (want <user>:<group>, each a name or a decimal ID)"}},
		{"owner empty part", hdr + "file 0644 root: a /a\n", ModuleID{}, []string{"S:2: invalid owner \"root:\" (want <user>:<group>, each a name or a decimal ID)"}},
		{"owner uppercase", hdr + "file 0644 Root:root a /a\n", ModuleID{}, []string{"S:2: invalid owner \"Root:root\" (want <user>:<group>, each a name or a decimal ID)"}},
		{"owner 3 parts", hdr + "file 0644 a:b:c a /a\n", ModuleID{}, []string{"S:2: invalid owner \"a:b:c\" (want <user>:<group>, each a name or a decimal ID)"}},
		{"owner id too big", hdr + "file 0644 4294967296:0 a /a\n", ModuleID{}, []string{"S:2: invalid owner \"4294967296:0\" (want <user>:<group>, each a name or a decimal ID)"}},
		{"owner digit first", hdr + "file 0644 1a:root a /a\n", ModuleID{}, []string{"S:2: invalid owner \"1a:root\" (want <user>:<group>, each a name or a decimal ID)"}},
		{"owner double dollar", hdr + "file 0644 a$$:root a /a\n", ModuleID{}, []string{"S:2: invalid owner \"a$$:root\" (want <user>:<group>, each a name or a decimal ID)"}},
		{"source absolute", hdr + "file 0644 " + ok + " /a /a\n", ModuleID{}, []string{"S:2: source \"/a\" must be relative"}},
		{"source dot", hdr + "file 0644 " + ok + " ./a /a\n", ModuleID{}, []string{"S:2: source \"./a\" must not contain \".\" components"}},
		{"source dotdot", hdr + "file 0644 " + ok + " a/../b /a\n", ModuleID{}, []string{"S:2: source \"a/../b\" must not contain \"..\" components"}},
		{"source empty comp", hdr + "file 0644 " + ok + " a//b /a\n", ModuleID{}, []string{"S:2: source \"a//b\" must not contain empty components"}},
		{"source trailing slash", hdr + "file 0644 " + ok + " a/ /a\n", ModuleID{}, []string{"S:2: source \"a/\" must not end with \"/\""}},
		{"dir source not dash", hdr + "dir 0755 " + ok + " a /a\n", ModuleID{}, []string{"S:2: dir: source must be \"-\", got \"a\""}},
		{"data source not dash", hdr + "data 0755 " + ok + " a /a\n", ModuleID{}, []string{"S:2: data: source must be \"-\", got \"a\""}},
		{"dest relative", hdr + "file 0644 " + ok + " a etc/a\n", ModuleID{}, []string{"S:2: destination \"etc/a\" must be absolute"}},
		{"dest root", hdr + "dir 0755 " + ok + " - /\n", ModuleID{}, []string{"S:2: destination must not be \"/\""}},
		{"dest dotdot", hdr + "dir 0755 " + ok + " - /a/../b\n", ModuleID{}, []string{"S:2: destination \"/a/../b\" must not contain \"..\" components"}},
		{"dest dot", hdr + "dir 0755 " + ok + " - /a/./b\n", ModuleID{}, []string{"S:2: destination \"/a/./b\" must not contain \".\" components"}},
		{"dest empty comp", hdr + "dir 0755 " + ok + " - /a//b\n", ModuleID{}, []string{"S:2: destination \"/a//b\" must not contain empty components"}},
		{"dest trailing slash", hdr + "dir 0755 " + ok + " - /a/\n", ModuleID{}, []string{"S:2: destination \"/a/\" must not end with \"/\""}},
		{"dest undefined var", hdr + "dir 0755 " + ok + " - /a/${FOO}\n", ModuleID{}, []string{"S:2: undefined variable ${FOO}"}},
		{"dest bare dollar", hdr + "dir 0755 " + ok + " - /a/$HOST\n", ModuleID{}, []string{"S:2: invalid variable syntax in \"/a/$HOST\": \"$\" must start ${NAME}"}},
		{"dest unclosed var", hdr + "dir 0755 " + ok + " - /a/${HOST\n", ModuleID{}, []string{"S:2: invalid variable syntax in \"/a/${HOST\": \"$\" must start ${NAME}"}},
		{"dest lowercase var", hdr + "dir 0755 " + ok + " - /a/${host}\n", ModuleID{}, []string{"S:2: invalid variable syntax in \"/a/${host}\": \"$\" must start ${NAME}"}},
		{"dest trailing dollar", hdr + "dir 0755 " + ok + " - /a$\n", ModuleID{}, []string{"S:2: invalid variable syntax in \"/a$\": \"$\" must start ${NAME}"}},
		{"dest duplicate", hdr + "dir 0755 " + ok + " - /a\nfile 0644 " + ok + " x /a\n", ModuleID{}, []string{"S:3: duplicate destination \"/a\" (first used at line 2)"}},
		{"unit no suffix", hdr + "enable foo\n", ModuleID{}, []string{"S:2: invalid unit name \"foo\""}},
		{"unit bare suffix", hdr + "enable .service\n", ModuleID{}, []string{"S:2: invalid unit name \".service\""}},
		{"unit slash", hdr + "enable a/b.service\n", ModuleID{}, []string{"S:2: invalid unit name \"a/b.service\""}},
		{"unit bad suffix", hdr + "start a.device\n", ModuleID{}, []string{"S:2: invalid unit name \"a.device\""}},
		{"unit twice same role", hdr + "enable a.service\nenable a.service\n", ModuleID{}, []string{"S:3: unit \"a.service\" listed twice under enable (first at line 2)"}},
		{"start twice", hdr + "start a.timer\nstart a.timer\n", ModuleID{}, []string{"S:3: unit \"a.timer\" listed twice under start (first at line 2)"}},
		{"target uppercase", hdr + "requires Foo\n", ModuleID{}, []string{"S:2: invalid requires target \"Foo\" (want <component> or <category>/<component>)"}},
		{"target leading dash", hdr + "requires -foo\n", ModuleID{}, []string{"S:2: invalid requires target \"-foo\" (want <component> or <category>/<component>)"}},
		{"target 3 parts", hdr + "requires a/b/c\n", ModuleID{}, []string{"S:2: invalid requires target \"a/b/c\" (want <component> or <category>/<component>)"}},
		{"target empty part", hdr + "requires a/\n", ModuleID{}, []string{"S:2: invalid requires target \"a/\" (want <component> or <category>/<component>)"}},
		{"target host profile", hdr + "requires hosts/pve\n", ModuleID{}, []string{"S:2: invalid requires target \"hosts/pve\": a host profile cannot be a target"}},
		{"target twice", hdr + "requires a\nrequires a\n", ModuleID{}, []string{"S:3: target \"a\" listed twice (first at line 2)"}},
		{"full target twice", hdr + "requires x/a\nrequires x/a\n", ModuleID{}, []string{"S:3: target \"x/a\" listed twice (first at line 2)"}},
		{"short then full", hdr + "requires a\nrequires x/a\n", ModuleID{}, []string{"S:3: target \"x/a\" duplicates or is ambiguous with \"a\" (line 2)"}},
		{"full then short", hdr + "requires x/a\nrequires a\n", ModuleID{}, []string{"S:3: target \"a\" duplicates or is ambiguous with \"x/a\" (line 2)"}},
		{"self short", hdr + "requires me\n", ModuleID{Category: "c", Component: "me"}, []string{"S:2: \"me\" requires itself"}},
		{"self full", hdr + "requires c/me\n", ModuleID{Category: "c", Component: "me"}, []string{"S:2: \"c/me\" requires itself"}},
		{"multiple errors", hdr + "file 0644 root a /a\nbogus\nenable x\n", ModuleID{}, []string{
			"S:2: invalid owner \"root\" (want <user>:<group>, each a name or a decimal ID)",
			"S:3: unknown keyword \"bogus\"",
			"S:4: invalid unit name \"x\"",
		}},
		{"several errors one line", hdr + "file 99 x y /\n", ModuleID{}, []string{
			"S:2: invalid mode \"99\" (want four octal digits)",
			"S:2: invalid owner \"x\" (want <user>:<group>, each a name or a decimal ID)",
			"S:2: destination must not be \"/\"",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sf, err := Parse("S", []byte(tt.src), tt.self)
			if err == nil {
				t.Fatalf("expected errors, got %+v", sf)
			}
			got := errLines(t, err)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

func TestParseOtherSelfNotRejected(t *testing.T) {
	// Same component name in another category is not a self-require when
	// written in full form.
	_, err := Parse("S", []byte(hdr+"requires other/me\n"), ModuleID{Category: "c", Component: "me"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEnumString(t *testing.T) {
	for typ, want := range map[EntryType]string{File: "file", Config: "config", Dir: "dir", Data: "data", 0: "invalid"} {
		if typ.String() != want {
			t.Errorf("EntryType %d: %q", typ, typ.String())
		}
	}
	for a, want := range map[UnitAction]string{Enable: "enable", Start: "start", 0: "invalid"} {
		if a.String() != want {
			t.Errorf("UnitAction %d: %q", a, a.String())
		}
	}
}

func TestModuleIDString(t *testing.T) {
	if s := (ModuleID{Component: "a"}).String(); s != "a" {
		t.Errorf("short: %q", s)
	}
	if s := (ModuleID{Category: "c", Component: "a"}).String(); s != "c/a" {
		t.Errorf("full: %q", s)
	}
}

func TestErrorsString(t *testing.T) {
	e := Errors{{"a", 1, "x"}, {"a", 2, "y"}}
	if e.Error() != "a:1: x\na:2: y" {
		t.Fatalf("%q", e.Error())
	}
}
