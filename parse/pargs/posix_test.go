package pargs

import (
	"testing"

	"github.com/gobwas/flagutil/parse"
	"github.com/gobwas/flagutil/parse/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestPosixParse(t *testing.T) {
	for _, test := range []struct {
		name       string
		args       []string
		exp        [][2]string
		err        bool
		isBoolFlag map[string]bool
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
			isBoolFlag: map[string]bool{
				"a": true,
				"b": true,
				"c": true,
				"d": true,
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
			isBoolFlag: map[string]bool{
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
	} {
		t.Run(test.name, func(t *testing.T) {
			var act [][2]string
			p := Parser{
				Args: test.args,
			}
			err := p.parse(
				func(name, value string) error {
					act = append(act, [2]string{name, value})
					return nil
				},
				func(name string) bool {
					return test.isBoolFlag[name]
				},
			)
			if !test.err && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if test.err && err == nil {
				t.Fatalf("want error; got nothing")
			}
			if test.err {
				return
			}
			if !cmp.Equal(act, test.exp) {
				t.Errorf(
					"unexpected set pairs:\n%s",
					cmp.Diff(test.exp, act),
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
		return p.Parse(fs)
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
