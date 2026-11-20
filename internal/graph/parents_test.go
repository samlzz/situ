// SPDX-License-Identifier: GPL-3.0-or-later

package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/samlzz/situ/internal/exitcode"
	"github.com/samlzz/situ/internal/testutil"
)

func parentsGraph(t *testing.T, lines ...string) *Graph {
	t.Helper()
	return mustBuild(t, map[string]string{
		"components/a/x/Situfile": testutil.Situfile(lines...),
		"components/a/x/s":        "x",
	}, "h", comp("a", "x"))
}

func TestParentExistsUnderRoot(t *testing.T) {
	root := testutil.Tree(t, map[string]string{"etc/": "", "top/": ""})
	g := parentsGraph(t, "file 0644 root:root s /etc/a", "dir 0755 root:root - /top")
	if err := g.CheckParents(root); err != nil {
		t.Fatal(err)
	}
}

func TestParentDeclared(t *testing.T) {
	for _, role := range []string{"dir", "data"} {
		g := parentsGraph(t,
			"file 0644 root:root s /opt/app/a",
			"dir 0755 root:root - /opt/app/sub/deep",
			role+" 0755 root:root - /opt/app/sub",
			role+" 0755 root:root - /opt/app",
			"dir 0755 root:root - /opt",
		)
		if err := g.CheckParents(t.TempDir()); err != nil {
			t.Errorf("%s: %v", role, err)
		}
	}
	// Declared by another module of the graph.
	g := mustBuild(t, map[string]string{
		"components/a/x/Situfile": testutil.Situfile("requires y", "file 0644 root:root s /srv/a"),
		"components/a/x/s":        "x",
		"components/a/y/Situfile": testutil.Situfile("data 0755 root:root - /srv"),
	}, "h", comp("a", "x"))
	if err := g.CheckParents(t.TempDir()); err != nil {
		t.Error(err)
	}
}

func TestParentUndeclaredMissing(t *testing.T) {
	g := parentsGraph(t, "dir 0755 root:root - /var/lib/app", "file 0644 root:root s /var/lib/app/a", "file 0644 root:root s /var/lib/b")
	err := g.CheckParents(t.TempDir())
	wantErr(t, err,
		`components/a/x/Situfile:2: parent directory "/var/lib" of "/var/lib/app" does not exist and is not declared by a dir or data entry`,
		`components/a/x/Situfile:4: parent directory "/var/lib" of "/var/lib/b" does not exist and is not declared by a dir or data entry`)
}

// A file is not a declaration: only dir and data are.
func TestParentDeclaredByFileIsNotEnough(t *testing.T) {
	g := parentsGraph(t, "file 0644 root:root s /x", "file 0644 root:root s /x/y")
	wantErr(t, g.CheckParents(t.TempDir()),
		`components/a/x/Situfile:3: parent directory "/x" of "/x/y" does not exist and is not declared by a dir or data entry`)
}

func TestParentIsNotADirectory(t *testing.T) {
	root := testutil.Tree(t, map[string]string{"etc/regular": "x"})
	g := parentsGraph(t, "file 0644 root:root s /etc/regular/a", "file 0644 root:root s /etc/regular/deeper/b")
	err := g.CheckParents(root)
	wantErr(t, err,
		`components/a/x/Situfile:2: parent directory "/etc/regular" of "/etc/regular/a" is not a directory`,
		`components/a/x/Situfile:3: parent directory "/etc/regular/deeper" of "/etc/regular/deeper/b" is not a directory (a component of its path is not)`)
}

func TestParentSymlinkToDirectory(t *testing.T) {
	root := testutil.Tree(t, map[string]string{"real/": "", "file": "x"})
	for name, target := range map[string]string{"link": "real", "badlink": "file", "dangling": "nowhere"} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	g := parentsGraph(t, "file 0644 root:root s /link/a")
	if err := g.CheckParents(root); err != nil {
		t.Errorf("link to a directory refused: %v", err)
	}
	// A link to a regular file or to nothing is not a directory.
	wantErr(t, parentsGraph(t, "file 0644 root:root s /badlink/a").CheckParents(root),
		`components/a/x/Situfile:2: parent directory "/badlink" of "/badlink/a" is not a directory`)
	wantErr(t, parentsGraph(t, "file 0644 root:root s /dangling/a").CheckParents(root),
		`components/a/x/Situfile:2: parent directory "/dangling" of "/dangling/a" does not exist and is not declared by a dir or data entry`)
}

func TestParentStatErrorIsSystem(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
	root := testutil.Tree(t, map[string]string{"locked/inner/": ""})
	if err := os.Chmod(filepath.Join(root, "locked"), 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(filepath.Join(root, "locked"), 0o755) })
	err := parentsGraph(t, "file 0644 root:root s /locked/inner/a").CheckParents(root)
	if exitcode.CodeOf(err) != exitcode.System {
		t.Errorf("err = %v, want code 3", err)
	}
}
