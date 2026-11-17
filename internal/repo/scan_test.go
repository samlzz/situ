// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"reflect"
	"strings"
	"testing"

	"github.com/samlzz/situ/internal/exitcode"
	"github.com/samlzz/situ/internal/situfile"
	"github.com/samlzz/situ/internal/testutil"
)

func TestComponentsAndProfiles(t *testing.T) {
	dir := testutil.Tree(t, map[string]string{
		"components/zz/a/Situfile": testutil.Situfile(), "components/backup/core/Situfile": testutil.Situfile(),
		"components/backup/x/Situfile": testutil.Situfile(), "components/stray.txt": "x", "components/backup/note": "x",
		"hosts/pve/Situfile": testutil.Situfile(), "hosts/alpha/Situfile": testutil.Situfile(), "hosts/README": "x",
	})
	r := Open(dir)
	comps, err := r.Components()
	want := []situfile.ModuleID{id("backup", "core"), id("backup", "x"), id("zz", "a")}
	if err != nil || !reflect.DeepEqual(comps, want) {
		t.Fatalf("Components = %v, %v", comps, err)
	}
	hosts, err := r.Profiles()
	if err != nil || !reflect.DeepEqual(hosts, []situfile.ModuleID{host("alpha"), host("pve")}) {
		t.Fatalf("Profiles = %v, %v", hosts, err)
	}
}

func TestMissingSituFile(t *testing.T) {
	dir := testutil.Tree(t, map[string]string{
		"components/c/ok/Situfile": testutil.Situfile(), "components/c/x/": "", "components/c/y/Situfile/": "",
		"hosts/h/": "",
	})
	r := Open(dir)
	_, err := r.Components()
	wantCode(t, err, exitcode.Usage, "components/c/x: no Situfile")
	if !strings.Contains(err.Error(), "components/c/y/Situfile is not a regular file") {
		t.Errorf("both problems must be reported: %v", err)
	}
	_, err = r.Profiles()
	wantCode(t, err, exitcode.Usage, "hosts/h: no Situfile")
}

// The level is checked only when scanned.
func TestScanIsLazy(t *testing.T) {
	dir := testutil.Tree(t, map[string]string{"components/c/x/": "", "hosts/h/Situfile": testutil.Situfile()})
	r := Open(dir)
	if _, err := r.Profiles(); err != nil {
		t.Fatalf("Profiles must not read components/: %v", err)
	}
	if got, err := r.ResolveName("hosts/h"); err != nil || got != host("h") {
		t.Fatalf("ResolveName = %v, %v", got, err)
	}
}

func TestReservedCategoryAndInvalidNames(t *testing.T) {
	r := Open(testutil.Tree(t, map[string]string{"components/hosts/a/Situfile": testutil.Situfile()}))
	_, err := r.Components()
	wantCode(t, err, exitcode.Usage, `"hosts" is reserved`)

	r = Open(testutil.Tree(t, map[string]string{"components/Bad/a/Situfile": testutil.Situfile(), "components/ok/-a/Situfile": testutil.Situfile()}))
	_, err = r.Components()
	wantCode(t, err, exitcode.Usage, "components/Bad: invalid category name")
	if !strings.Contains(err.Error(), "components/ok/-a: invalid component name") {
		t.Errorf("err = %v", err)
	}

	r = Open(testutil.Tree(t, map[string]string{"hosts/PVE/Situfile": testutil.Situfile()}))
	_, err = r.Profiles()
	wantCode(t, err, exitcode.Usage, "hosts/PVE: invalid host name")
}
