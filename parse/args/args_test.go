package args

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/gobwas/flagutil"
	"github.com/gobwas/flagutil/parse"
	"github.com/gobwas/flagutil/parse/testutil"
)

var _ flagutil.Printer = new(Parser)

func TestFlagsParseArgs(t *testing.T) {
	for _, test := range []struct {
		name     string
		flags    map[string]bool
		args     []string
		expPairs [][2]string
		expArgs  []string
		err      bool
	}{
		{
			name: "basic",
			flags: map[string]bool{
				"a":   true,
				"b":   true,
				"c":   true,
				"foo": false,
				"bar": false,
			},
			args: []string{
				"-a",
				"-b=true",
				"-c=false",
				"-foo", "value",
				"--bar", "value",
				"arg1", "arg2", "arg3",
			},
			expPairs: [][2]string{
				{"a", "true"},
				{"b", "true"},
				{"c", "false"},
				{"foo", "value"},
				{"bar", "value"},
			},
			expArgs: []string{
				"arg1", "arg2", "arg3",
			},
		},
		{
			name: "flags termination",
			flags: map[string]bool{
				"param": false,
			},
			args: []string{
				"--param", "value",
				"--",
				"arg1", "arg2", "arg3",
			},
			expPairs: [][2]string{
				{"param", "value"},
			},
			expArgs: []string{
				"arg1", "arg2", "arg3",
			},
		},
		{
			name: "basic error",
			flags: map[string]bool{
				"param": false,
			},
			args: []string{
				"-param",
			},
			err: true,
		},
		{
			name: "basic error",
			flags: map[string]bool{
				"param": false,
			},
			args: []string{
				"--param",
			},
			err: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fs := testutil.StubFlagSet{
				IgnoreUndefined: true,
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
			if exp, act := test.expArgs, p.NonFlagArgs(); !cmp.Equal(act, exp) {
				t.Errorf(
					"unexpected non-flag arguments:\n%s",
					cmp.Diff(exp, act),
				)
			}
		})
	}
}

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
