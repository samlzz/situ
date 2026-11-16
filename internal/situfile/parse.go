// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"slices"
	"strings"
	"unicode/utf8"
)

// ### Structs ###

type unitKey struct {
	action UnitAction
	name   string
}

type parser struct {
	sf   *Situfile
	self ModuleID
	errs Errors

	firstStmt  int // line of the first statement, 0 if none yet
	formatLine int // line of the format statement, 0 if none yet
	units      map[unitKey]int
}

// ### Helpers ###

// firstEncodingError returns the line and message of the first carriage
// return or invalid UTF-8 byte of data, or "" when there is none.
func firstEncodingError(data []byte) (line int, msg string) {
	line = 1
	for i := 0; i < len(data); {
		if data[i] == '\n' {
			line++
		}
		if data[i] == '\r' {
			return line, "carriage return is not allowed (LF line endings only)"
		}
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size == 1 {
			return line, "invalid UTF-8"
		}
		i += size
	}
	return 0, ""
}

func splitFields(line string) []string {
	fields := strings.FieldsFunc(line, func(r rune) bool { return r == ' ' || r == '\t' })
	for i, f := range fields {
		if strings.HasPrefix(f, "#") {
			return fields[:i]
		}
	}
	return fields
}

func (p *parser) count(ln int, kw string, args []string, want int) bool {
	if len(args) == want {
		return true
	}
	noun := "fields"
	if want == 1 {
		noun = "field"
	}
	p.errs.add(p.sf.Path, ln, "%s: expected %d %s after the keyword, got %d", kw, want, noun, len(args))
	return false
}

// ### Main parse functions ###

// Parse reports every error of data as Errors; path is the display path.
// Early return on encoding error: only the first error is returned.
func Parse(path string, data []byte, self ModuleID) (*Situfile, error) {
	p := &parser{
		sf:    &Situfile{Path: path},
		self:  self,
		units: map[unitKey]int{},
	}
	if line, msg := firstEncodingError(data); msg != "" {
		p.errs.add(path, line, "%s", msg)
		return nil, p.errs
	}
	stopped := false
	for i, raw := range strings.Split(string(data), "\n") {
		ln := i + 1
		fields := splitFields(raw)
		if len(fields) == 0 {
			continue
		}
		kw, args := fields[0], fields[1:]
		if p.firstStmt == 0 {
			p.firstStmt = ln
		}
		if kw == "format" {
			if p.format(ln, args) {
				stopped = true
				break
			}
			continue
		}
		p.statement(ln, kw, args)
	}
	if !stopped && p.formatLine == 0 {
		line := p.firstStmt
		if line == 0 {
			line = 1
		}
		p.errs.add(path, line, "missing format statement")
	}
	if len(p.errs) > 0 {
		p.errs.sortByLine()
		return nil, p.errs
	}
	return p.sf, nil
}

// format handles a format statement. stop is true when the version is
// unsupported: the rest of the file is not read.
func (p *parser) format(ln int, args []string) (stop bool) {
	if p.formatLine != 0 {
		p.errs.add(p.sf.Path, ln, "duplicate format statement")
		return false
	}
	p.formatLine = ln
	if p.firstStmt != ln {
		p.errs.add(p.sf.Path, p.firstStmt, "format must be the first statement")
	}
	if !p.count(ln, "format", args, 1) {
		return false
	}
	if args[0] != "1" {
		p.errs.add(p.sf.Path, ln, "unsupported format version %q (this situ reads version 1)", args[0])
		return true
	}
	return false
}

func (p *parser) statement(ln int, kw string, args []string) {
	switch kw {
	case "requires":
		if p.count(ln, kw, args, 1) {
			p.requires(ln, args[0])
		}
	case "file":
		p.entry(ln, File, args)
	case "config":
		p.entry(ln, Config, args)
	case "dir":
		p.entry(ln, Dir, args)
	case "data":
		p.entry(ln, Data, args)
	case "enable":
		if p.count(ln, kw, args, 1) {
			p.unit(ln, Enable, args[0])
		}
	case "start":
		if p.count(ln, kw, args, 1) {
			p.unit(ln, Start, args[0])
		}
	default:
		p.errs.add(p.sf.Path, ln, "unknown keyword %q", kw)
	}
}

func (p *parser) entry(ln int, typ EntryType, args []string) {
	if !p.count(ln, typ.String(), args, 4) {
		return
	}
	path := p.sf.Path
	ok := true
	e := Entry{Line: ln, Type: typ, Dest: args[3]}

	if m, valid := parseMode(args[0]); valid {
		e.Mode = m
	} else {
		p.errs.add(path, ln, "invalid mode %q (want four octal digits)", args[0])
		ok = false
	}

	if o, valid := parseOwner(args[1]); valid {
		e.Owner = o
	} else {
		p.errs.add(path, ln, "invalid owner %q (want <user>:<group>, each a name or a decimal ID)", args[1])
		ok = false
	}

	if msg := checkSourceField(typ, args[2]); msg != "" {
		p.errs.add(path, ln, "%s", msg)
		ok = false
	} else if typ == File || typ == Config {
		e.Source = args[2]
	}

	if msg := checkVars(args[3], func(name string) bool { return slices.Contains(format1Vars, name) }); msg != "" {
		p.errs.add(path, ln, "%s", msg)
		ok = false
	} else if msg := checkDest(args[3]); msg != "" {
		p.errs.add(path, ln, "%s", msg)
		ok = false
	}

	if ok {
		p.sf.Entries = append(p.sf.Entries, e)
	}
}

func (p *parser) unit(ln int, action UnitAction, name string) {
	if !validUnit(name) {
		p.errs.add(p.sf.Path, ln, "invalid unit name %q", name)
		return
	}
	key := unitKey{action, name}
	if first, dup := p.units[key]; dup {
		p.errs.add(p.sf.Path, ln, "unit %q listed twice under %s (first at line %d)", name, action, first)
		return
	}
	p.units[key] = ln
	p.sf.Units = append(p.sf.Units, Unit{Line: ln, Action: action, Name: name})
}

func (p *parser) requires(ln int, target string) {
	path := p.sf.Path
	id, ok := parseTarget(target)
	if !ok {
		p.errs.add(path, ln, "invalid requires target %q (want <component> or <category>/<component>)", target)
		return
	}
	if id.IsProfile() {
		p.errs.add(path, ln, "invalid requires target %q: a host profile cannot be a target", target)
		return
	}
	if !p.self.IsProfile() && (id == p.self || id.Category == "" && id.Component == p.self.Component) {
		p.errs.add(path, ln, "%q requires itself", target)
		return
	}
	// A short and a full form of the same component are always a duplicate
	// or ambiguous; two full forms with different categories are distinct.
	for _, prev := range p.sf.Requires {
		if prev.ModuleID == id {
			p.errs.add(path, ln, "target %q listed twice (first at line %d)", target, prev.Line)
			return
		}
		if prev.Component == id.Component && (prev.Category == "" || id.Category == "") {
			p.errs.add(path, ln, "target %q duplicates or is ambiguous with %q (line %d)", target, prev.String(), prev.Line)
			return
		}
	}
	p.sf.Requires = append(p.sf.Requires, Require{Line: ln, ModuleID: id})
}
