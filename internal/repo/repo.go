// SPDX-License-Identifier: GPL-3.0-or-later

// Package repo finds a situ repository and names its modules.
package repo

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/samlzz/situ/internal/exitcode"
	"github.com/samlzz/situ/internal/situfile"
)

type loadResult struct {
	sf  *situfile.Situfile
	err error
}

// Repo is a situ repository. Create it with Find or Open. The scans are
// done once, on first use.
type Repo struct {
	Dir string

	componentScan scan                             // Components()
	profileScan   scan                             // Profiles()
	loadCache     map[situfile.ModuleID]loadResult // Load()
	currentCache  map[string]profileResult         // CurrentProfile(), by short hostname
}

// Open returns the repository at dir without checking it.
func Open(dir string) *Repo {
	return &Repo{Dir: dir, loadCache: map[situfile.ModuleID]loadResult{}, currentCache: map[string]profileResult{}}
}

func isRepo(dir string) (bool, error) {
	for _, sub := range []string{situfile.ComponentsDir, situfile.HostsDir} {
		fi, err := os.Stat(filepath.Join(dir, sub))
		if errors.Is(err, fs.ErrNotExist) || err == nil && !fi.IsDir() {
			return false, nil
		}
		if err != nil {
			return false, exitcode.Wrap(exitcode.System, err)
		}
	}
	return true, nil
}

// Find returns the repository: dirOpt (-C) when set, otherwise the first
// directory from cwd upwards that holds components/ and hosts/.
func Find(dirOpt, cwd string) (*Repo, error) {
	if dirOpt != "" {
		ok, err := isRepo(dirOpt)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, exitcode.New(exitcode.Usage, "%s is not a situ repository (no %s/ and %s/)",
				dirOpt, situfile.ComponentsDir, situfile.HostsDir)
		}
		return Open(dirOpt), nil
	}
	dir, err := filepath.Abs(cwd)
	if err != nil {
		return nil, exitcode.Wrap(exitcode.System, err)
	}
	for {
		ok, err := isRepo(dir)
		if err != nil {
			return nil, err
		}
		if ok {
			return Open(dir), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, exitcode.New(exitcode.Usage, "no situ repository found (no %s/ and %s/ in %s or above; use -C <dir>)",
				situfile.ComponentsDir, situfile.HostsDir, cwd)
		}
		dir = parent
	}
}

// Load reads and parses the Situfile of id. Parse errors have code 2, an
// I/O error code 3. The result is cached and must not be modified.
func (r *Repo) Load(id situfile.ModuleID) (*situfile.Situfile, error) {
	if l, ok := r.loadCache[id]; ok {
		return l.sf, l.err
	}
	data, err := os.ReadFile(filepath.Join(r.ModuleDir(id), SituFileName))
	var l loadResult
	if err != nil {
		l.err = exitcode.Wrap(exitcode.System, err)
	} else {
		l.sf, l.err = situfile.Parse(SituFile(id), data, id)
	}
	r.loadCache[id] = l
	return l.sf, l.err
}
