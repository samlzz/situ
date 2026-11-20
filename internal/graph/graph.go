// SPDX-License-Identifier: GPL-3.0-or-later

// Package graph builds the module graph of a run and checks the rules that
// span several modules (docs/spec.md, Resolution).
package graph

import (
	"cmp"
	"errors"
	"slices"
	"strings"

	"github.com/samlzz/situ/internal/repo"
	"github.com/samlzz/situ/internal/situfile"
)

// ModuleID is a short name for situfile.ModuleID.
type ModuleID = situfile.ModuleID

// Node is a module of the graph.
type Node struct {
	ID ModuleID
	SF *situfile.Situfile
	// Deps are the resolved requires targets, in Situfile order:
	// Deps[i] is SF.Requires[i].
	Deps []ModuleID
	// Entries are the entries of SF, in the same order, with the
	// destinations expanded. Without a host they are the entries as written.
	Entries []situfile.Entry
}

// Graph is the set of modules reachable from the roots of a run.
type Graph struct {
	HostName string
	// Order is a topological order, dependencies first.
	Order []*Node
	nodes map[ModuleID]*Node
}

// Node returns the module id, or nil when it is not in the graph.
func (g *Graph) Node(id ModuleID) *Node { return g.nodes[id] }

type origin struct {
	id   ModuleID
	line int
}

// byName orders modules: components first, then hosts, each group sorted
// lexically by repository path.
func byName(a, b ModuleID) int {
	boolToInt := func (b bool) int { if b { return 1 }; return 0}

	return cmp.Or(
		cmp.Compare(boolToInt(a.IsProfile()), boolToInt(b.IsProfile())),
		strings.Compare(repo.Rel(a), repo.Rel(b)),
	)
}

// Build returns the graph of roots. Every stage runs only when the previous
// one succeeded. All errors are reported together, sorted by path then line.
// Without a host, destinations aren't expanded (so destinations aren't checked).
func Build(r *repo.Repo, roots []ModuleID, host string) (*Graph, error) {
	g := &Graph{HostName: host, nodes: map[ModuleID]*Node{}}
	if err := g.read(r, roots); err != nil {
		return nil, err
	}
	if err := g.sort(); err != nil {
		return nil, err
	}
	if host != "" {
		if err := g.entries(r); err != nil {
			return nil, err
		}
	} else {
		for _, n := range g.Order {
			n.Entries = slices.Clone(n.SF.Entries)
		}
	}
	if err := g.units(); err != nil {
		return nil, err
	}
	return g, nil
}

// read is stage 1, plus the resolution of targets: it parses every
// reachable Situfile. A Situfile with errors is not followed. Failed
// targets are stage 2 errors, reported only when every file parsed.
func (g *Graph) read(r *repo.Repo, roots []ModuleID) error {
	var parseErrs, resErrs situfile.Errors
	queue := slices.Clone(roots)
	slices.SortFunc(queue, byName)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if g.nodes[id] != nil {
			continue
		}
		sf, err := r.Load(id)
		if err != nil {
			if err := parseErrs.Collect(err); err != nil {
				return err
			}
			g.nodes[id] = &Node{ID: id} // seen, not followed
			continue
		}
		n := &Node{ID: id, SF: sf}
		g.nodes[id] = n
		for _, req := range sf.Requires {
			dep, err := r.ResolveTarget(req.ModuleID)
			if err != nil {
				if !errors.Is(err, repo.ErrNotFound) && !errors.Is(err, repo.ErrAmbiguous) {
					return err
				}
				resErrs.Add(sf.Path, req.Line, "requires: %s", err)
				continue
			}
			n.Deps = append(n.Deps, dep)
			queue = append(queue, dep)
		}
	}
	if err := parseErrs.Err(); err != nil {
		return err
	}
	return resErrs.Err()
}

// sort is stage 2: cycles, then the topological order (Kahn). Ties are
// broken by Rel, so components come before hosts, lexically.
func (g *Graph) sort() error {
	var errs situfile.Errors
	state := map[ModuleID]int{} // 1 on the stack, 2 done
	var stack []ModuleID
	var visit func(n *Node)
	visit = func(n *Node) {
		state[n.ID] = 1
		stack = append(stack, n.ID)
		for i, dep := range n.Deps {
			switch state[dep] {
			case 0:
				visit(g.nodes[dep])
			case 1:
				cycle := append(slices.Clone(stack[slices.Index(stack, dep):]), dep)
				names := make([]string, len(cycle))
				for j, c := range cycle {
					names[j] = c.String()
				}
				errs.Add(n.SF.Path, n.SF.Requires[i].Line, "dependency cycle: %s", strings.Join(names, " -> "))
			}
		}
		stack = stack[:len(stack)-1]
		state[n.ID] = 2
	}
	ids := g.ids()
	for _, id := range ids {
		if state[id] == 0 {
			visit(g.nodes[id])
		}
	}
	if err := errs.Err(); err != nil {
		return err
	}

	waiting := map[ModuleID]int{}
	dependents := map[ModuleID][]ModuleID{}
	var ready []ModuleID
	for _, id := range ids {
		n := g.nodes[id]
		waiting[id] = len(n.Deps)
		for _, d := range n.Deps {
			dependents[d] = append(dependents[d], id)
		}
		if len(n.Deps) == 0 {
			ready = append(ready, id)
		}
	}
	for len(ready) > 0 {
		slices.SortFunc(ready, byName)
		id := ready[0]
		ready = ready[1:]
		g.Order = append(g.Order, g.nodes[id])
		for _, d := range dependents[id] {
			if waiting[d]--; waiting[d] == 0 {
				ready = append(ready, d)
			}
		}
	}
	return nil
}

func (g *Graph) ids() []ModuleID {
	ids := make([]ModuleID, 0, len(g.nodes))
	for id := range g.nodes {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, byName)
	return ids
}

// entries is stage 3: expansion for the host, sources, then destinations
// declared by two modules.
func (g *Graph) entries(r *repo.Repo) error {
	var errs situfile.Errors
	vars := map[string]string{situfile.VarHost: g.HostName}
	for _, n := range g.Order {
		entries, err := situfile.Expand(n.SF, vars)
		if err := errs.Collect(err); err != nil {
			return err
		}
		n.Entries = entries
		if err := errs.Collect(situfile.CheckSources(n.SF, r.ModuleDir(n.ID))); err != nil {
			return err
		}
	}
	if err := errs.Err(); err != nil {
		return err
	}
	seen := map[string]origin{}
	for _, n := range g.Order {
		for _, e := range n.Entries {
			o, dup := seen[e.Dest]
			switch {
			case !dup:
				seen[e.Dest] = origin{n.ID, e.Line}
			case o.id != n.ID:
				errs.Add(n.SF.Path, e.Line, "duplicate destination %q, also declared by %s at line %d", e.Dest, repo.Rel(o.id), o.line)
			}
		}
	}
	return errs.Err()
}

// units is stage 4: a unit declared by two modules. The same unit under
// enable and start of one module is allowed.
func (g *Graph) units() error {
	var errs situfile.Errors
	seen := map[string]origin{}
	for _, n := range g.Order {
		for _, u := range n.SF.Units {
			o, dup := seen[u.Name]
			switch {
			case !dup:
				seen[u.Name] = origin{n.ID, u.Line}
			case o.id != n.ID:
				errs.Add(n.SF.Path, u.Line, "duplicate unit %q, also declared by %s at line %d", u.Name, repo.Rel(o.id), o.line)
			}
		}
	}
	return errs.Err()
}
