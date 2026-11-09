// SPDX-License-Identifier: GPL-3.0-or-later

// Package cli implements the situ command line.
package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/samlzz/situ/internal/exitcode"
	"github.com/samlzz/situ/internal/help"
)

type App struct {
	Stdout  io.Writer
	Stderr  io.Writer
	Version func() string
	Opts    Options
}

func Run(args []string, stdout, stderr io.Writer) int {
	a := &App{Stdout: stdout, Stderr: stderr, Version: buildVersion}
	return a.Main(args)
}

func (a *App) Main(args []string) int {
	err := a.Execute(args)
	if err == nil {
		return exitcode.OK
	}
	fmt.Fprintf(a.Stderr, "situ: %s\n", err)
	if _, ok := errors.AsType[*usageError](err); ok {
		io.WriteString(a.Stderr, shortUsage)
	}
	return exitcode.CodeOf(err)
}

func (a *App) Execute(args []string) error {
	req, err := parseArgs(args)
	if err != nil {
		return err
	}
	a.Opts = req.opts
	switch {
	case req.version:
		return a.runVersion(req)
	case req.help && req.cmd.name != "":
		_, err := io.WriteString(a.Stdout, req.cmd.usage())
		return err
	case req.help:
		return a.runHelp(request{})
	}
	return req.cmd.run(a, req)
}

func (a *App) runVersion(request) error {
	_, err := fmt.Fprintln(a.Stdout, a.Version())
	return err
}

func (a *App) runHelp(req request) error {
	if len(req.targets) == 0 {
		_, err := io.WriteString(a.Stdout, topUsage(help.Topics()))
		return err
	}
	name := req.targets[0]
	if help.IsStub(name) {
		return exitcode.New(exitcode.System, "help topic %s: not implemented", name)
	}
	if text, ok := help.Topic(name); ok {
		_, err := io.WriteString(a.Stdout, text)
		return err
	}
	if _, isCmd := lookup(name); isCmd {
		return exitcode.New(exitcode.Usage, "no help topic %q (use situ %s --help)", name, name)
	}
	return exitcode.New(exitcode.Usage, "no help topic %q (use situ help)", name)
}
