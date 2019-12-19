package env

import (
	"flag"
	"os"
	"strings"
	"sync"

	"github.com/gobwas/flagutil"
	"github.com/gobwas/flagutil/parse"
)

var DefaultReplace = map[string]string{
	"-": "_",
}

var DefaultSetSeparator = "__"

type Parser struct {
	Prefix        string
	SetSeparator  string
	ListSeparator string
	Replace       map[string]string

	LookupEnvFunc func(string) (string, bool)

	once     sync.Once
	replacer *strings.Replacer
}

func (p *Parser) init() {
	p.once.Do(func() {
		separator := p.SetSeparator
		if separator == "" {
			separator = DefaultSetSeparator
		}
		replace := p.Replace
		if replace == nil {
			replace = DefaultReplace
		}
		p.replacer = makeReplacer(separator, replace)
	})
}

func makeReplacer(sep string, repl map[string]string) *strings.Replacer {
	var oldnew []string
	oldnew = append(oldnew,
		flagutil.SetSeparator, sep,
	)
	for old, new := range repl {
		oldnew = append(oldnew,
			old, new,
		)
	}
	return strings.NewReplacer(oldnew...)
}

func (p *Parser) Parse(fs parse.FlagSet) (err error) {
	p.init()

	set := func(f *flag.Flag, s string) {
		e := f.Value.Set(s)
		if e != nil && err == nil {
			err = e
		}
	}
	fs.VisitAll(func(f *flag.Flag) {
		name := p.Prefix + strings.ToUpper(f.Name)
		name = p.replacer.Replace(name)
		value, has := p.lookupEnv(name)
		if !has {
			return
		}
		if sep := p.ListSeparator; sep != "" {
			for _, v := range strings.Split(value, p.ListSeparator) {
				set(f, v)
			}
		} else {
			set(f, value)
		}
	})

	return err
}

func (p *Parser) lookupEnv(name string) (value string, has bool) {
	if f := p.LookupEnvFunc; f != nil {
		return f(name)
	}
	return os.LookupEnv(name)
}
