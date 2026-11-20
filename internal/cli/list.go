// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/samlzz/situ/internal/exitcode"
	"github.com/samlzz/situ/internal/graph"
	"github.com/samlzz/situ/internal/repo"
	"github.com/samlzz/situ/internal/situfile"
)

// runList implements situ list. It reads the repository
// only: never the host, never --root.
func (a *App) runList(req request) error {
	cwd := ""
	if a.Opts.Dir == "" {
		var err error
		if cwd, err = a.Getwd(); err != nil {
			return exitcode.Wrap(exitcode.System, err)
		}
	}
	r, err := repo.Find(a.Opts.Dir, cwd)
	if err != nil {
		return err
	}
	switch {
	case len(req.targets) == 0:
		return a.listProfile(r)
	case req.targets[0][0] == '/':
		return a.listPath(r, req.targets[0])
	}
	return a.listModule(r, req.targets[0])
}

// currentProfile returns the profile of this host. ok is false when there is none.
func (a *App) currentProfile(r *repo.Repo) (id situfile.ModuleID, hostname string, ok bool, err error) {
	hostname, err = a.Hostname()
	if err != nil {
		return id, "", false, exitcode.Wrap(exitcode.System, err)
	}
	id, ok, err = r.CurrentProfile(hostname)
	return id, hostname, ok, err
}

// listProfile lists the graph of the current profile: tree, then one block per module.
func (a *App) listProfile(r *repo.Repo) error {
	id, hostname, ok, err := a.currentProfile(r)
	if err != nil {
		return err
	}
	if !ok {
		return exitcode.New(exitcode.Usage, "no host profile for host %q (use situ list hosts/<host>)", hostname)
	}
	return a.writeProfile(r, id)
}

// writeProfile prints the listing of profile id: tree, then one block per module.
func (a *App) writeProfile(r *repo.Repo, id situfile.ModuleID) error {
	g, err := graph.Build(r, []situfile.ModuleID{id}, id.Component)
	if err != nil {
		return err
	}
	var b strings.Builder
	writeTree(&b, g, id, 0, map[situfile.ModuleID]bool{})
	for _, n := range g.Order {
		b.WriteString("\n")
		writeBlock(&b, n)
	}
	_, err = io.WriteString(a.Stdout, b.String())
	return err
}

// listModule lists a host profile in full, or one component, expanded for the
// current profile, or not expanded without profile.
func (a *App) listModule(r *repo.Repo, name string) error {
	id, err := r.ResolveName(name)
	if err != nil {
		return err
	}
	if id.IsProfile() {
		return a.writeProfile(r, id)
	}
	host := ""
	p, _, ok, err := a.currentProfile(r)
	if err != nil {
		return err
	}
	if ok {
		host = p.Component
	}
	g, err := graph.Build(r, []situfile.ModuleID{id}, host)
	if err != nil {
		return err
	}
	var b strings.Builder
	writeBlock(&b, g.Node(id))
	_, err = io.WriteString(a.Stdout, b.String())
	return err
}

type key struct {
	file string
	line int
}
type match struct {
	src      key
	text     string
	profiles []string
}

// listPath prints every Situfile line whose destination, expanded for a
// profile, is path (grep -n style).
func (a *App) listPath(r *repo.Repo, path string) error {
	profiles, err := r.Profiles()
	if err != nil {
		return err
	}
	found := map[key]*match{}
	for _, p := range profiles {
		g, err := graph.Build(r, []situfile.ModuleID{p}, p.Component)
		if err != nil {
			return err
		}
		for _, n := range g.Order {
			for i, e := range n.Entries {
				if e.Dest != path {
					continue
				}
				k := key{n.SF.Path, e.Line}
				if found[k] == nil {
					// ? Entries and SF.Entries have the same order: print the line as written.
					found[k] = &match{src: k, text: n.SF.Entries[i].Format()}
				}
				found[k].profiles = append(found[k].profiles, repo.Rel(p))
			}
		}
	}
	if len(found) == 0 {
		return exitcode.New(exitcode.Drift, "no entry has destination %s", path)
	}
	matches := make([]*match, 0, len(found))
	for _, m := range found {
		slices.Sort(m.profiles)
		matches = append(matches, m)
	}
	slices.SortFunc(matches, func(x, y *match) int {
		return cmp.Or(cmp.Compare(x.src.file, y.src.file), cmp.Compare(x.src.line, y.src.line))
	})
	var b strings.Builder
	for _, m := range matches {
		fmt.Fprintf(&b, "%s:%d: %s  (%s)\n", m.src.file, m.src.line, m.text, strings.Join(m.profiles, ", "))
	}
	_, err = io.WriteString(a.Stdout, b.String())
	return err
}

// writeTree writes the dependency tree under id. A module already written
// is shown again with "(above)", without its dependencies.
func writeTree(b *strings.Builder, g *graph.Graph, id situfile.ModuleID, depth int, printed map[situfile.ModuleID]bool) {
	indent := strings.Repeat("  ", depth)
	if printed[id] {
		fmt.Fprintf(b, "%s%s (above)\n", indent, repo.Rel(id))
		return
	}
	printed[id] = true
	fmt.Fprintf(b, "%s%s\n", indent, repo.Rel(id))
	deps := slices.Clone(g.Node(id).Deps)
	slices.SortFunc(deps, func(x, y situfile.ModuleID) int { return cmp.Compare(repo.Rel(x), repo.Rel(y)) })
	for _, d := range deps {
		writeTree(b, g, d, depth+1, printed)
	}
}

// writeBlock writes a module: its name, then requires, entries and units.
func writeBlock(b *strings.Builder, n *graph.Node) {
	fmt.Fprintf(b, "%s\n", repo.Rel(n.ID))
	for _, d := range n.Deps {
		fmt.Fprintf(b, "  requires  %s\n", d)
	}
	for _, e := range n.Entries {
		fmt.Fprintf(b, "  %s\n", e.Format())
	}
	for _, u := range n.SF.Units {
		fmt.Fprintf(b, "  %s\n", u.Format())
	}
}
