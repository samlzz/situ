// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"fmt"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/samlzz/situ/internal/exitcode"
)

type option struct {
	names []string
	key   string
	desc  string
	// needsTarget makes the option an error without a target.
	needsTarget bool
}

type command struct {
	name     string
	synopsis string
	summary  string
	desc     string
	options  []option
	min, max int // max is -1 for no limit
	paths    bool
	run      func(a *App, req request) error
}

func (c command) option(arg string) (option, bool) {
	i := slices.IndexFunc(c.options, func(o option) bool { return slices.Contains(o.names, arg) })
	if i < 0 {
		return option{}, false
	}
	return c.options[i], true
}

func (c command) usage() string {
	var b strings.Builder
	fmt.Fprintf(&b, "usage: %s\n\n%s\n", c.line(), c.desc)
	if len(c.options) > 0 {
		b.WriteString("\nOptions:\n")
		tw := tabwriter.NewWriter(&b, 0, 0, 3, ' ', 0)
		for _, o := range c.options {
			fmt.Fprintf(tw, "  %s\t%s\n", strings.Join(o.names, ", "), o.desc)
		}
		tw.Flush()
	}
	return b.String()
}

// line is the usage line of the command.
func (c command) line() string {
	return strings.TrimSpace("situ [global options] " + c.name + " " + c.synopsis)
}

// single description of the commands: parser, help texts and dispatch are all derived from it.
func commands() []command {
	return []command{
		{
			name:     "install",
			synopsis: "[--force] [<module>...]",
			summary:  "Deploy modules and their dependencies",
			desc: `Deploy the given modules and their dependencies. With no module, deploy
the module of the current host profile.`,
			options: []option{{
				names: []string{"--force"}, key: "force", needsTarget: true,
				desc: "Overwrite a config file edited on the host (named modules only)",
			}},
			max: -1,
			run: notImplemented,
		},
		{
			name:     "check",
			synopsis: "[<module>...]",
			summary:  "Compare repository, host and state",
			desc: `Compare repository, host and recorded state without changing anything.
With no module, check the module of the current host profile. Exits 1 on drift or conflict.`,
			max: -1,
			run: notImplemented,
		},
		{
			name:     "adopt",
			synopsis: "<path|module>...",
			summary:  "Record existing host files as installed",
			desc: `Record files already present on the host as installed, when they match
the repository. A target starting with / is a path.`,
			min: 1, max: -1, paths: true,
			run: notImplemented,
		},
		{
			name:     "remove",
			synopsis: "[-r] [--purge] <module>...",
			summary:  "Remove what a module installed",
			desc:     "Remove what the given modules installed.",
			options: []option{
				{names: []string{"-r", "--recursive"}, key: "recursive", desc: "Also remove the modules required only by these ones"},
				{names: []string{"--purge"}, key: "purge", desc: "Also remove config files edited on the host"},
			},
			min: 1, max: -1,
			run: notImplemented,
		},
		{
			name:     "list",
			synopsis: "[<module>|<path>]",
			summary:  "List modules, or find the owner of a path",
			desc: `Without argument, list the modules. With a module, list its entries.
With a path (starting with /), show which module entry owns it.`,
			max: 1, paths: true,
			run: notImplemented,
		},
		{
			name:    "version",
			summary: "Print the version",
			desc:    "Print the version. Same as situ --version.",
			run:     (*App).runVersion,
		},
		{
			name:     "help",
			synopsis: "[<topic>]",
			summary:  "Print this text, or a help topic",
			desc: `Without argument, print the commands and help topics. With a topic,
print that topic.`,
			max: 1,
			run: (*App).runHelp,
		},
	}
}

func lookup(name string) (command, bool) {
	i := slices.IndexFunc(commands(), func(c command) bool { return c.name == name })
	if i < 0 {
		return command{}, false
	}
	return commands()[i], true
}

func notImplemented(_ *App, req request) error {
	return exitcode.New(exitcode.System, "%s: not implemented", req.cmd.name)
}
