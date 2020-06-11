package flagutil

import (
	"bytes"
	"flag"
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

	PrintDefaults(fs,
		WithParser(
			&fullParser{
				Parser: nil,
				Printer: PrinterFunc(func(fs parse.FlagSet) func(*flag.Flag, func(string)) {
					return func(f *flag.Flag, it func(string)) {
						it("MUST-IGNORE-" + strings.ToUpper(f.Name))
					}
				}),
			},
			WithStashPrefix("f"),
			WithStashName("bar"),
		),
		WithParser(
			&fullParser{
				Parser: nil,
				Printer: PrinterFunc(func(fs parse.FlagSet) func(*flag.Flag, func(string)) {
					return func(f *flag.Flag, it func(string)) {
						it("-" + string(f.Name[0]))
						it("-" + f.Name)
					}
				}),
			},
			WithStashName("foo"),
		),
		WithParser(&fullParser{
			Parser: nil,
			Printer: PrinterFunc(func(fs parse.FlagSet) func(*flag.Flag, func(string)) {
				return func(f *flag.Flag, it func(string)) {
					it("$" + strings.ToUpper(f.Name))
				}
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

func stringFlag(name, def, desc string) flag.Flag {
	fs := flag.NewFlagSet("", flag.PanicOnError)
	fs.String(name, def, desc)
	f := fs.Lookup(name)
	return *f
}
