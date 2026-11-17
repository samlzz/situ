// SPDX-License-Identifier: GPL-3.0-or-later

// Package testutil holds helpers shared by the tests.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// Tree writes files (slash-separated path -> content) under a new temporary
// directory and returns it. A path ending with "/" creates a directory.
func Tree(t testing.TB, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for p, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if p[len(p)-1] == '/' {
			if err := os.MkdirAll(full, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}
