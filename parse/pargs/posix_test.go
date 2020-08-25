package pargs

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/gobwas/flagutil"
	"github.com/gobwas/flagutil/parse"
	"github.com/gobwas/flagutil/parse/testutil"
)

var _ flagutil.Printer = new(Parser)

func TestPosixParse(t *testing.T) {
	for _, test := range []struct {
		name  string
		args  []string
		exp   [][2]string
		err   bool
		flags map[string]bool

		shorthand bool
	}{
		{
			name: "short basic",
			args: []string{
				"-a",
				"-bcd",
				"-efoo",
				"-f=bar",
				"-g", "baz",
			},
			flags: map[string]bool{
				"a": true,
				"b": true,
				"c": true,
				"d": true,
				"e": false,
				"f": false,
				"g": false,
			},
			exp: [][2]string{
				{"a", "true"},
				{"b", "true"},
				{"c", "true"},
				{"d", "true"},

				{"e", "foo"},
				{"f", "bar"},
				{"g", "baz"},
			},
		},
		{
			name: "long basic",
			args: []string{
				"--a",
				"--foo",
				"--bar=baz",
				"--opt", "val",
			},
			flags: map[string]bool{
				"a":   true,
				"foo": true,
				"bar": false,
				"opt": false,
			},
			exp: [][2]string{
				{"a", "true"},
				{"foo", "true"},
				{"bar", "baz"},
				{"opt", "val"},
			},
		},
		{
			name: "boolean things",
			args: []string{
				"-t", "true",
				"-f", "false",
			},
			flags: map[string]bool{
				"t": true,
				"f": true,
			},
			exp: [][2]string{
				{"t", "true"},
				{"f", "false"},
			},
		},
		{
			name: "invalid name",
			args: []string{
				"--=foo",
			},
			err: true,
		},
		{
			name: "short ambiguous",
			args: []string{
				"-abc=foo",
			},
			err: true,
		},

		{
			name:      "shorthand basic",
			shorthand: true,
			flags: map[string]bool{
				"shorthand": false,
			},
			args: []string{
				"-s=foo",
			},
			exp: [][2]string{
				{"shorthand", "foo"},
			},
		},
		{
			name:      "shorthand ambiguous",
			shorthand: true,
			flags: map[string]bool{
				"some-foo": false,
				"some-bar": false,
			},
			args: []string{
				"-s=foo",
			},
			exp: [][2]string{
				{"s", "foo"},
			},
		},
		{
			name:      "shorthand collision",
			shorthand: true,
			flags: map[string]bool{
				"some-foo": false,
				"s":        false,
			},
			args: []string{
				"-s=foo",
			},
			exp: [][2]string{
				{"s", "foo"},
			},
		},
		{
			name:      "shorthand only top",
			shorthand: true,
			flags: map[string]bool{
				"some.foo": false,
			},
			args: []string{
				"-s=foo",
			},
			exp: [][2]string{
				{"s", "foo"},
			},
		},

		{
			name:  "non-existing",
			flags: map[string]bool{},
			args:  []string{"-w"},
			err:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var fs testutil.StubFlagSet
			for name, isBool := range test.flags {
				if isBool {
					fs.AddBoolFlag(name, false)
				} else {
					fs.AddFlag(name, "")
				}
			}
			p := Parser{
				Args:      test.args,
				Shorthand: test.shorthand,
			}
			err := p.Parse(context.Background(), &fs)
			if !test.err && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if test.err && err == nil {
				t.Fatalf("want error; got nothing")
			}
			if test.err {
				return
			}
			if exp, act := test.exp, fs.Pairs(); !cmp.Equal(act, exp) {
				t.Errorf(
					"unexpected set pairs:\n%s",
					cmp.Diff(exp, act),
				)
			}
		})
	}
}

func TestPosix(t *testing.T) {
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
			if len(name) == 1 {
				args = append(args, "-"+name)
			} else {
				args = append(args, "--"+name)
			}
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
