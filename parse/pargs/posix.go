// Package pargs implements POSIX program argument syntax conventions.
//
// See https://www.gnu.org/software/libc/manual/html_node/Argument-Syntax.html
package pargs

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/gobwas/flagutil"
	"github.com/gobwas/flagutil/parse"
)

type Parser struct {
	Args []string

	// Shorthand specifies whether parser should try to provide shorthand
	// version (e.g. just first letter of name) of each top level flag.
	Shorthand bool

	// ShorthandFunc allows user to define custom way of picking shorthand
	// version of flag with given name.
	// Shorthand field must be true when setting ShorthandFunc.
	// Returning empty string means that no shorthand is possible for given
	// name.
	ShorthandFunc func(string) string

	pos   int
	err   error
	mult  bool
	name  string
	value string
	fs    parse.FlagSet
	alias map[string]string
}

func (p *Parser) Parse(_ context.Context, fs parse.FlagSet) (err error) {
	p.reset(fs)

	for p.next() {
		p.pairs(func(name, value string) bool {
			name = p.resolve(name)

			_, isHelp := lookup(fs, name)
			if isHelp {
				err = flag.ErrHelp
				return false
			}

			err = fs.Set(name, value)

			return err == nil
		})
		if err != nil {
			return err
		}
	}
	return p.err
}

func (p *Parser) NonOptionArgs() []string {
	if p.pos < len(p.Args) {
		return p.Args[p.pos:]
	}
	return nil
}

func (p *Parser) resolve(name string) string {
	if s, has := p.alias[name]; has {
		name = s
	}
	return name
}

func (p *Parser) Name(_ context.Context, fs parse.FlagSet) (func(*flag.Flag, func(string)), error) {
	short := p.shorthands(fs)
	return func(f *flag.Flag, it func(string)) {
		if p.Shorthand {
			s := p.shorthand(f)
			if _, has := short[s]; has {
				it("-" + s)
			}
		}
		var prefix string
		if len(f.Name) == 1 {
			prefix = "-"
		} else {
			prefix = "--"
		}
		it(prefix + f.Name)
	}, nil
}

func (p *Parser) shorthand(f *flag.Flag) string {
	if fn := p.ShorthandFunc; fn != nil {
		return fn(f.Name)
	}
	if !isTopSet(f) {
		// Not a topmost flag set.
		return ""
	}
	return string(f.Name[0])
}

func (p *Parser) pairs(fn func(name, value string) bool) {
	if p.mult {
		for i := range p.name {
			if !fn(p.name[i:i+1], p.value) {
				return
			}
		}
		return
	}

	fn(p.name, p.value)
}

func (p *Parser) reset(fs parse.FlagSet) {
	p.pos = 0
	p.err = nil
	p.mult = false
	p.name = ""
	p.value = ""
	p.fs = fs
	if p.Shorthand {
		p.alias = p.shorthands(fs)
	}
}

func lookup(fs parse.FlagSet, name string) (f *flag.Flag, isHelp bool) {
	f = fs.Lookup(name)
	isHelp = f == nil && (name == "h" || name == "help")
	return
}

func (p *Parser) isBoolFlag(name string) bool {
	name = p.resolve(name)
	f, isHelp := lookup(p.fs, name)
	if isHelp {
		return true
	}
	if f == nil {
		return false
	}
	return isBoolFlag(f)
}

func (p *Parser) next() bool {
	if p.err != nil {
		return false
	}
	if p.pos >= len(p.Args) {
		return false
	}
	s := p.Args[p.pos]
	if len(s) < 2 || s[0] != '-' {
		return false
	}
	p.mult = false
	p.pos++
	var short bool
	if s[1] == '-' {
		if len(s) == 2 {
			// "--" terminates all options.
			return false
		}
		s = s[2:]
	} else {
		short = true
		s = s[1:]
	}
	name, value, hasValue := split(s, '=')
	if !hasValue && p.pos < len(p.Args) {
		value = p.Args[p.pos]
		if len(value) == 0 || value[0] != '-' {
			if p.isBoolFlag(name) {
				dash := "--"
				if short {
					dash = "-"
				}
				p.fail(""+
					"ambiguous boolean flag %[1]s%[2]s value: can't guess whether "+
					"the %[3]q is the flag value or the non-flag argument "+
					"(consider using `%[1]s%[2]s=%[3]s` or `%[1]s%[2]s -- %[3]s`)",
					dash, name, value,
				)
				return false
			}
			hasValue = true
			p.pos++
		}
	}
	if short {
		if hasValue && len(name) > 1 { // -abc=foo, -abc foo
			p.fail("invalid short option syntax for %q", name)
			return false
		}
		if !hasValue { // [-o, -abc] or [-ofoo]
			if !p.isBoolFlag(name[:1]) { // -ofoo
				if len(name) == 1 {
					p.fail("argument is required for option %q", name)
					return false
				}
				value = name[1:]
				name = name[:1]
			} else {
				p.mult = true
				value = "true"
			}
		}
	} else {
		if !hasValue {
			if !p.isBoolFlag(name) {
				p.fail("argument is required for option %q", name)
				return false
			}
			value = "true"
		}
	}
	if !isValidName(name, short) {
		p.fail("invalid option name: %q", name)
		return false
	}

	p.name = name
	p.value = value

	return true
}

func (p *Parser) shorthands(fs parse.FlagSet) map[string]string {
	short := make(map[string]string)
	// Need to provide all shorthand aliases to not fail on meeting some
	// shorthand version of already provided flag.
	fs.VisitAll(func(f *flag.Flag) {
		s := p.shorthand(f)
		if s == "" {
			return
		}
		if _, has := short[s]; has {
			// Mark this shorthand name as ambiguous.
			short[s] = ""
		} else {
			short[s] = f.Name
		}
	})
	for s, n := range short {
		if n == "" {
			delete(short, s)
		}
		if fs.Lookup(s) != nil {
			delete(short, s)
		}
	}
	return short
}

func (p *Parser) fail(f string, args ...interface{}) {
	p.err = fmt.Errorf("pargs: %s", fmt.Sprintf(f, args...))
}

func split(s string, sep byte) (a, b string, ok bool) {
	i := strings.IndexByte(s, sep)
	if i == -1 {
		return s, "", false
	}
	return s[:i], s[i+1:], true
}

func isValidName(s string, short bool) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !isLetter(c) && !isDigit(c) && (short || !isSpecial(c)) {
			return false
		}
	}
	return true
}

var special = [...]bool{
	'.': true,
	'_': true,
	'-': true,
}

func isSpecial(c byte) bool {
	return special[c]
}

func isDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

func isLetter(c byte) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}

func isBoolFlag(f *flag.Flag) bool {
	x, ok := f.Value.(interface {
		IsBoolFlag() bool
	})
	return ok && x.IsBoolFlag()
}

func isTopSet(f *flag.Flag) bool {
	return strings.Index(f.Name, flagutil.SetSeparator) == -1
}
