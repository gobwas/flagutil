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

// Parser reads the arguments in the POSIX syntax. A one-letter name is a
// short option, "-s", and a longer one a long option, "--settings".
//
// Which names a flag has is the flag set's business, as flagutil.WithNameFunc
// decides it, and Shorthands gives the first-letter convention.
type Parser struct {
	Args []string

	pos   int
	err   error
	mult  bool
	name  string
	value string
	rest  string // The tail of a cluster ("-ofoo" of "-aofoo").
	fs    parse.FlagSet
}

// Shorthands returns a name function, for [flagutil.WithNameFunc], that gives
// every top level flag of fs its first letter as a shorthand beside its name:
// "-s" for "settings", unless another flag starts with the same letter or a
// flag is named by it, in which case the flag keeps its name alone.
func Shorthands(fs *flag.FlagSet) func(name string) []string {
	short := make(map[string]string)
	fs.VisitAll(func(f *flag.Flag) {
		if !isTopSet(f) || len(f.Name) < 2 {
			return
		}
		s := f.Name[:1]
		if _, has := short[s]; has {
			// Mark this shorthand as ambiguous.
			short[s] = ""
		} else {
			short[s] = f.Name
		}
	})
	for s, n := range short {
		if n == "" || fs.Lookup(s) != nil {
			delete(short, s)
		}
	}
	return func(name string) []string {
		if name != "" && short[name[:1]] == name {
			return []string{name[:1], name}
		}
		return []string{name}
	}
}

func (p *Parser) Parse(_ context.Context, fs parse.FlagSet) (err error) {
	p.reset(fs)

	for p.next() {
		p.pairs(func(name, value string) bool {
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

// Name implements flagutil.Printer: a one-letter name prints with one dash, a
// longer one with two.
func (p *Parser) Name(_ context.Context, fs parse.FlagSet) (func(*flag.Flag, func(string)), error) {
	return func(f *flag.Flag, it func(string)) {
		var prefix string
		if len(f.Name) == 1 {
			prefix = "-"
		} else {
			prefix = "--"
		}
		it(prefix + f.Name)
	}, nil
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
	p.rest = ""
	p.fs = fs
}

func lookup(fs parse.FlagSet, name string) (f *flag.Flag, isHelp bool) {
	f = fs.Lookup(name)
	isHelp = f == nil && (name == "h" || name == "help")
	return
}

func (p *Parser) isBoolFlag(name string) bool {
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
	s := p.rest
	p.rest = ""
	if s == "" {
		if p.pos >= len(p.Args) {
			return false
		}
		s = p.Args[p.pos]
		if !isOption(s) {
			return false
		}
		p.pos++
	}
	p.mult = false
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
	// Only an option that requires an argument takes the next word.
	//
	// POSIX knows two kinds of option: with an option-argument and without.
	// A flag whose IsBoolFlag() is true is an option without an
	// option-argument, so "--flag value" is "--flag" and the operand "value".
	// The "--flag=value" form for it is not POSIX but the GNU long-option
	// convention, where getopt_long() takes an optional argument only inside
	// the same word.
	//
	// A short cluster ("-abc", "-ofoo") holds its own argument, if any.
	cluster := short && len(name) > 1
	if !hasValue && !cluster && !p.isBoolFlag(name) && p.pos < len(p.Args) {
		value = p.Args[p.pos]
		if !isOption(value) {
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
				// Let Set handle undefined options, including the ignore-undefined policy.
				if len(name) == 1 && p.fs.Lookup(name) != nil {
					p.fail("argument is required for option %q", name)
					return false
				}
				value = name[1:]
				name = name[:1]
			} else { // -abc, or -aofoo as -a -ofoo
				for i := range name {
					if !p.isBoolFlag(name[i : i+1]) {
						if !isValidName(name[i:i+1], true) {
							p.fail("invalid option name: %q", name[i:i+1])
							return false
						}
						p.rest = "-" + name[i:]
						name = name[:i]
						break
					}
				}
				p.mult = true
				value = "true"
			}
		}
	} else {
		if !hasValue {
			if p.isBoolFlag(name) {
				value = "true"
			} else if p.fs.Lookup(name) != nil {
				p.fail("argument is required for option %q", name)
				return false
			}
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

func (p *Parser) fail(f string, args ...interface{}) {
	p.err = fmt.Errorf("pargs: %s", fmt.Sprintf(f, args...))
}

// isOption reports whether s looks like an option: a dash followed by at
// least one character. A lone "-" is an operand, as POSIX defines it.
func isOption(s string) bool {
	return len(s) >= 2 && s[0] == '-'
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
