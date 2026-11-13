// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"reflect"
	"testing"
)

func parseOK(t *testing.T, body string) *Situfile {
	t.Helper()
	sf, err := Parse("S", []byte(hdr+body), ModuleID{})
	if err != nil {
		t.Fatal(err)
	}
	return sf
}

func TestExpandHost(t *testing.T) {
	sf := parseOK(t, "data 0700 root:root - /var/backups/${HOST}/etc\nfile 0644 root:root a /etc/${HOST}/${HOST}.conf\nfile 0644 root:root b /etc/b\n")
	got, err := Expand(sf, map[string]string{"HOST": "pve-xps"})
	if err != nil {
		t.Fatal(err)
	}
	var dests []string
	for _, e := range got {
		dests = append(dests, e.Dest)
	}
	want := []string{"/var/backups/pve-xps/etc", "/etc/pve-xps/pve-xps.conf", "/etc/b"}
	if !reflect.DeepEqual(dests, want) {
		t.Fatalf("got %v", dests)
	}
	if sf.Entries[0].Dest != "/var/backups/${HOST}/etc" {
		t.Fatal("Expand modified its input")
	}
}

func TestExpandErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		vars map[string]string
		want []string
	}{
		{"undefined var", "dir 0755 root:root - /a/${HOST}\n", map[string]string{},
			[]string{"S:2: undefined variable ${HOST}"}},
		{"recheck: slash in value", "dir 0755 root:root - /a/${HOST}\n", map[string]string{"HOST": "x/"},
			[]string{"S:2: destination \"/a/x/\" must not end with \"/\" (expanded from \"/a/${HOST}\")"}},
		{"recheck: dotdot value", "dir 0755 root:root - /a/${HOST}/b\n", map[string]string{"HOST": ".."},
			[]string{"S:2: destination \"/a/../b\" must not contain \"..\" components (expanded from \"/a/${HOST}/b\")"}},
		{"recheck: empty value", "dir 0755 root:root - /a/${HOST}/b\n", map[string]string{"HOST": ""},
			[]string{"S:2: destination \"/a//b\" must not contain empty components (expanded from \"/a/${HOST}/b\")"}},
		{"recheck: root", "dir 0755 root:root - /${HOST}\n", map[string]string{"HOST": ""},
			[]string{"S:2: destination must not be \"/\" (expanded from \"/${HOST}\")"}},
		{"expanded duplicate", "dir 0755 root:root - /a/${HOST}\ndir 0755 root:root - /a/h1\n", map[string]string{"HOST": "h1"},
			[]string{"S:3: duplicate destination \"/a/h1\" after expansion (first used at line 2)"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sf := parseOK(t, tt.body)
			_, err := Expand(sf, tt.vars)
			if err == nil {
				t.Fatal("expected error")
			}
			if got := errLines(t, err); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}
