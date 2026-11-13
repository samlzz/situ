// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"cmp"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

var unitSuffixes = [...]string{
	".service", ".socket", ".timer", ".target", ".path",
	".mount", ".automount", ".swap", ".slice",
}

type parser struct {
	sf   *Situfile
	self ModuleID
	errs Errors

	sawStmt      bool
	firstLine    int
	firstIsFmt   bool
	formatCount  int
	dests        map[string]int
	units        map[string]int
	fullRequires map[string]int
	shortReqs    map[string]int
}

// Parse reports every error of data as Errors; path is the display path.
func Parse(path string, data []byte, self ModuleID) (*Situfile, error) {
	p := &parser{
		sf:           &Situfile{Path: path},
		self:         self,
		dests:        map[string]int{},
		units:        map[string]int{},
		fullRequires: map[string]int{},
		shortReqs:    map[string]int{},
	}
	stopped := false
	for i, raw := range strings.Split(string(data), "\n") {
		ln := i + 1
		if strings.ContainsRune(raw, '\r') {
			p.errf(ln, "carriage return is not allowed (LF line endings only)")
			continue
		}
		if !utf8.ValidString(raw) {
			p.errf(ln, "invalid UTF-8")
			continue
		}
		fields := splitFields(raw)
		if len(fields) == 0 {
			continue
		}
		kw, args := fields[0], fields[1:]
		if !p.sawStmt {
			p.sawStmt = true
			p.firstLine = ln
			p.firstIsFmt = kw == "format"
		}
		if kw == "format" {
			if !p.format(ln, args) {
				stopped = true
				break
			}
			continue
		}
		p.statement(ln, kw, args)
	}
	if !stopped && p.formatCount == 0 {
		line := 1
		if p.sawStmt {
			line = p.firstLine
		}
		p.errf(line, "missing format statement")
	}
	if len(p.errs) > 0 {
		slices.SortStableFunc(p.errs, func(a, b LineError) int { return cmp.Compare(a.Line, b.Line) })
		return nil, p.errs
	}
	return p.sf, nil
}

func (p *parser) errf(line int, format string, a ...any) {
	p.errs = append(p.errs, LineError{Path: p.sf.Path, Line: line, Msg: fmt.Sprintf(format, a...)})
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

func (p *parser) format(ln int, args []string) bool {
	p.formatCount++
	if p.formatCount > 1 {
		p.errf(ln, "duplicate format statement")
		return true
	}
	if !p.firstIsFmt {
		p.errf(p.firstLine, "format must be the first statement")
	}
	if len(args) != 1 {
		p.errf(ln, "format: expected 1 field, got %d", len(args))
		return true
	}
	if args[0] != "1" {
		p.errf(ln, "unsupported format version %q (this situ reads version 1)", args[0])
		return false
	}
	return true
}

func (p *parser) statement(ln int, kw string, args []string) {
	switch kw {
	case "requires":
		if p.count(ln, kw, args, 1) {
			p.requires(ln, args[0])
		}
	case "file", "config", "dir", "data":
		if p.count(ln, kw, args, 4) {
			p.entry(ln, kw, args)
		}
	case "enable", "start":
		if p.count(ln, kw, args, 1) {
			p.unit(ln, kw, args[0])
		}
	default:
		p.errf(ln, "unknown keyword %q", kw)
	}
}

func (p *parser) count(ln int, kw string, args []string, want int) bool {
	if len(args) == want {
		return true
	}
	p.errf(ln, "%s: expected %d fields after the keyword, got %d", kw, want, len(args))
	return false
}

func (p *parser) entry(ln int, kw string, args []string) {
	typ := map[string]EntryType{"file": File, "config": Config, "dir": Dir, "data": Data}[kw]
	ok := true
	e := Entry{Line: ln, Type: typ, Dest: args[3]}

	if validMode(args[0]) {
		m, _ := strconv.ParseUint(args[0], 8, 32)
		e.Mode = fs.FileMode(m & 0o777)
		if m&0o4000 != 0 {
			e.Mode |= fs.ModeSetuid
		}
		if m&0o2000 != 0 {
			e.Mode |= fs.ModeSetgid
		}
		if m&0o1000 != 0 {
			e.Mode |= fs.ModeSticky
		}
	} else {
		p.errf(ln, "invalid mode %q (want four octal digits)", args[0])
		ok = false
	}

	if user, group, found := strings.Cut(args[1], ":"); found && validIDPart(user) && validIDPart(group) {
		e.Owner = Owner{User: user, Group: group}
	} else {
		p.errf(ln, "invalid owner %q (want <user>:<group>, each a name or a decimal ID)", args[1])
		ok = false
	}

	if typ == Dir || typ == Data {
		if args[2] != "-" {
			p.errf(ln, "%s: source must be \"-\", got %q", kw, args[2])
			ok = false
		}
	} else if msg := checkSource(args[2]); msg != "" {
		p.errf(ln, "%s", msg)
		ok = false
	} else {
		e.Source = args[2]
	}

	if msg := checkVars(args[3], func(name string) bool { return name == "HOST" }); msg != "" {
		p.errf(ln, "%s", msg)
		ok = false
	} else if msg := checkDest(args[3]); msg != "" {
		p.errf(ln, "%s", msg)
		ok = false
	}

	if first, dup := p.dests[args[3]]; dup {
		p.errf(ln, "duplicate destination %q (first used at line %d)", args[3], first)
		ok = false
	} else {
		p.dests[args[3]] = ln
	}
	if ok {
		p.sf.Entries = append(p.sf.Entries, e)
	}
}

func (p *parser) unit(ln int, kw, name string) {
	action := Enable
	if kw == "start" {
		action = Start
	}
	if !validUnit(name) {
		p.errf(ln, "invalid unit name %q", name)
		return
	}
	key := kw + " " + name
	if first, dup := p.units[key]; dup {
		p.errf(ln, "unit %q listed twice under %s (first at line %d)", name, kw, first)
		return
	}
	p.units[key] = ln
	p.sf.Units = append(p.sf.Units, Unit{Line: ln, Action: action, Name: name})
}

func (p *parser) requires(ln int, target string) {
	parts := strings.Split(target, "/")
	for _, part := range parts {
		if !validName(part) {
			p.errf(ln, "invalid requires target %q (want <component> or <category>/<component>)", target)
			return
		}
	}
	var r Require
	switch len(parts) {
	case 1:
		r = Require{Line: ln, ModuleID: ModuleID{Component: parts[0]}}
	case 2:
		if parts[0] == "hosts" {
			p.errf(ln, "invalid requires target %q: a host profile cannot be a target", target)
			return
		}
		r = Require{Line: ln, ModuleID: ModuleID{Category: parts[0], Component: parts[1]}}
	default:
		p.errf(ln, "invalid requires target %q (want <component> or <category>/<component>)", target)
		return
	}
	if r.Component == p.self.Component && p.self.Component != "" && (r.Category == "" || r.Category == p.self.Category) {
		p.errf(ln, "%q requires itself", target)
		return
	}
	// A short and a full form of the same name are always a duplicate or ambiguous.
	if r.Category == "" {
		if first, dup := p.shortReqs[r.Component]; dup {
			p.errf(ln, "target %q listed twice (first at line %d)", target, first)
			return
		}
		for full, first := range p.fullRequires {
			if strings.HasSuffix(full, "/"+r.Component) {
				p.errf(ln, "target %q duplicates or is ambiguous with %q (line %d)", target, full, first)
				return
			}
		}
		p.shortReqs[r.Component] = ln
	} else {
		if first, dup := p.fullRequires[target]; dup {
			p.errf(ln, "target %q listed twice (first at line %d)", target, first)
			return
		}
		if first, dup := p.shortReqs[r.Component]; dup {
			p.errf(ln, "target %q duplicates or is ambiguous with %q (line %d)", target, r.Component, first)
			return
		}
		p.fullRequires[target] = ln
	}
	p.sf.Requires = append(p.sf.Requires, r)
}

func validName(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || i > 0 && c == '-') {
			return false
		}
	}
	return true
}

func validMode(s string) bool {
	if len(s) != 4 {
		return false
	}
	for i := range 4 {
		if s[i] < '0' || s[i] > '7' {
			return false
		}
	}
	return true
}

func isDigits(s string) bool {
	return s != "" && !strings.ContainsFunc(s, func(r rune) bool { return r < '0' || r > '9' })
}

// validIDPart reports whether s is a user or group name or a decimal ID
// that fits in 32 bits.
func validIDPart(s string) bool {
	if isDigits(s) {
		_, err := strconv.ParseUint(s, 10, 32)
		return err == nil
	}
	name := strings.TrimSuffix(s, "$")
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		first := c >= 'a' && c <= 'z' || c == '_'
		if !(first || i > 0 && (c >= '0' && c <= '9' || c == '-')) {
			return false
		}
	}
	return true
}

func validUnit(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' ||
			strings.IndexByte(":_.@\\-", c) >= 0) {
			return false
		}
	}
	return slices.ContainsFunc(unitSuffixes[:], func(suf string) bool {
		return len(s) > len(suf) && strings.HasSuffix(s, suf)
	})
}

// checkComponents returns a message when the slash-separated components of
// rel (a path without its leading slash) are not all plain names.
func checkComponents(what, full, rel string) string {
	if strings.HasSuffix(rel, "/") {
		return fmt.Sprintf("%s %q must not end with \"/\"", what, full)
	}
	for c := range strings.SplitSeq(rel, "/") {
		switch c {
		case "":
			return fmt.Sprintf("%s %q must not contain empty components", what, full)
		case ".", "..":
			return fmt.Sprintf("%s %q must not contain %q components", what, full, c)
		}
	}
	return ""
}

func checkSource(s string) string {
	if strings.HasPrefix(s, "/") {
		return fmt.Sprintf("source %q must be relative", s)
	}
	return checkComponents("source", s, s)
}

func checkDest(s string) string {
	if !strings.HasPrefix(s, "/") {
		return fmt.Sprintf("destination %q must be absolute", s)
	}
	if s == "/" {
		return "destination must not be \"/\""
	}
	return checkComponents("destination", s, s[1:])
}

// scanVars walks the ${NAME} references of s. For each valid one it calls
// ref(name) and, if that returns a message, stops with it. A '$' that does
// not start a valid reference is an error.
func scanVars(s string, ref func(name string) string) string {
	for i := 0; i < len(s); i++ {
		if s[i] != '$' {
			continue
		}
		bad := func() string {
			return fmt.Sprintf("invalid variable syntax in %q: \"$\" must start ${NAME}", s)
		}
		if i+2 >= len(s) || s[i+1] != '{' {
			return bad()
		}
		end := strings.IndexByte(s[i+2:], '}')
		if end < 0 {
			return bad()
		}
		name := s[i+2 : i+2+end]
		if !validVarName(name) {
			return bad()
		}
		if msg := ref(name); msg != "" {
			return msg
		}
		i += 2 + end
	}
	return ""
}

func validVarName(s string) bool {
	if s == "" || s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}

// checkVars checks the variable syntax of s and that every name is defined.
func checkVars(s string, defined func(name string) bool) string {
	return scanVars(s, func(name string) string {
		if !defined(name) {
			return fmt.Sprintf("undefined variable ${%s}", name)
		}
		return ""
	})
}
