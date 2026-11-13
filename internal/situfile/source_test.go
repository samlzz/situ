// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/samlzz/situ/internal/exitcode"
)

func TestCheckSources(t *testing.T) {
	dir := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.WriteFile(filepath.Join(dir, "ok"), []byte("x"), 0o644))
	must(os.MkdirAll(filepath.Join(dir, "files"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "files", "a.conf"), []byte("x"), 0o644))
	must(os.Mkdir(filepath.Join(dir, "adir"), 0o755))
	must(os.Symlink("ok", filepath.Join(dir, "link")))
	must(os.Symlink("files", filepath.Join(dir, "dirlink")))
	must(os.Symlink("nowhere", filepath.Join(dir, "dangling")))

	body := "file 0644 root:root ok /d/1\n" + // 2 ok
		"config 0644 root:root files/a.conf /d/2\n" + // 3 ok
		"file 0644 root:root missing /d/3\n" + // 4
		"file 0644 root:root adir /d/4\n" + // 5
		"file 0644 root:root link /d/5\n" + // 6
		"file 0644 root:root dirlink/a.conf /d/6\n" + // 7
		"config 0644 root:root dangling /d/7\n" + // 8
		"file 0644 root:root ok/sub /d/8\n" + // 9
		"file 0644 root:root nodir/x /d/9\n" + // 10
		"dir 0755 root:root - /d/10\n"
	sf := parseOK(t, body)
	err := CheckSources(sf, dir)
	want := []string{
		"S:4: source \"missing\" does not exist",
		"S:5: source \"adir\" is a directory, not a regular file",
		"S:6: source \"link\": \"link\" is a symbolic link",
		"S:7: source \"dirlink/a.conf\": \"dirlink\" is a symbolic link",
		"S:8: source \"dangling\": \"dangling\" is a symbolic link",
		"S:9: source \"ok/sub\": \"ok\" is not a directory",
		"S:10: source \"nodir/x\" does not exist",
	}
	if got := errLines(t, err); !reflect.DeepEqual(got, want) {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}

	if err := CheckSources(parseOK(t, "file 0644 root:root ok /d\ndata 0700 root:root - /e\n"), dir); err != nil {
		t.Fatalf("valid sources: %v", err)
	}
}

func TestCheckSourcesIOError(t *testing.T) {
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })
	if _, err := os.Lstat(filepath.Join(locked, "x")); err == nil || os.IsNotExist(err) {
		t.Skip("directory permissions not enforced (running as root?)")
	}
	err := CheckSources(parseOK(t, "file 0644 root:root locked/x /d\n"), dir)
	if err == nil || exitcode.CodeOf(err) != exitcode.System {
		t.Fatalf("err = %v, code %d, want code 3", err, exitcode.CodeOf(err))
	}
}
