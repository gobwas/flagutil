package args

import (
	"flag"
	"io/ioutil"

	"github.com/gobwas/flagutil/parse"
)

type Parser struct {
	Args []string
}

func (p *Parser) Parse(fs parse.FlagSet) error {
	dup := flag.NewFlagSet("", flag.ContinueOnError)
	dup.SetOutput(ioutil.Discard)
	dup.Usage = func() {}

	fs.VisitAll(func(f *flag.Flag) {
		dup.Var(f.Value, f.Name, f.Usage)
	})

	return dup.Parse(p.Args)
}
