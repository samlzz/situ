// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/samlzz/situ/internal/exitcode"
	"github.com/samlzz/situ/internal/situfile"
)

var (
	ErrNotFound  = errors.New("unknown module")
	ErrAmbiguous = errors.New("ambiguous module name")
)

type profileResult struct {
	id situfile.ModuleID
	ok bool
}

func notFound(id situfile.ModuleID) error {
	return exitcode.Wrap(exitcode.Usage, fmt.Errorf("%w %q", ErrNotFound, id))
}

func (r *Repo) resolveProfile(id situfile.ModuleID) (situfile.ModuleID, error) {
	ids, err := r.Profiles()
	if err != nil {
		return id, err
	}
	if !slices.Contains(ids, id) {
		return id, notFound(id)
	}
	return id, nil
}

func (r *Repo) resolveComponent(id situfile.ModuleID) (situfile.ModuleID, error) {
	ids, err := r.Components()
	if err != nil {
		return id, err
	}
	if id.Category != "" {
		if !slices.Contains(ids, id) {
			return id, notFound(id)
		}
		return id, nil
	}
	var found []situfile.ModuleID
	for _, c := range ids {
		if c.Component == id.Component {
			found = append(found, c)
		}
	}
	switch len(found) {
	case 0:
		return id, notFound(id)
	case 1:
		return found[0], nil
	}
	names := make([]string, len(found))
	for i, f := range found {
		names[i] = f.String()
	}
	return id, exitcode.Wrap(exitcode.Usage, fmt.Errorf("%w %q: %s", ErrAmbiguous, id, strings.Join(names, ", ")))
}

// ResolveTarget resolves a module: the short form of a component is completed by a scan.
func (r *Repo) ResolveTarget(id situfile.ModuleID) (situfile.ModuleID, error) {
	if id.IsProfile() {
		return r.resolveProfile(id)
	}
	return r.resolveComponent(id)
}

// ResolveName resolves a module named on the command line:
// <category>/<component>, <component> or hosts/<host>.
func (r *Repo) ResolveName(name string) (situfile.ModuleID, error) {
	parts := strings.Split(name, "/")
	for _, p := range parts {
		if !situfile.ValidName(p) {
			return situfile.ModuleID{}, exitcode.New(exitcode.Usage, "invalid module name %q", name)
		}
	}
	var id situfile.ModuleID
	switch len(parts) {
	case 1:
		id = situfile.ModuleID{Component: parts[0]}
	case 2:
		id = situfile.ModuleID{Category: parts[0], Component: parts[1]}
	default:
		return situfile.ModuleID{}, exitcode.New(exitcode.Usage, "invalid module name %q", name)
	}
	return r.ResolveTarget(id)
}

// CurrentProfile returns the profile of hostname (cut at the first "."),
// and false when hosts/<name>/ does not exist. A profile directory without
// Situfile is an error (code 2).
func (r *Repo) CurrentProfile(hostname string) (situfile.ModuleID, bool, error) {
	name, _, _ := strings.Cut(hostname, ".")
	name = strings.ToLower(name)
	if !situfile.ValidName(name) {
		return situfile.ModuleID{}, false, nil
	}
	if p, done := r.currentCache[name]; done {
		return p.id, p.ok, nil
	}
	id := situfile.ModuleID{Category: situfile.HostsDir, Component: name}
	fi, err := os.Stat(r.ModuleDir(id))
	switch {
	case errors.Is(err, fs.ErrNotExist) || err == nil && !fi.IsDir():
		r.currentCache[name] = profileResult{}
		return situfile.ModuleID{}, false, nil
	case err != nil:
		return situfile.ModuleID{}, false, exitcode.Wrap(exitcode.System, err)
	}
	if err := checkSituFile(r.ModuleDir(id), Rel(id)); err != nil {
		return situfile.ModuleID{}, false, err
	}
	r.currentCache[name] = profileResult{id, true}
	return id, true, nil
}
