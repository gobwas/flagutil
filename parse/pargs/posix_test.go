package pargs

import (
	"context"
	"flag"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/gobwas/flagutil"
	"github.com/gobwas/flagutil/parse"
	"github.com/gobwas/flagutil/parse/testutil"
)

var _ flagutil.Printer = new(Parser)

func TestPosixParse(t *testing.T) {
	for _, test := range []struct {
		name            string
		args            []string
		expPairs        [][2]string
		expArgs         []string
		err             bool
		flags           map[string]bool
		ignoreUndefined bool
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
			expPairs: [][2]string{
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
			expPairs: [][2]string{
				{"a", "true"},
				{"foo", "true"},
				{"bar", "baz"},
				{"opt", "val"},
			},
		},
		{
			name: "short booleans",
			args: []string{
				"-t=true",
				"-f=false",
			},
			flags: map[string]bool{
				"t": true,
				"f": true,
			},
			expPairs: [][2]string{
				{"t", "true"},
				{"f", "false"},
			},
		},
		{
			name: "short boolean takes no argument",
			args: []string{
				"-t", "true",
				"-f", "false",
			},
			flags: map[string]bool{
				"t": true,
				"f": true,
			},
			expPairs: [][2]string{
				{"t", "true"},
			},
			expArgs: []string{
				"true", "-f", "false",
			},
		},
		{
			name: "short cluster before operand",
			args: []string{
				"-ab", "arg",
				"-c",
			},
			flags: map[string]bool{
				"a": true,
				"b": true,
				"c": true,
			},
			expPairs: [][2]string{
				{"a", "true"},
				{"b", "true"},
			},
			expArgs: []string{
				"arg", "-c",
			},
		},
		{
			name: "short with attached argument before operand",
			args: []string{
				"-efoo", "arg",
			},
			flags: map[string]bool{
				"e": false,
			},
			expPairs: [][2]string{
				{"e", "foo"},
			},
			expArgs: []string{
				"arg",
			},
		},
		{
			name: "mixed cluster with argument in next word",
			args: []string{
				"-ao", "report.txt",
				"arg",
			},
			flags: map[string]bool{
				"a": true,
				"o": false,
			},
			expPairs: [][2]string{
				{"a", "true"},
				{"o", "report.txt"},
			},
			expArgs: []string{
				"arg",
			},
		},
		{
			name: "mixed cluster with attached argument",
			args: []string{
				"-abofoo", "arg",
			},
			flags: map[string]bool{
				"a": true,
				"b": true,
				"o": false,
			},
			expPairs: [][2]string{
				{"a", "true"},
				{"b", "true"},
				{"o", "foo"},
			},
			expArgs: []string{
				"arg",
			},
		},
		{
			name: "mixed cluster without argument",
			args: []string{
				"-ao",
			},
			flags: map[string]bool{
				"a": true,
				"o": false,
			},
			err: true,
		},
		{
			name: "dash in cluster is not a terminator",
			args: []string{
				"-a-", "--foo", "report.txt",
			},
			flags: map[string]bool{
				"a":   true,
				"foo": false,
			},
			err: true,
		},
		{
			name: "dash in cluster is not a long option",
			args: []string{
				"-a-foo",
			},
			flags: map[string]bool{
				"a":   true,
				"foo": true,
			},
			err: true,
		},
		{
			name: "mixed cluster argument is the rest",
			args: []string{
				"-aob",
			},
			flags: map[string]bool{
				"a": true,
				"o": false,
				"b": true,
			},
			expPairs: [][2]string{
				{"a", "true"},
				{"o", "b"},
			},
		},
		{
			name: "mixed cluster before option",
			args: []string{
				"-ao", "-b",
			},
			flags: map[string]bool{
				"a": true,
				"o": false,
				"b": true,
			},
			err: true,
		},
		{
			name: "boolean before dash-dash",
			args: []string{
				"--foo", "--", "file",
			},
			flags: map[string]bool{
				"foo": true,
			},
			expPairs: [][2]string{
				{"foo", "true"},
			},
			expArgs: []string{
				"file",
			},
		},
		{
			name: "long booleans",
			args: []string{
				"--foo=true",
				"--bar=false",
			},
			flags: map[string]bool{
				"foo": true,
				"bar": true,
			},
			expPairs: [][2]string{
				{"foo", "true"},
				{"bar", "false"},
			},
		},
		{
			name: "empty parameter",
			args: []string{
				"--foo", "",
			},
			flags: map[string]bool{
				"foo": false,
			},
			expPairs: [][2]string{
				{"foo", ""},
			},
		},
		{
			name: "empty operand after boolean",
			args: []string{
				"--foo", "",
			},
			flags: map[string]bool{
				"foo": true,
			},
			expPairs: [][2]string{
				{"foo", "true"},
			},
			expArgs: []string{
				"",
			},
		},
		{
			name: "last bool parameter",
			flags: map[string]bool{
				"param1": false,
				"param2": true,
			},
			args: []string{
				"--param1", "value",
				"--param2",
			},
			expPairs: [][2]string{
				{"param1", "value"},
				{"param2", "true"},
			},
		},
		{
			name: "long boolean takes no argument",
			args: []string{
				"--foo", "true",
				"--bar", "false",
			},
			flags: map[string]bool{
				"foo": true,
				"bar": true,
			},
			expPairs: [][2]string{
				{"foo", "true"},
			},
			expArgs: []string{
				"true", "--bar", "false",
			},
		},
		{
			name: "non-boolean without argument",
			args: []string{
				"--param",
			},
			flags: map[string]bool{
				"param": false,
			},
			err: true,
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
			name: "dash as long parameter value",
			args: []string{
				"--file", "-",
			},
			flags: map[string]bool{
				"file": false,
			},
			expPairs: [][2]string{
				{"file", "-"},
			},
		},
		{
			name: "dash as short parameter value",
			args: []string{
				"-f", "-",
			},
			flags: map[string]bool{
				"f": false,
			},
			expPairs: [][2]string{
				{"f", "-"},
			},
		},
		{
			name: "dash as operand after boolean",
			args: []string{
				"--foo", "-",
			},
			flags: map[string]bool{
				"foo": true,
			},
			expPairs: [][2]string{
				{"foo", "true"},
			},
			expArgs: []string{
				"-",
			},
		},
		{
			name: "dash as non-flag argument",
			args: []string{
				"--param", "value",
				"-", "arg1",
			},
			flags: map[string]bool{
				"param": false,
			},
			expPairs: [][2]string{
				{"param", "value"},
			},
			expArgs: []string{
				"-", "arg1",
			},
		},

		{
			name: "non-flag arguments basic",
			args: []string{
				"-a",
				"--param", "value",
				"arg1", "arg2", "arg3",
			},
			flags: map[string]bool{
				"a":     true,
				"param": false,
			},
			expPairs: [][2]string{
				{"a", "true"},
				{"param", "value"},
			},
			expArgs: []string{
				"arg1", "arg2", "arg3",
			},
		},
		{
			name: "non-flag arguments basic with dash-dash",
			args: []string{
				"-a",
				"--param", "value",
				"--",
				"arg1", "arg2", "arg3",
			},
			flags: map[string]bool{
				"a":     true,
				"param": false,
			},
			expPairs: [][2]string{
				{"a", "true"},
				{"param", "value"},
			},
			expArgs: []string{
				"arg1", "arg2", "arg3",
			},
		},

		{
			name:  "non-existing-short-single",
			flags: map[string]bool{},
			args:  []string{"-w"},
			err:   true,
		},
		{
			name:  "non-existing-short-multi",
			flags: map[string]bool{},
			args:  []string{"-www"},
			err:   true,
		},
		{
			name:  "non-existing-long",
			flags: map[string]bool{},
			args:  []string{"--www"},
			err:   true,
		},
		{
			name: "non-existing-long-with-value",
			args: []string{"--www=value"},
			err:  true,
		},
		{
			name:  "non-existing-long-before-option",
			args:  []string{"--www", "--foo"},
			flags: map[string]bool{"foo": true},
			err:   true,
		},
		{
			name:  "non-existing-short-before-option",
			args:  []string{"-w", "--foo"},
			flags: map[string]bool{"foo": true},
			err:   true,
		},
		{
			name:  "non-existing-in-cluster",
			args:  []string{"-aw"},
			flags: map[string]bool{"a": true},
			err:   true,
		},
		{
			name:            "ignore-non-existing-without-argument",
			args:            []string{"--www", "-w"},
			ignoreUndefined: true,
		},
		{
			name:            "ignore-non-existing-before-option",
			args:            []string{"--www", "-aw", "--foo"},
			flags:           map[string]bool{"a": true, "foo": true},
			ignoreUndefined: true,
			expPairs:        [][2]string{{"a", "true"}, {"foo", "true"}},
		},
		{
			name: "help-without-argument",
			args: []string{"--help"},
			err:  true,
		},
		{
			name:  "defined-help-without-argument",
			args:  []string{"--help"},
			flags: map[string]bool{"help": false},
			err:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fs := testutil.StubFlagSet{
				IgnoreUndefined: test.ignoreUndefined,
			}
			for name, isBool := range test.flags {
				if isBool {
					fs.AddBoolFlag(name, false)
				} else {
					fs.AddFlag(name, "")
				}
			}
			p := Parser{
				Args: test.args,
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
			if exp, act := test.expPairs, fs.Pairs(); !cmp.Equal(act, exp) {
				t.Errorf(
					"unexpected set pairs:\n%s",
					cmp.Diff(exp, act),
				)
			}
			if exp, act := test.expArgs, p.NonOptionArgs(); !cmp.Equal(act, exp) {
				t.Errorf(
					"unexpected non-flag arguments:\n%s",
					cmp.Diff(exp, act),
				)
			}
		})
	}
}

// TestShorthands runs the parser over a flag set named by Shorthands.
func TestShorthands(t *testing.T) {
	for _, test := range []struct {
		name string
		// flags maps a name to whether the flag is a bool.
		flags map[string]bool
		args  []string
		exp   map[string]string
		err   bool
	}{
		{
			name: "basic",
			flags: map[string]bool{
				"shorthand": false,
			},
			args: []string{"-s=foo"},
			exp: map[string]string{
				"shorthand": "foo",
			},
		},
		{
			name: "mixed cluster",
			flags: map[string]bool{
				"include": true,
				"lease":   false,
			},
			args: []string{"-il", "lease.json"},
			exp: map[string]string{
				"include": "true",
				"lease":   "lease.json",
			},
		},
		{
			name: "ambiguous",
			flags: map[string]bool{
				"some-foo": false,
				"some-bar": false,
			},
			args: []string{"-s=foo"},
			err:  true,
		},
		{
			name: "collision",
			flags: map[string]bool{
				"some-foo": false,
				"s":        false,
			},
			args: []string{"-s=foo"},
			exp: map[string]string{
				"s": "foo",
			},
		},
		{
			name: "only top",
			flags: map[string]bool{
				"some.foo": false,
			},
			args: []string{"-s=foo"},
			err:  true,
		},
		{
			name: "without argument",
			flags: map[string]bool{
				"output": false,
			},
			args: []string{"-o"},
			err:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			flags := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
			for name, isBool := range test.flags {
				if isBool {
					flags.Bool(name, false, "")
				} else {
					flags.String(name, "", "")
				}
			}
			fs := parse.NewFlagSet(flags)
			parse.Name(fs, Shorthands(flags))
			p := Parser{
				Args: test.args,
			}
			err := p.Parse(context.Background(), fs)
			if !test.err && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if test.err && err == nil {
				t.Fatalf("want error; got nothing")
			}
			if test.err {
				return
			}
			act := make(map[string]string)
			flags.Visit(func(f *flag.Flag) {
				act[f.Name] = f.Value.String()
			})
			if diff := cmp.Diff(test.exp, act); diff != "" {
				t.Errorf("values:\n%s", diff)
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
