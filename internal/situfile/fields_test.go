// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"io/fs"
	"testing"
)

func TestParseMode(t *testing.T) {
	tests := []struct {
		in   string
		want fs.FileMode
		ok   bool
	}{
		{"0644", 0o644, true},
		{"4755", 0o755 | fs.ModeSetuid, true},
		{"2775", 0o775 | fs.ModeSetgid, true},
		{"1777", 0o777 | fs.ModeSticky, true},
		{"7000", fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky, true},
		{"644", 0, false},
		{"0948", 0, false},
		{"00644", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := parseMode(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("got %v, %v", got, ok)
			}
		})
	}
}

func TestParseOwner(t *testing.T) {
	tests := []struct {
		in   string
		want Owner
		ok   bool
	}{
		{"root:root", Owner{"root", "root"}, true},
		{"_svc$:1000", Owner{"_svc$", "1000"}, true},
		{"0:0", Owner{"0", "0"}, true},
		{"root", Owner{}, false},
		{"root:", Owner{}, false},
		{"Root:root", Owner{}, false},
		{"a:b:c", Owner{}, false},
		{"4294967296:0", Owner{}, false},
		{"1a:root", Owner{}, false},
		{"a$$:root", Owner{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := parseOwner(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("got %+v, %v", got, ok)
			}
		})
	}
}

func TestParseTarget(t *testing.T) {
	tests := []struct {
		in   string
		want ModuleID
		ok   bool
	}{
		{"a", ModuleID{Component: "a"}, true},
		{"x/a-1", ModuleID{Category: "x", Component: "a-1"}, true},
		{"hosts/pve", ModuleID{Category: "hosts", Component: "pve"}, true},
		{"Foo", ModuleID{}, false},
		{"-foo", ModuleID{}, false},
		{"a/b/c", ModuleID{}, false},
		{"a/", ModuleID{}, false},
		{"", ModuleID{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := parseTarget(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("got %+v, %v", got, ok)
			}
		})
	}
}

func TestCheckSourceField(t *testing.T) {
	tests := []struct {
		name string
		typ  EntryType
		in   string
		want string
	}{
		{"dir dash", Dir, "-", ""},
		{"data dash", Data, "-", ""},
		{"dir value", Dir, "a", `dir: source must be "-", got "a"`},
		{"data value", Data, "a", `data: source must be "-", got "a"`},
		{"file dash", File, "-", `file: source is required ("-" means no source)`},
		{"config dash", Config, "-", `config: source is required ("-" means no source)`},
		{"file path", File, "files/a", ""},
		{"config path", Config, "a", ""},
		{"file bad path", File, "/a", `source "/a" must be relative`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkSourceField(tt.typ, tt.in); got != tt.want {
				t.Fatalf("got %q", got)
			}
		})
	}
}

func TestValidName(t *testing.T) {
	for in, want := range map[string]bool{
		"a": true, "a-b": true, "0a": true, "a1": true,
		"": false, "-a": false, "A": false, "a_b": false, "a/b": false,
	} {
		if got := validName(in); got != want {
			t.Errorf("%q: %v", in, got)
		}
	}
}

func TestValidIDPart(t *testing.T) {
	for in, want := range map[string]bool{
		"root": true, "_svc$": true, "a-1": true, "1000": true, "4294967295": true,
		"4294967296": false, "a$$": false, "$": false, "": false, "1a": false, "-a": false, "A": false,
	} {
		if got := validIDPart(in); got != want {
			t.Errorf("%q: %v", in, got)
		}
	}
}

func TestValidUnit(t *testing.T) {
	for in, want := range map[string]bool{
		"a.service": true, "foo@bar.service": true, `dev-sda\x2d1.mount`: true, "a.slice": true,
		"foo": false, ".service": false, "a/b.service": false, "a.device": false, "a b.service": false,
	} {
		if got := validUnit(in); got != want {
			t.Errorf("%q: %v", in, got)
		}
	}
}

func TestIsDigits(t *testing.T) {
	for in, want := range map[string]bool{"0": true, "123": true, "": false, "1a": false, "-1": false} {
		if got := isDigits(in); got != want {
			t.Errorf("%q: %v", in, got)
		}
	}
}
