// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/samlzz/situ/internal/testutil"
)

const fixture = "../../testdata/repo"

// runAs runs situ with a fake host name and working directory.
func runAs(hostname, cwd string, args ...string) result {
	var out, errOut bytes.Buffer
	a := New(&out, &errOut, func() string { return "situ v9.9.9" })
	a.Hostname = func() (string, error) { return hostname, nil }
	a.Getwd = func() (string, error) { return cwd, nil }
	code := a.Main(args)
	return result{code, out.String(), errOut.String()}
}

func wantOut(t *testing.T, r result, code int, out string) {
	t.Helper()
	if r.code != code || r.out != out {
		t.Errorf("code = %d (want %d), out:\n%s\nwant:\n%s\nstderr: %s", r.code, code, r.out, out, r.errOut)
	}
}

const profileListing = `hosts/host-b
  components/backup/etc-bundle
    components/backup/backup-core

components/backup/backup-core
  file  0755  root:root  backup-core  /usr/local/sbin/backup-core
  dir  0750  root:root  -  /var/backups

components/backup/etc-bundle
  requires  backup/backup-core
  file  0755  root:root  etc-bundle  /usr/local/sbin/etc-bundle
  file  0644  root:root  etc-bundle.service  /etc/systemd/system/etc-bundle.service
  config  0644  root:root  etc-bundle.default  /etc/default/etc-bundle
  dir  0700  root:root  -  /etc/etc-bundle
  data  0700  root:root  -  /var/backups/host-b/etc
  enable  etc-bundle.service

hosts/host-b
  requires  backup/etc-bundle
  file  0644  root:root  example.conf  /etc/etc-bundle/example.conf
`

func TestListCurrentProfile(t *testing.T) {
	wantOut(t, runAs("host-b.lan", "/", "-C", fixture, "list"), 0, profileListing)
}

func TestListFindsRepoFromCwd(t *testing.T) {
	wantOut(t, runAs("host-b", fixture+"/components/backup", "list"), 0, profileListing)
}

func TestListTreeAbove(t *testing.T) {
	dir := testutil.Tree(t, map[string]string{
		"hosts/h/Situfile":        testutil.Situfile("requires b", "requires a"),
		"components/c/a/Situfile": testutil.Situfile("requires c"),
		"components/c/b/Situfile": testutil.Situfile("requires c", "requires a"),
		"components/c/c/Situfile": testutil.Situfile(),
	})
	r := runAs("h", "/", "-C", dir, "list")
	tree, _, _ := strings.Cut(r.out, "\n\n")
	want := `hosts/h
  components/c/a
    components/c/c
  components/c/b
    components/c/a (above)
    components/c/c (above)`
	if r.code != 0 || tree != want {
		t.Errorf("tree:\n%s\nwant:\n%s\n(code %d, %s)", tree, want, r.code, r.errOut)
	}
}

func TestListNoProfile(t *testing.T) {
	r := runAs("laptop", "/", "-C", fixture, "list")
	if r.code != 2 || r.out != "" || !strings.Contains(r.errOut, `no host profile for host "laptop" (use situ list hosts/<host>)`) {
		t.Errorf("%+v", r)
	}
}

func TestListComponent(t *testing.T) {
	const unexpanded = `components/backup/etc-bundle
  requires  backup/backup-core
  file  0755  root:root  etc-bundle  /usr/local/sbin/etc-bundle
  file  0644  root:root  etc-bundle.service  /etc/systemd/system/etc-bundle.service
  config  0644  root:root  etc-bundle.default  /etc/default/etc-bundle
  dir  0700  root:root  -  /etc/etc-bundle
  data  0700  root:root  -  /var/backups/${HOST}/etc
  enable  etc-bundle.service
`
	// No current profile: ${HOST} stays.
	wantOut(t, runAs("laptop", "/", "-C", fixture, "list", "backup/etc-bundle"), 0, unexpanded)
	wantOut(t, runAs("laptop", "/", "-C", fixture, "list", "etc-bundle"), 0, unexpanded)
	// Current profile: expanded for it.
	expanded := strings.Replace(unexpanded, "${HOST}", "host-b", 1)
	wantOut(t, runAs("host-b", "/", "-C", fixture, "list", "etc-bundle"), 0, expanded)
}

func TestListProfileArgument(t *testing.T) {
	// Full listing, even without a current profile.
	wantOut(t, runAs("laptop", "/", "-C", fixture, "list", "hosts/host-b"), 0, profileListing)

	dir := testutil.Tree(t, map[string]string{
		"hosts/a/Situfile": testutil.Situfile("requires x"), "hosts/b/Situfile": testutil.Situfile("requires x"),
		"components/c/x/Situfile": testutil.Situfile("data 0700 root:root - /v/${HOST}"),
	})
	wantOut(t, runAs("a", "/", "-C", dir, "list", "hosts/b"), 0,
		"hosts/b\n  components/c/x\n\ncomponents/c/x\n  data  0700  root:root  -  /v/b\n\nhosts/b\n  requires  c/x\n")
	// A component is expanded with the current profile.
	wantOut(t, runAs("a", "/", "-C", dir, "list", "x"), 0, "components/c/x\n  data  0700  root:root  -  /v/a\n")
}

func TestListPathSearch(t *testing.T) {
	wantOut(t, runAs("laptop", "/", "-C", fixture, "list", "/etc/default/etc-bundle"), 0,
		"components/backup/etc-bundle/Situfile:7: config  0644  root:root  etc-bundle.default  /etc/default/etc-bundle  (hosts/host-a, hosts/host-b)\n")
	// The line is printed as written (variables unexpanded), found through its expansion.
	wantOut(t, runAs("laptop", "/", "-C", fixture, "list", "/var/backups/host-b/etc"), 0,
		"components/backup/etc-bundle/Situfile:9: data  0700  root:root  -  /var/backups/${HOST}/etc  (hosts/host-b)\n")
}

func TestListPathSeveralProfiles(t *testing.T) {
	dir := testutil.Tree(t, map[string]string{
		"hosts/b/Situfile": testutil.Situfile("requires x"), "hosts/a/Situfile": testutil.Situfile("requires x", "requires y"),
		"hosts/c/Situfile":        testutil.Situfile("requires y"),
		"components/c/x/Situfile": testutil.Situfile("", "file 0644 root:root f /etc/shared", "data 0700 root:root - /v/${HOST}"),
		"components/c/x/f":        "x",
		"components/c/y/Situfile": testutil.Situfile("file 0644 root:root g /etc/only-y"),
		"components/c/y/g":        "x",
	})
	// One line matched for two profiles is printed once.
	wantOut(t, runAs("a", "/", "-C", dir, "list", "/etc/shared"), 0,
		"components/c/x/Situfile:3: file  0644  root:root  f  /etc/shared  (hosts/a, hosts/b)\n")
	wantOut(t, runAs("a", "/", "-C", dir, "list", "/etc/only-y"), 0,
		"components/c/y/Situfile:2: file  0644  root:root  g  /etc/only-y  (hosts/a, hosts/c)\n")
	// Different expansions of the same line are different profiles of one line.
	wantOut(t, runAs("a", "/", "-C", dir, "list", "/v/b"), 0,
		"components/c/x/Situfile:4: data  0700  root:root  -  /v/${HOST}  (hosts/b)\n")
}

func TestListPathNotFound(t *testing.T) {
	r := runAs("host-b", "/", "-C", fixture, "list", "/nope")
	if r.code != 1 || r.out != "" || !strings.Contains(r.errOut, "no entry has destination /nope") {
		t.Errorf("%+v", r)
	}
}

func TestListErrors(t *testing.T) {
	r := runAs("host-b", "/", "-C", fixture, "list", "nope")
	if r.code != 2 || !strings.Contains(r.errOut, `unknown module "nope"`) {
		t.Errorf("%+v", r)
	}
	r = runAs("host-b", t.TempDir(), "list")
	if r.code != 2 || !strings.Contains(r.errOut, "no situ repository found") {
		t.Errorf("%+v", r)
	}
	// An invalid Situfile: path:line, code 2.
	dir := testutil.Tree(t, map[string]string{"hosts/h/Situfile": testutil.Situfile("bogus"), "components/": ""})
	r = runAs("h", "/", "-C", dir, "list")
	if r.code != 2 || !strings.Contains(r.errOut, `hosts/h/Situfile:2: unknown keyword "bogus"`) {
		t.Errorf("%+v", r)
	}
}

func TestListSystemErrors(t *testing.T) {
	var out, errOut bytes.Buffer
	a := New(&out, &errOut, func() string { return "" })
	a.Hostname = func() (string, error) { return "", errors.New("boom") }
	a.Getwd = func() (string, error) { return fixture, nil }
	if code := a.Main([]string{"list"}); code != 3 || !strings.Contains(errOut.String(), "boom") {
		t.Errorf("hostname: code %d, %s", code, errOut.String())
	}
	a.Getwd = func() (string, error) { return "", errors.New("no cwd") }
	errOut.Reset()
	if code := a.Main([]string{"list"}); code != 3 || !strings.Contains(errOut.String(), "no cwd") {
		t.Errorf("getwd: code %d, %s", code, errOut.String())
	}
}

// list reads the repository only: --root is never looked at.
func TestListIgnoresRoot(t *testing.T) {
	for _, args := range [][]string{
		{"-C", fixture, "--root", "/does/not/exist", "list"},
		{"-C", fixture, "--root", "/does/not/exist", "list", "etc-bundle"},
		{"-C", fixture, "--root", "/does/not/exist", "list", "/etc/default/etc-bundle"},
	} {
		if r := runAs("host-b", "/", args...); r.code != 0 {
			t.Errorf("%v: %+v", args, r)
		}
	}
}

func TestListHelp(t *testing.T) {
	r := run("list", "--help")
	if r.code != 0 || !strings.Contains(r.out, "dependency tree") || strings.Contains(r.out, "list the modules") {
		t.Errorf("%+v", r)
	}
}
