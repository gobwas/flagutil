package env

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gobwas/flagutil/parse"
	"github.com/gobwas/flagutil/parse/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestEnvParser(t *testing.T) {
	for _, test := range []struct {
		name   string
		parser Parser
		flags  []string
		env    map[string]string
		exp    [][2]string
	}{
		{
			name: "basic",
			parser: Parser{
				Prefix: "F_",
			},
			flags: []string{
				"foo",
				"bar.baz",
				"xxx",
			},
			env: map[string]string{
				"F_FOO":      "bar",
				"F_BAR__BAZ": "qux",
			},
			exp: [][2]string{
				{"foo", "bar"},
				{"bar.baz", "qux"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var fs testutil.StubFlagSet
			for _, name := range test.flags {
				fs.AddFlag(name, "")
			}

			p := test.parser
			p.LookupEnvFunc = func(name string) (value string, has bool) {
				value, has = test.env[name]
				return
			}
			if err := p.Parse(&fs); err != nil {
				t.Fatal(err)
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

func TestEnv(t *testing.T) {
	testutil.TestParser(t, func(values testutil.Values, fs parse.FlagSet) error {
		env := marshal(values)
		p := Parser{
			LookupEnvFunc: func(name string) (value string, has bool) {
				value, has = env[name]
				return
			},
			ListSeparator: ";",
		}
		return p.Parse(fs)
	})
}

func marshal(values testutil.Values) map[string]string {
	var (
		env      = make(map[string]string)
		replacer = makeReplacer(DefaultSetSeparator, DefaultReplace)
	)
	parse.Setup(values, parse.VisitorFunc{
		SetFunc: func(name, value string) error {
			name = strings.ToUpper(replacer.Replace(name))
			prev, has := env[name]
			if has {
				value = prev + ";" + value
			}
			env[name] = value
			return nil
		},
		HasFunc: func(string) bool {
			return false
		},
	})
	fmt.Println("marshal", env)
	return env
}
