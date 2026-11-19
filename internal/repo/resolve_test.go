// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"errors"
	"testing"

	"github.com/samlzz/situ/internal/exitcode"
	"github.com/samlzz/situ/internal/situfile"
	"github.com/samlzz/situ/internal/testutil"
)

func namedRepo(t *testing.T) *Repo {
	return Open(testutil.Tree(t, map[string]string{
		"components/backup/core/Situfile": testutil.Situfile(), "components/backup/uniq/Situfile": testutil.Situfile(),
		"components/net/core/Situfile": testutil.Situfile(), "hosts/pve/Situfile": testutil.Situfile(),
	}))
}

func TestResolveName(t *testing.T) {
	r := namedRepo(t)
	for name, want := range map[string]situfile.ModuleID{
		"uniq": id("backup", "uniq"), "backup/core": id("backup", "core"),
		"net/core": id("net", "core"), "hosts/pve": host("pve"),
	} {
		if got, err := r.ResolveName(name); err != nil || got != want {
			t.Errorf("ResolveName(%q) = %v, %v; want %v", name, got, err, want)
		}
	}
}

func TestResolveNameErrors(t *testing.T) {
	r := namedRepo(t)
	_, err := r.ResolveName("core")
	wantCode(t, err, exitcode.Usage, "backup/core, net/core")
	if !errors.Is(err, ErrAmbiguous) {
		t.Error("want ErrAmbiguous")
	}
	for _, name := range []string{"nope", "backup/nope", "hosts/nope"} {
		_, err = r.ResolveName(name)
		wantCode(t, err, exitcode.Usage, "unknown module")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: want ErrNotFound", name)
		}
	}
	for _, name := range []string{"Bad", "a/b/c", "a//b", "../x"} {
		_, err = r.ResolveName(name)
		wantCode(t, err, exitcode.Usage, "invalid module name")
	}
}

func TestCurrentProfile(t *testing.T) {
	r := Open(testutil.Tree(t, map[string]string{"hosts/pve/Situfile": testutil.Situfile(), "hosts/nosf/": "", "hosts/afile": ""}))
	if got, ok, err := r.CurrentProfile("pve"); err != nil || !ok || got != host("pve") {
		t.Errorf("pve = %v %v %v", got, ok, err)
	}
	if got, ok, err := r.CurrentProfile("pve.example.org"); err != nil || !ok || got != host("pve") {
		t.Errorf("FQDN = %v %v %v", got, ok, err)
	}
	for _, h := range []string{"PVE", "Pve.Example.org"} {
		if got, ok, err := r.CurrentProfile(h); err != nil || !ok || got != host("pve") {
			t.Errorf("%q = %v %v %v, want pve", h, got, ok, err)
		}
	}
	for _, h := range []string{"other", "", "afile", ".x", "-bad"} {
		if _, ok, err := r.CurrentProfile(h); ok || err != nil {
			t.Errorf("%q: ok=%v err=%v, want no profile", h, ok, err)
		}
	}
	_, _, err := r.CurrentProfile("nosf")
	wantCode(t, err, exitcode.Usage, "hosts/nosf: no Situfile")
}
