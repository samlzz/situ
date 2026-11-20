// SPDX-License-Identifier: GPL-3.0-or-later

package graph

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"syscall"

	"github.com/samlzz/situ/internal/exitcode"
	"github.com/samlzz/situ/internal/situfile"
)

// CheckParents checks that the parent of every destination is declared by a
// dir or data entry of the graph, or is a directory under root (a symbolic
// link to a directory counts). The graph must be built with a host. A bad
// parent is code 2, any other stat error code 3.
func (g *Graph) CheckParents(root string) error {
	declared := map[string]bool{}
	for _, n := range g.Order {
		for _, e := range n.Entries {
			if e.Type == situfile.Dir || e.Type == situfile.Data {
				declared[e.Dest] = true
			}
		}
	}
	var errs situfile.Errors
	checked := map[string]string{} // parent -> problem, "" when fine
	for _, n := range g.Order {
		for _, e := range n.Entries {
			parent := path.Dir(e.Dest)
			if declared[parent] {
				continue
			}
			msg, done := checked[parent]
			if !done {
				var err error
				if msg, err = statParent(root, parent); err != nil {
					return exitcode.Wrap(exitcode.System, err)
				}
				checked[parent] = msg
			}
			if msg != "" {
				errs.Add(n.SF.Path, e.Line, "parent directory %q of %q %s", parent, e.Dest, msg)
			}
		}
	}
	return errs.Err()
}

// statParent returns a message when parent is not an existing directory
// under root and not declared, or an error when it cannot be told.
func statParent(root, parent string) (string, error) {
	fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(parent)))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "does not exist and is not declared by a dir or data entry", nil
	case errors.Is(err, syscall.ENOTDIR):
		return "is not a directory (a component of its path is not)", nil
	case err != nil:
		return "", err
	case !fi.IsDir():
		return "is not a directory", nil
	}
	return "", nil
}
