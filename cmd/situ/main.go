// SPDX-License-Identifier: GPL-3.0-or-later

// Command situ deploys files and systemd units described in Situfiles.
package main

import (
	"os"

	"github.com/samlzz/situ/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
