// SPDX-License-Identifier: GPL-3.0-or-later

package graph

import (
	"reflect"
	"strings"
	"testing"

	"github.com/samlzz/situ/internal/exitcode"
	"github.com/samlzz/situ/internal/repo"
	"github.com/samlzz/situ/internal/testutil"
)

func comp(cat, c string) ModuleID { return ModuleID{Category: cat, Component: c} }

func host(h string) ModuleID { return ModuleID{Category: "hosts", Component: h} }

func build(t *testing.T, files map[string]string, host string, roots ...ModuleID) (*Graph, error) {
	t.Helper()
	return Build(repo.Open(testutil.Tree(t, files)), roots, host)
}

func mustBuild(t *testing.T, files map[string]string, host string, roots ...ModuleID) *Graph {
	t.Helper()
	g, err := build(t, files, host, roots...)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func order(g *Graph) []string {
	var out []string
	for _, n := range g.Order {
		out = append(out, repo.Rel(n.ID))
	}
	return out
}

func wantErr(t *testing.T, err error, lines ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("no error, want %q", lines)
	}
	if exitcode.CodeOf(err) != exitcode.Usage {
		t.Errorf("code = %d, want 2", exitcode.CodeOf(err))
	}
	if got := strings.Split(err.Error(), "\n"); !reflect.DeepEqual(got, lines) {
		t.Errorf("error lines:\n got %q\nwant %q", got, lines)
	}
}

func TestTransitiveRequires(t *testing.T) {
	files := map[string]string{
		"hosts/h/Situfile":             testutil.Situfile("requires b/mid", "requires other"),
		"components/b/mid/Situfile":    testutil.Situfile("requires base"),
		"components/b/base/Situfile":   testutil.Situfile(),
		"components/c/other/Situfile":  testutil.Situfile(),
		"components/c/unused/Situfile": testutil.Situfile(),
	}
	g := mustBuild(t, files, "h", host("h"))
	want := []string{"components/b/base", "components/b/mid", "components/c/other", "hosts/h"}
	if got := order(g); !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
	n := g.Node(host("h"))
	if !reflect.DeepEqual(n.Deps, []ModuleID{comp("b", "mid"), comp("c", "other")}) {
		t.Errorf("Deps = %v", n.Deps)
	}
	if g.Node(comp("c", "unused")) != nil {
		t.Error("unused module in graph")
	}
}

func TestCycle(t *testing.T) {
	g, err := build(t, map[string]string{
		"components/c/a/Situfile": testutil.Situfile("requires b"),
		"components/c/b/Situfile": testutil.Situfile("requires d"),
		"components/c/d/Situfile": testutil.Situfile("", "requires c/a"),
	}, "", comp("c", "a"))
	if g != nil {
		t.Error("graph returned with an error")
	}
	wantErr(t, err, "components/c/d/Situfile:3: dependency cycle: c/a -> c/b -> c/d -> c/a")
}

func TestMissingAndAmbiguousTarget(t *testing.T) {
	files := map[string]string{
		"components/a/x/Situfile":   testutil.Situfile("requires nope", "requires dup", "requires a/gone"),
		"components/a/dup/Situfile": testutil.Situfile(),
		"components/b/dup/Situfile": testutil.Situfile(),
	}
	_, err := build(t, files, "", comp("a", "x"))
	wantErr(t, err,
		`components/a/x/Situfile:2: requires: unknown module "nope"`,
		`components/a/x/Situfile:3: requires: ambiguous module name "dup": a/dup, b/dup`,
		`components/a/x/Situfile:4: requires: unknown module "a/gone"`)
}

func TestProfileAsTarget(t *testing.T) {
	_, err := build(t, map[string]string{
		"components/a/x/Situfile": testutil.Situfile("requires hosts/h"),
		"hosts/h/Situfile":        testutil.Situfile(),
	}, "", comp("a", "x"))
	wantErr(t, err, `components/a/x/Situfile:2: invalid requires target "hosts/h": a host profile cannot be a target`)
}

func TestDuplicateDestinationAcrossModules(t *testing.T) {
	files := map[string]string{
		"components/a/x/Situfile": testutil.Situfile("requires y", "config 0644 root:root s /etc/app.conf", "dir 0755 root:root - /etc/${HOST}"),
		"components/a/x/s":        "x",
		"components/a/y/Situfile": testutil.Situfile("", "dir 0755 root:root - /etc/app.conf", "dir 0755 root:root - /etc/h"),
		"hosts/h/Situfile":        testutil.Situfile("requires x"),
	}
	_, err := build(t, files, "h", host("h"))
	// y comes first in the order, so x is the one reported.
	wantErr(t, err,
		`components/a/x/Situfile:3: duplicate destination "/etc/app.conf", also declared by components/a/y at line 3`,
		`components/a/x/Situfile:4: duplicate destination "/etc/h", also declared by components/a/y at line 4`)
}

func TestDuplicateUnit(t *testing.T) {
	files := map[string]string{
		"components/a/x/Situfile": testutil.Situfile("requires y", "enable app.service", "start app.service"),
		"components/a/y/Situfile": testutil.Situfile("start app.service"),
	}
	_, err := build(t, files, "", comp("a", "x"))
	wantErr(t, err,
		`components/a/x/Situfile:3: duplicate unit "app.service", also declared by components/a/y at line 2`,
		`components/a/x/Situfile:4: duplicate unit "app.service", also declared by components/a/y at line 2`)

	// The same unit under enable and start of one module is allowed.
	mustBuild(t, map[string]string{"components/a/x/Situfile": testutil.Situfile("enable app.service", "start app.service")}, "", comp("a", "x"))
}

func TestOrderTies(t *testing.T) {
	files := map[string]string{
		"hosts/h/Situfile":        testutil.Situfile("requires c/b", "requires a/z", "requires a/m"),
		"components/c/b/Situfile": testutil.Situfile(),
		"components/a/z/Situfile": testutil.Situfile(),
		"components/a/m/Situfile": testutil.Situfile("requires c/b"),
		"hosts/g/Situfile":        testutil.Situfile(),
	}
	want := []string{"components/a/z", "components/c/b", "components/a/m", "hosts/h"}
	for range 3 {
		if got := order(mustBuild(t, files, "h", host("h"))); !reflect.DeepEqual(got, want) {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
	// Roots of different kinds: components first, then hosts.
	got := order(mustBuild(t, files, "", host("h"), host("g")))
	if !reflect.DeepEqual(got, []string{"components/a/z", "components/c/b", "components/a/m", "hosts/g", "hosts/h"}) {
		t.Errorf("order = %v", got)
	}
}

// Parse errors of several Situfiles are reported together; later stages
// (here a missing target) are not reached, and a failed file is not followed.
func TestParseErrorsReportedTogether(t *testing.T) {
	files := map[string]string{
		"hosts/h/Situfile":        testutil.Situfile("requires a", "requires b", "requires missing"),
		"components/c/a/Situfile": testutil.Situfile("bogus", "requires ghost"),
		"components/c/b/Situfile": testutil.Situfile("", "bogus2"),
	}
	_, err := build(t, files, "h", host("h"))
	wantErr(t, err,
		`components/c/a/Situfile:2: unknown keyword "bogus"`,
		`components/c/b/Situfile:3: unknown keyword "bogus2"`)
}

func TestScanErrorIsFatal(t *testing.T) {
	_, err := build(t, map[string]string{
		"components/c/a/Situfile": testutil.Situfile("requires b"),
		"components/c/nosf/":      "",
	}, "", comp("c", "a"))
	if err == nil || exitcode.CodeOf(err) != exitcode.Usage || !strings.Contains(err.Error(), "components/c/nosf: no Situfile") {
		t.Errorf("err = %v", err)
	}
}

func TestExpandPerProfile(t *testing.T) {
	files := map[string]string{
		"components/a/x/Situfile": testutil.Situfile("data 0700 root:root - /var/backups/${HOST}"),
	}
	for _, h := range []string{"one", "two"} {
		g := mustBuild(t, files, h, comp("a", "x"))
		if got := g.Node(comp("a", "x")).Entries[0].Dest; got != "/var/backups/"+h {
			t.Errorf("host %s: dest = %q", h, got)
		}
		if got := g.Node(comp("a", "x")).SF.Entries[0].Dest; got != "/var/backups/${HOST}" {
			t.Errorf("SF modified: %q", got)
		}
	}
}

func TestWithoutHostNothingIsExpandedNorChecked(t *testing.T) {
	files := map[string]string{
		// Collisions that exist only once expanded, or across modules.
		"components/a/x/Situfile": testutil.Situfile("requires y", "dir 0755 root:root - /d/${HOST}", "dir 0755 root:root - /d/h", "dir 0755 root:root - /same"),
		"components/a/y/Situfile": testutil.Situfile("dir 0755 root:root - /same"),
	}
	g := mustBuild(t, files, "", comp("a", "x"))
	if got := g.Node(comp("a", "x")).Entries[0].Dest; got != "/d/${HOST}" {
		t.Errorf("dest = %q", got)
	}
	_, err := build(t, files, "h", comp("a", "x"))
	// The collision inside x (after expansion) comes first; stage 3 stops there.
	wantErr(t, err, `components/a/x/Situfile:4: duplicate destination "/d/h", first used at line 3`)
}

func TestSourcesAreChecked(t *testing.T) {
	_, err := build(t, map[string]string{
		"components/a/x/Situfile": testutil.Situfile("file 0644 root:root missing /etc/a"),
	}, "h", comp("a", "x"))
	wantErr(t, err, `components/a/x/Situfile:2: source "missing" does not exist`)
}

func TestFormat(t *testing.T) {
	g := mustBuild(t, map[string]string{
		"components/a/x/Situfile": testutil.Situfile("file 2755 root:root s /usr/bin/${HOST}", "dir 0700 u:g - /d", "start a.service"),
		"components/a/x/s":        "x",
	}, "h", comp("a", "x"))
	n := g.Node(comp("a", "x"))
	if got := n.Entries[0].Format(); got != "file  2755  root:root  s  /usr/bin/h" {
		t.Errorf("Format = %q", got)
	}
	if got := n.SF.Entries[1].Format(); got != "dir  0700  u:g  -  /d" {
		t.Errorf("Format = %q", got)
	}
	if got := n.SF.Units[0].Format(); got != "start  a.service" {
		t.Errorf("Format = %q", got)
	}
}
