// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/samlzz/situ/internal/exitcode"
)

// CheckSources checks that file and config sources are regular files, links excluded.
func CheckSources(sf *Situfile, moduleDir string) error {
	var errs Errors
	for _, e := range sf.Entries {
		if e.Type != File && e.Type != Config {
			continue
		}
		msg, err := statSource(moduleDir, e.Source)
		if err != nil {
			return exitcode.Wrap(exitcode.System, fmt.Errorf("%s:%d: source %q: %w", sf.Path, e.Line, e.Source, err))
		}
		if msg != "" {
			errs.add(sf.Path, e.Line, "%s", msg)
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func statSource(moduleDir, source string) (string, error) {
	parts := strings.Split(source, "/")
	cur := moduleDir
	for i, c := range parts {
		cur = filepath.Join(cur, c)
		fi, err := os.Lstat(cur)
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Sprintf("source %q does not exist", source), nil
		}
		if err != nil {
			return "", err
		}
		if (fi.Mode() & fs.ModeSymlink) != 0 {
			return fmt.Sprintf("source %q: %q is a symbolic link", source, strings.Join(parts[:i+1], "/")), nil
		}
		if i < len(parts)-1 {
			if !fi.IsDir() {
				return fmt.Sprintf("source %q: %q is not a directory", source, strings.Join(parts[:i+1], "/")), nil
			}
			continue
		}
		if fi.IsDir() {
			return fmt.Sprintf("source %q is a directory, not a regular file", source), nil
		}
		if !fi.Mode().IsRegular() {
			return fmt.Sprintf("source %q is not a regular file", source), nil
		}
	}
	return "", nil
}
