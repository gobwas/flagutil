package args

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/gobwas/flagutil/parse"
)

type Parser struct {
	Args []string

	fs    parse.FlagSet
	pos   int
	name  string
	value string
	err   error
}

func (p *Parser) Parse(_ context.Context, fs parse.FlagSet) error {
	p.reset(fs)
	for p.next() {
		if fs.Lookup(p.name) == nil && (p.name == "help" || p.name == "h") {
			return flag.ErrHelp
		}
		if err := fs.Set(p.name, p.value); err != nil {
			return err
		}
	}
	return p.err
}

func (p *Parser) NonFlagArgs() []string {
	if p.pos < len(p.Args) {
		return p.Args[p.pos:]
	}
	return nil
}

func (p *Parser) Name(_ context.Context, fs parse.FlagSet) (func(*flag.Flag, func(string)), error) {
	return func(f *flag.Flag, it func(string)) {
		it("-" + f.Name)
	}, nil
}

func (p *Parser) reset(fs parse.FlagSet) {
	p.fs = fs
	p.pos = 0
	p.err = nil
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
	p.pos++

	minuses := 1
	if s[1] == '-' {
		minuses = 2
		if len(s) == minuses { // "--" terminates all flags.
			return false
		}
	}
	name := s[minuses:]
	if name[0] == '-' || name[0] == '=' {
		p.fail("bad flag syntax: %s", s)
		return false
	}

	name, value, hasValue := split(name, '=')
	if !hasValue && p.pos < len(p.Args) {
		value = p.Args[p.pos]
		if len(value) == 0 || value[0] != '-' {
			// NOTE: this is NOT the same behaviour as for flag.Parse().
			//       flag.Parse() works well if we pass `-flag=true`, but not
			//       if we pass `-flag true`.
			if p.isBoolFlag(name) {
				p.fail(""+
					"ambiguous boolean flag -%[1]s value: can't guess whether "+
					"the %[2]q is the flag value or the non-flag argument "+
					"(consider using `-%[1]s=%[2]s` or `-%[1]s -- %[2]s`)",
					name, value,
				)
				return false
			}
			hasValue = true
			p.pos++
		}
	}
	if !hasValue && p.isBoolFlag(name) {
		value = "true"
		hasValue = true
	}
	if !hasValue {
		p.fail("flag needs an argument: -%s", name)
		return false
	}

	p.name = name
	p.value = value
	return true
}

func (p *Parser) isBoolFlag(name string) bool {
	f := p.fs.Lookup(name)
	if f == nil && (name == "help" || name == "h") {
		// Special case for help message request.
		return true
	}
	if f == nil {
		return false
	}
	return isBoolFlag(f)
}

func (p *Parser) fail(f string, args ...interface{}) {
	p.err = fmt.Errorf("args: %s", fmt.Sprintf(f, args...))
}

func split(s string, sep byte) (a, b string, ok bool) {
	i := strings.IndexByte(s, sep)
	if i == -1 {
		return s, "", false
	}
	return s[:i], s[i+1:], true
}

func isBoolFlag(f *flag.Flag) bool {
	x, ok := f.Value.(interface {
		IsBoolFlag() bool
	})
	return ok && x.IsBoolFlag()
}
