// SPDX-License-Identifier: GPL-3.0-or-later

// Package exitcode defines errors that carry an exit code.
package exitcode

import (
	"errors"
	"fmt"
)

const (
	OK     = 0
	Drift  = 1
	Usage  = 2
	System = 3
)

type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }

func (e *Error) Unwrap() error { return e.Err }

func (e *Error) ExitCode() int { return e.Code }

func New(code int, format string, a ...any) error {
	return &Error{Code: code, Err: fmt.Errorf(format, a...)}
}

func Wrap(code int, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Err: err}
}

type coder interface {
	error
	ExitCode() int
}

// CodeOf returns 0 for nil, the carried code, or System for other errors.
func CodeOf(err error) int {
	if err == nil {
		return OK
	}
	if c, ok := errors.AsType[coder](err); ok {
		return c.ExitCode()
	}
	return System
}
