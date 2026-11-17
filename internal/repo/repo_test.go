// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/samlzz/situ/internal/exitcode"
	"github.com/samlzz/situ/internal/situfile"
	"github.com/samlzz/situ/internal/testutil"
)

func id(cat, comp string) situfile.ModuleID { return situfile.ModuleID{Category: cat, Component: comp} }

func host(h string) situfile.ModuleID { return id("hosts", h) }

func wantCode(t *testing.T, err error, code int, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("no error, want code %d with %q", code, substr)
	}
	if got := exitcode.CodeOf(err); got != code {
		t.Errorf("code = %d, want %d (%v)", got, code, err)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Errorf("error %q lacks %q", err, substr)
	}
}

func TestFindWithC(t *testing.T) {
	dir := testutil.Tree(t, map[string]string{"components/": "", "hosts/": ""})
	r, err := Find(dir, "/nonexistent")
	if err != nil || r.Dir != dir {
		t.Fatalf("Find = %v, %v", r, err)
	}
}

func TestFindWithCInvalid(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"no hosts":      {"components/": ""},
		"no components": {"hosts/": ""},
		"file":          {"components/": "", "hosts": "x"},
		"empty":         {"x/": ""},
	} {
		_, err := Find(testutil.Tree(t, files), "/")
		t.Run(name, func(t *testing.T) { wantCode(t, err, exitcode.Usage, "not a situ repository") })
	}
	_, err := Find(filepath.Join(t.TempDir(), "missing"), "/")
	wantCode(t, err, exitcode.Usage, "not a situ repository")
}

func TestFindWalksUp(t *testing.T) {
	dir := testutil.Tree(t, map[string]string{"components/": "", "hosts/": "", "components/a/b/deep/": ""})
	r, err := Find("", filepath.Join(dir, "components", "a", "b", "deep"))
	if err != nil || r.Dir != dir {
		t.Fatalf("Find = %+v, %v", r, err)
	}
}

func TestFindNone(t *testing.T) {
	_, err := Find("", t.TempDir())
	wantCode(t, err, exitcode.Usage, "no situ repository found")
}

func TestLoad(t *testing.T) {
	r := Open(testutil.Tree(t, map[string]string{
		"components/c/ok/Situfile":  testutil.Situfile("requires other", "enable a.service"),
		"components/c/bad/Situfile": testutil.Situfile("foo"),
	}))
	got, err := r.Load(id("c", "ok"))
	if err != nil || got.Path != "components/c/ok/Situfile" || len(got.Requires) != 1 || len(got.Units) != 1 {
		t.Fatalf("Load = %+v, %v", got, err)
	}
	_, err = r.Load(id("c", "bad"))
	wantCode(t, err, exitcode.Usage, `components/c/bad/Situfile:2: unknown keyword "foo"`)
	_, err = r.Load(id("c", "missing"))
	wantCode(t, err, exitcode.System, "no such file")
}
