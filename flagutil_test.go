package flagutil

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/gobwas/flagutil/parse"
)

type fullParser struct {
	Parser
	Printer
}

func TestPrintUsage(t *testing.T) {
	var buf bytes.Buffer
	fs := flag.NewFlagSet("test", flag.PanicOnError)
	fs.SetOutput(&buf)
	var (
		foo string
		bar bool
		baz int
	)
	fs.StringVar(&foo, "foo", "", "`custom` description here")
	fs.BoolVar(&bar, "bar", bar, "description here")
	fs.IntVar(&baz, "baz", baz, "description here")

	PrintDefaults(context.Background(), fs,
		WithParser(
			&fullParser{
				Parser: nil,
				Printer: PrinterFunc(func(_ context.Context, fs parse.FlagSet) (func(*flag.Flag, func(string)), error) {
					return func(f *flag.Flag, it func(string)) {
						it("MUST-IGNORE-" + strings.ToUpper(f.Name))
					}, nil
				}),
			},
			WithStashPrefix("f"),
			WithStashName("bar"),
		),
		WithParser(
			&fullParser{
				Parser: nil,
				Printer: PrinterFunc(func(_ context.Context, fs parse.FlagSet) (func(*flag.Flag, func(string)), error) {
					return func(f *flag.Flag, it func(string)) {
						it("-" + string(f.Name[0]))
						it("-" + f.Name)
					}, nil
				}),
			},
			WithStashName("foo"),
		),
		WithParser(&fullParser{
			Parser: nil,
			Printer: PrinterFunc(func(_ context.Context, fs parse.FlagSet) (func(*flag.Flag, func(string)), error) {
				return func(f *flag.Flag, it func(string)) {
					it("$" + strings.ToUpper(f.Name))
				}, nil
			}),
		}),
		WithStashRegexp(regexp.MustCompile(".*baz.*")),
	)
	exp := "" +
		"  $BAR, -b, -bar\n" +
		"    \tbool\n" +
		"    \tdescription here (default false)\n" +
		"\n" +
		"  $FOO\n" + // -foo is ignored.
		"    \tcustom\n" +
		"    \tcustom description here (default \"\")\n" +
		"\n"
	if act := buf.String(); act != exp {
		t.Error(cmp.Diff(exp, act))
	}
}

func TestUnquoteUsage(t *testing.T) {
	type expMode map[UnquoteUsageMode][2]string
	for _, test := range []struct {
		name  string
		flag  flag.Flag
		modes expMode
	}{
		{
			flag: flag.Flag{
				Usage: "foo `bar` baz",
			},
			modes: expMode{
				UnquoteNothing: [2]string{
					"", "foo `bar` baz",
				},
				UnquoteQuoted: [2]string{
					"bar", "foo bar baz",
				},
				UnquoteClean: [2]string{
					"", "foo baz",
				},
			},
		},
		{
			flag: stringFlag("", "", "some kind of `hello` message"),
			modes: expMode{
				UnquoteDefault: [2]string{
					"hello", "some kind of hello message",
				},
				UnquoteInferType: [2]string{
					"string", "some kind of `hello` message",
				},
				UnquoteInferType | UnquoteClean: [2]string{
					"string", "some kind of message",
				},
			},
		},
		{
			flag: stringFlag("", "", "no quoted info"),
			modes: expMode{
				UnquoteQuoted: [2]string{
					"", "no quoted info",
				},
				UnquoteInferType: [2]string{
					"string", "no quoted info",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for mode, exp := range test.modes {
				t.Run(mode.String(), func(t *testing.T) {
					actName, actUsage := unquoteUsage(mode, &test.flag)
					if expName := exp[0]; actName != expName {
						t.Errorf("unexpected name:\n%s", cmp.Diff(expName, actName))
					}
					if expUsage := exp[1]; actUsage != expUsage {
						t.Errorf("unexpected usage:\n%s", cmp.Diff(expUsage, actUsage))
					}
				})
			}
		})
	}
}

func ExampleMerge() {
	fs := flag.NewFlagSet("superset", flag.PanicOnError)
	var (
		s0 string
		s1 string
	)
	// Setup flag in a superset.
	fs.StringVar(&s0,
		"foo", "42",
		"some flag usage here",
	)
	// Now we need to setup same flag (probably from some different place).
	// Setting it up again in a superset will cause error.
	Merge(fs, func(sub *flag.FlagSet) {
		// Notice that default value of this flag is different.
		// However, it will be discarded in favour of default value from superset.
		sub.StringVar(&s1,
			"foo", "84",
			"another flag usage here",
		)
	})

	fmt.Println(s0)
	fmt.Println(s1)

	fs.Set("foo", "34")
	fmt.Println(s0)
	fmt.Println(s1)

	flag := fs.Lookup("foo")
	fmt.Println(flag.Usage)

	// Output:
	// 42
	// 84
	// 34
	// 34
	// some flag usage here / another flag usage here
}

func ExampleMerge_different_types() {
	fs := flag.NewFlagSet("superset", flag.PanicOnError)
	var (
		s string
		i int
	)
	fs.StringVar(&s,
		"foo", "42",
		"some flag usage here",
	)
	Merge(fs, func(sub *flag.FlagSet) {
		sub.IntVar(&i,
			"foo", 84,
			"another flag usage here",
		)
	})
	fs.Set("foo", "34")
	fmt.Println(s)
	fmt.Println(i)
	// Output:
	// 34
	// 34
}

func TestMerge(t *testing.T) {
	fs := flag.NewFlagSet(t.Name(), flag.PanicOnError)
	var (
		s0 string
		s1 string
		s2 string
	)
	fs.StringVar(&s0,
		"foo", "bar",
		"superset usage",
	)
	Merge(fs, func(fs *flag.FlagSet) {
		fs.StringVar(&s1, "foo", "baz", "subset1 usage")
	})
	Merge(fs, func(fs *flag.FlagSet) {
		fs.StringVar(&s2, "foo", "baq", "subset2 usage")
	})
	if s0 == s1 || s1 == s2 {
		t.Fatalf("strings are equal: %q vs %q vs %q", s0, s1, s2)
	}
	if err := fs.Set("foo", "42"); err != nil {
		t.Fatal(err)
	}
	if s0 != "42" {
		t.Fatalf("unexpected value after Set(): %q", s0)
	}
	if s0 != s1 || s1 != s2 {
		t.Fatalf("strings are not equal: %q vs %q vs %q", s0, s1, s2)
	}

	f := fs.Lookup("foo")
	if s := f.Value.String(); s != s0 {
		t.Fatalf("String() is %q; want %q", s, s0)
	}
	if act, exp := f.Usage, "superset usage / subset1 usage / subset2 usage"; act != exp {
		t.Fatalf("unexpected usage: %q; want %q", act, exp)
	}
}

func TestCombineSets(t *testing.T) {
	var (
		nameInBoth   = "both"
		nameInFirst  = "first"
		nameInSecond = "second"
		nameUnknown  = "whoa"
	)
	var (
		fs0 = flag.NewFlagSet("FlagSet#0", flag.ContinueOnError)
		fs1 = flag.NewFlagSet("FlagSet#1", flag.ContinueOnError)
	)
	fs0.String(nameInFirst, "first-default", "")
	fs0.String(nameInBoth, "both-default-0", "")
	fs1.String(nameInBoth, "both-default-1", "")
	fs1.String(nameInSecond, "second-default", "")

	fs := CombineSets(fs0, fs1)

	mustNotBeDefined(t, fs, nameUnknown)
	mustBeEqualTo(t, fs, nameInFirst, "first-default")
	mustBeEqualTo(t, fs, nameInSecond, "second-default")
	mustBeEqualTo(t, fs, nameInBoth, "")

	mustNotSet(t, fs, nameUnknown, "want error")

	mustSet(t, fs, nameInFirst, "first")
	mustBeEqualTo(t, fs, nameInFirst, "first")
	mustBeEqualTo(t, fs0, nameInFirst, "first")

	mustSet(t, fs, nameInSecond, "second")
	mustBeEqualTo(t, fs, nameInSecond, "second")
	mustBeEqualTo(t, fs1, nameInSecond, "second")

	mustSet(t, fs, nameInBoth, "both")
	mustBeEqualTo(t, fs, nameInBoth, "both")
	mustBeEqualTo(t, fs0, nameInBoth, "both")
	mustBeEqualTo(t, fs1, nameInBoth, "both")
}

func mustNotSet(t *testing.T, fs *flag.FlagSet, name, value string) {
	if err := fs.Set(name, value); err == nil {
		t.Fatalf(
			"want error on setting flag %q value to %q: %v",
			name, value, err,
		)
	}
}

func mustSet(t *testing.T, fs *flag.FlagSet, name, value string) {
	if err := fs.Set(name, value); err != nil {
		t.Fatalf("can't set flag %q value to %q: %v", name, value, err)
	}
}

func mustBeEqualTo(t *testing.T, fs *flag.FlagSet, name, value string) {
	mustBeDefined(t, fs, name)
	if act, exp := fs.Lookup(name).Value.String(), value; act != exp {
		t.Fatalf("flag %q value is %q; want %q", name, act, exp)
	}
}

func mustNotBeDefined(t *testing.T, fs *flag.FlagSet, name string) {
	if fs.Lookup(name) != nil {
		t.Fatalf("want flag %q to not be present in set", name)
	}
}

func mustBeDefined(t *testing.T, fs *flag.FlagSet, name string) {
	if fs.Lookup(name) == nil {
		t.Fatalf("want flag %q to be present in set", name)
	}
}

func TestCombineFlags(t *testing.T) {
	for _, test := range []struct {
		name  string
		flags [2]flag.Flag
		exp   flag.Flag
		panic bool
	}{
		{
			name: "different names",
			flags: [2]flag.Flag{
				stringFlag("foo", "def", "desc#0"),
				stringFlag("bar", "def", "desc#1"),
			},
			panic: true,
		},
		{
			name: "different default values",
			flags: [2]flag.Flag{
				stringFlag("foo", "def#0", "desc#0"),
				stringFlag("foo", "def#1", "desc#1"),
			},
			exp: stringFlag("foo", "", "desc#0 / desc#1"),
		},
		{
			name: "basic",
			flags: [2]flag.Flag{
				stringFlag("foo", "def", "desc#0"),
				stringFlag("foo", "def", "desc#1"),
			},
			exp: stringFlag("foo", "def", "desc#0 / desc#1"),
		},
		{
			name: "basic",
			flags: [2]flag.Flag{
				stringFlag("foo", "def", "desc#0"),
				stringFlag("foo", "", "desc#1"),
			},
			exp: stringFlag("foo", "", "desc#0 / desc#1"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			type flagOrPanic struct {
				flag  *flag.Flag
				panic interface{}
			}
			done := make(chan flagOrPanic)
			go func() {
				defer func() {
					if p := recover(); p != nil {
						done <- flagOrPanic{
							panic: p,
						}
					}
				}()
				done <- flagOrPanic{
					flag: CombineFlags(&test.flags[0], &test.flags[1]),
				}
			}()
			x := <-done
			if !test.panic && x.panic != nil {
				t.Fatalf("panic() recovered: %s", x.panic)
			}
			if test.panic {
				if x.panic == nil {
					t.Fatalf("want panic; got nothing")
				}
				return
			}
			opts := []cmp.Option{
				cmp.Transformer("Value", func(v flag.Value) string {
					return v.String()
				}),
			}
			if act, exp := x.flag, &test.exp; !cmp.Equal(act, exp, opts...) {
				t.Errorf("unexpected flag:\n%s", cmp.Diff(exp, act, opts...))
			}
			exp := fmt.Sprintf("%x", rand.Int63())
			if err := x.flag.Value.Set(exp); err != nil {
				t.Fatalf("unexpected Set() error: %v", err)
			}
			for _, f := range test.flags {
				if act := f.Value.String(); act != exp {
					t.Errorf("unexpected flag value: %s; want %s", act, exp)
				}
			}
		})
	}
}

func stringFlag(name, def, desc string) flag.Flag {
	fs := flag.NewFlagSet("", flag.PanicOnError)
	fs.String(name, def, desc)
	f := fs.Lookup(name)
	return *f
}
