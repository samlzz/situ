// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/samlzz/situ/internal/exitcode"
	"github.com/samlzz/situ/internal/situfile"
)

// namePattern describes a valid module, category or host name in messages.
const namePattern = "[a-z0-9][a-z0-9-]*"

type scan struct {
	done bool
	ids  []situfile.ModuleID
	err  error
}

// ### Helpers ###

// subdirs lists the directories of dir, sorted. Other entries are ignored.
func subdirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, exitcode.Wrap(exitcode.System, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func checkSituFile(dir, rel string) error {
	fi, err := os.Stat(filepath.Join(dir, SituFileName))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return exitcode.New(exitcode.Usage, "%s: no %s", rel, SituFileName)
	case err != nil:
		return exitcode.Wrap(exitcode.System, err)
	case !fi.Mode().IsRegular():
		return exitcode.New(exitcode.Usage, "%s/%s is not a regular file", rel, SituFileName)
	}
	return nil
}

// checkModule checks the name of the module at dir and that it holds a
// Situfile. rel is the display path of dir and kind the word used in the
// name error ("component" or "host").
func checkModule(dir, rel, kind string) error {
	if !situfile.ValidName(path.Base(rel)) {
		return exitcode.New(exitcode.Usage, "%s: invalid %s name (want %s)", rel, kind, namePattern)
	}
	return checkSituFile(dir, rel)
}

// joinErrs merges the errors of a scan into one error of code 2, or the first
// system error.
func joinErrs(errs []error) error {
	for _, e := range errs {
		if exitcode.CodeOf(e) != exitcode.Usage {
			return e
		}
	}
	if len(errs) == 0 {
		return nil
	}
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	return exitcode.New(exitcode.Usage, "%s", strings.Join(msgs, "\n"))
}

// ### Main scan functions ###

func (r *Repo) scanComponents() ([]situfile.ModuleID, error) {
	root := filepath.Join(r.Dir, situfile.ComponentsDir)
	cats, err := subdirs(root)
	if err != nil {
		return nil, err
	}
	var ids []situfile.ModuleID
	var errs []error
	for _, cat := range cats {
		rel := situfile.ComponentsDir + "/" + cat
		switch {
		case cat == situfile.HostsDir:
			errs = append(errs, exitcode.New(exitcode.Usage, "%s: %q is reserved and cannot be a category", rel, cat))
			continue
		case !situfile.ValidName(cat):
			errs = append(errs, exitcode.New(exitcode.Usage, "%s: invalid category name (want %s)", rel, namePattern))
			continue
		}
		comps, err := subdirs(filepath.Join(root, cat))
		if err != nil {
			return nil, err
		}
		for _, c := range comps {
			crel := rel + "/" + c
			if err := checkModule(filepath.Join(root, cat, c), crel, "component"); err != nil {
				errs = append(errs, err)
				continue
			}
			ids = append(ids, situfile.ModuleID{Category: cat, Component: c})
		}
	}
	return ids, joinErrs(errs)
}

func (r *Repo) scanProfiles() ([]situfile.ModuleID, error) {
	root := filepath.Join(r.Dir, situfile.HostsDir)
	names, err := subdirs(root)
	if err != nil {
		return nil, err
	}
	var ids []situfile.ModuleID
	var errs []error
	for _, h := range names {
		rel := situfile.HostsDir + "/" + h
		if err := checkModule(filepath.Join(root, h), rel, "host"); err != nil {
			errs = append(errs, err)
			continue
		}
		ids = append(ids, situfile.ModuleID{Category: situfile.HostsDir, Component: h})
	}
	return ids, joinErrs(errs)
}

// Components returns every component, sorted by category then name.
// A directory without Situfile, an invalid name or a category named
// hosts is an error (code 2).
func (r *Repo) Components() ([]situfile.ModuleID, error) {
	if !r.componentScan.done {
		r.componentScan.ids, r.componentScan.err = r.scanComponents()
		r.componentScan.done = true
	}
	return r.componentScan.ids, r.componentScan.err
}

// Profiles returns every host profile, sorted by name.
func (r *Repo) Profiles() ([]situfile.ModuleID, error) {
	if !r.profileScan.done {
		r.profileScan.ids, r.profileScan.err = r.scanProfiles()
		r.profileScan.done = true
	}
	return r.profileScan.ids, r.profileScan.err
}
