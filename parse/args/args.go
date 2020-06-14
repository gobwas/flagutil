package args

import (
	"context"
	"flag"
	"io/ioutil"

	"github.com/gobwas/flagutil/parse"
)

type Parser struct {
	Args []string
}

func (p *Parser) Parse(_ context.Context, fs parse.FlagSet) error {
	dup := flag.NewFlagSet("", flag.ContinueOnError)
	dup.SetOutput(ioutil.Discard)
	dup.Usage = func() {}

	// To get rid of "flag provided but not defined" error by flag package we
	// need to visit all flags here, even those which provided already.
	fs.VisitAll(func(f *flag.Flag) {
		dup.Var(f.Value, f.Name, f.Usage)
	})

	return dup.Parse(p.Args)
}

func (p *Parser) Name(_ context.Context, fs parse.FlagSet) (func(*flag.Flag, func(string)), error) {
	return func(f *flag.Flag, it func(string)) {
		it("-" + f.Name)
	}, nil
}
