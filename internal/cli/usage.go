// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"
)

const synopsis = `usage: situ [-C <dir>] [--root <dir>] <command> [options] [targets...]
       situ --help | --version`

const shortUsage = synopsis + "\nRun \"situ --help\" for the list of commands.\n"

// topUsage is the text of situ help: synopsis, commands, topics.
func topUsage(cmds []command, topics []string) string {
	var b strings.Builder
	b.WriteString(synopsis + "\n\nCommands:\n")
	tw := tabwriter.NewWriter(&b, 0, 0, 3, ' ', 0)
	for _, c := range cmds {
		fmt.Fprintf(tw, "  %s\t%s\n", strings.TrimSpace(c.name+" "+c.synopsis), c.summary)
	}
	tw.Flush()
	b.WriteString("\nGlobal options (before the command):\n")
	tw = tabwriter.NewWriter(&b, 0, 0, 3, ' ', 0)
	for _, o := range [][2]string{
		{"-C <dir>", "Use the repository at <dir>"},
		{"--root <dir>", "Resolve every host path under <dir> (default /)"},
		{"--help", "Print this text"},
		{"--version", "Print the version"},
	} {
		fmt.Fprintf(tw, "  %s\t%s\n", o[0], o[1])
	}
	tw.Flush()
	b.WriteString("\nRun \"situ <command> --help\" for the options of a command.\n\nHelp topics (situ help <topic>):\n")
	for _, t := range topics {
		fmt.Fprintf(&b, "  %s\n", t)
	}
	return b.String()
}
