package pargs

import (
	"flag"
	"fmt"
	"strings"

	"github.com/gobwas/flagutil/parse"
)

type Parser struct {
	Args []string

	pos   int
	err   error
	mult  bool
	name  string
	value string

	isBoolFlag func(string) bool
}

func (p *Parser) Parse(fs parse.FlagSet) (err error) {
	p.reset(func(name string) bool {
		return isBoolFlag(fs.Lookup(name))
	})
	for p.next() {
		p.pairs(func(name, value string) bool {
			err = fs.Set(name, value)
			return err == nil
		})
		if err != nil {
			return err
		}
	}
	return p.err
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

func (p *Parser) reset(isBoolFlag func(name string) bool) {
	p.pos = 0
	p.err = nil
	p.mult = false
	p.name = ""
	p.value = ""

	p.isBoolFlag = isBoolFlag
}

func (p *Parser) next() bool {
	if p.err != nil {
		return false
	}
	if p.pos >= len(p.Args) {
		return false
	}
	s := p.Args[p.pos]
	p.pos++
	p.mult = false
	if len(s) < 2 || s[0] != '-' {
		return false
	}
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
		if len(value) > 0 && value[0] != '-' {
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

func (p *Parser) fail(f string, args ...interface{}) {
	p.err = fmt.Errorf(f, args...)
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
