// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"fmt"
	"slices"
	"strings"
)

type Options struct {
	Dir  string
	Root string
}

type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

func (e *usageError) ExitCode() int { return 2 }

func usagef(format string, a ...any) error {
	return &usageError{msg: fmt.Sprintf(format, a...)}
}

// parsed command line
type request struct {
	opts          Options
	cmd           command
	flags         map[string]bool
	targets       []string
	help, version bool // ? when are set, nothing else runs
}

func (a *App) parseArgs(args []string) (request, error) {
	var req request
	i := 0
	// global options
loop:
	for ; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-C":
			if i+1 >= len(args) {
				return req, usagef("option -C requires an argument")
			}
			i++
			req.opts.Dir = args[i]
		case a == "--root":
			if i+1 >= len(args) {
				return req, usagef("option --root requires an argument")
			}
			i++
			req.opts.Root = args[i]
		case strings.HasPrefix(a, "--root="):
			req.opts.Root = strings.TrimPrefix(a, "--root=")
		case a == "--help":
			req.help = true
			return req, nil
		case a == "--version":
			req.version = true
			return req, nil
		case strings.HasPrefix(a, "-"):
			return req, usagef("unknown option %s", a)
		default:
			break loop
		}
	}
	if i >= len(args) {
		return req, usagef("missing command")
	}
	// command
	name := args[i]
	var ok bool
	if req.cmd, ok = a.lookup(name); !ok {
		return req, usagef("unknown command %q", name)
	}
	// command options
	rest := args[i+1:]
	if slices.Contains(rest, "--help") {
		req.help = true
		return req, nil
	}
	req.flags = map[string]bool{}
	j := 0
	for ; j < len(rest) && strings.HasPrefix(rest[j], "-"); j++ {
		opt, ok := req.cmd.option(rest[j])
		if !ok {
			return req, usagef("%s: unknown option %s", name, rest[j])
		}
		req.flags[opt.key] = true
	}
	req.targets = rest[j:]
	for _, t := range req.targets {
		switch {
		case strings.HasPrefix(t, "-"):
			return req, usagef("%s: option %s after a target (options must come before targets)", name, t)
		case t == "":
			return req, usagef("%s: empty target", name)
		case t[0] == '/' && !req.cmd.paths:
			return req, usagef("%s: %s is a path", name, t)
		}
	}
	n := len(req.targets)
	switch {
	case n < req.cmd.min:
		return req, usagef("%s: expected at least %d module, got %d", name, req.cmd.min, n)
	case req.cmd.max >= 0 && n > req.cmd.max:
		return req, usagef("%s: expected at most %d argument(s), got %d", name, req.cmd.max, n)
	case n == 0:
		for _, o := range req.cmd.options {
			if o.needsTarget && req.flags[o.key] {
				return req, usagef("%s: %s requires a module", name, o.names[0])
			}
		}
	}
	return req, nil
}
