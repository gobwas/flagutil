package args

import (
	"context"
	"testing"


	"github.com/gobwas/flagutil"
	"github.com/gobwas/flagutil/parse"
	"github.com/gobwas/flagutil/parse/testutil"
)

var _ flagutil.Printer = new(Parser)

func TestArgs(t *testing.T) {
	testutil.TestParser(t, func(values testutil.Values, fs parse.FlagSet) error {
		p := Parser{
			Args: marshal(values),
		}
		return p.Parse(context.Background(), fs)
	})
}

func marshal(values testutil.Values) (args []string) {
	parse.Setup(values, parse.VisitorFunc{
		SetFunc: func(name, value string) error {
			if value == "false" {
				args = append(args, "-"+name+"=false")
				return nil
			}
			args = append(args, "-"+name)
			if value != "true" {
				args = append(args, value)
			}
			return nil
		},
		HasFunc: func(string) bool {
			return false
		},
	})
	return args
}
