// SPDX-License-Identifier: GPL-3.0-or-later

// Package situfile parses Situfiles (docs/format.md).
package situfile

import "io/fs"

// ### Enums ###

type EntryType int

const (
	File EntryType = iota + 1
	Config
	Dir
	Data
)

func (t EntryType) String() string {
	switch t {
	case File:
		return "file"
	case Config:
		return "config"
	case Dir:
		return "dir"
	case Data:
		return "data"
	}
	return "invalid"
}

type UnitAction int

const (
	Enable UnitAction = iota + 1
	Start
)

func (a UnitAction) String() string {
	switch a {
	case Enable:
		return "enable"
	case Start:
		return "start"
	}
	return "invalid"
}

// ### Structs ###

type Situfile struct {
	Path       string
	Requires   []Require
	Entries    []Entry
	Units      []Unit
}

type ModuleID struct {
	Category  string
	Component string
}

func (m ModuleID) String() string {
	if m.Category == "" {
		return m.Component
	}
	return m.Category + "/" + m.Component
}

// Require holds target as written, short form included;
// resolution needs the whole repository.
type Require struct {
	Line int
	ModuleID
}

type Owner struct {
	User  string
	Group string
}

// Entry is a file, config, dir or data statement.
// Source is empty for dir and data. Dest is not expanded yet.
type Entry struct {
	Line   int
	Type   EntryType
	Mode   fs.FileMode
	Owner  Owner
	Source string
	Dest   string
}

type Unit struct {
	Line   int
	Name   string
	Action UnitAction
}
