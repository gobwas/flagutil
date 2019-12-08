package testutil

import (
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gobwas/flagutil/parse"
	"github.com/google/go-cmp/cmp"
)

type Values map[string]interface{}

func TestParser(t *testing.T, parseFunc func(Values, *parse.FlagSet) error) {
	for _, test := range []struct {
		name  string
		input Values
		setup Values
	}{
		{
			name: "basic",
			input: Values{
				"string":   "flagutil",
				"int":      42,
				"float":    3.14,
				"bool":     true,
				"b":        true,
				"duration": time.Second,
				"subset": Values{
					"foo": "bar",
				},
				"list": []string{
					"a", "b", "c",
				},
			},
		},
		{
			name: "override",
			setup: Values{
				"bar": "baz",
			},
			input: Values{
				"foo": "bar",
				"bar": "foo",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fs := flag.NewFlagSet(test.name, flag.ContinueOnError)
			fetch, input, exp, err := declare(fs, "", test.input, test.setup)
			if err != nil {
				panic(err)
			}

			err = parseFunc(input, parse.NewFlagSet(fs))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			act := fetch()
			if !cmp.Equal(act, exp) {
				t.Fatalf(
					"unexpected parsed values:\n%s",
					cmp.Diff(exp, act),
				)
			}
		})
	}
}

func declare(
	fs *flag.FlagSet, prefix string,
	values, setup Values,
) (
	fetch func() Values,
	input, exp Values,
	err error,
) {
	res := make(Values)
	exp = make(Values)
	input = make(Values)

	var funcs []func()
	appendFetch := func(f func()) {
		funcs = append(funcs, f)
	}
	defer func() {
		if err != nil {
			return
		}
		fetch = func() Values {
			for _, f := range funcs {
				f()
			}
			return res
		}
	}()

	for name, value := range values {
		var (
			name = name
			key  = join(prefix, name)
		)

		switch v := value.(type) {
		case Values:
			s, _ := setup[name].(Values)
			f, in, e, err := declare(fs, join(prefix, name), v, s)
			if err != nil {
				return nil, nil, nil, err
			}
			appendFetch(func() {
				res[name] = f()
			})
			exp[name] = e
			input[name] = in
			continue

		case []string:
			s := stringSlice{}
			fs.Var(&s, key, "")
			appendFetch(func() {
				res[name] = []string(s)
			})

		case string:
			p := new(string)
			fs.StringVar(p, key, "", "")
			appendFetch(func() {
				res[name] = *p
			})

		case time.Duration:
			p := new(time.Duration)
			fs.DurationVar(p, key, 0, "")
			appendFetch(func() {
				res[name] = *p
			})
			input[name] = v.String()

		case float64:
			p := new(float64)
			fs.Float64Var(p, key, 0, "")
			appendFetch(func() {
				res[name] = *p
			})

		case int:
			p := new(int)
			fs.IntVar(p, key, 0, "")
			appendFetch(func() {
				res[name] = *p
			})

		case bool:
			p := new(bool)
			fs.BoolVar(p, key, false, "")
			appendFetch(func() {
				res[name] = *p
			})

		default:
			return nil, nil, nil, fmt.Errorf("unexpected value type: %T", v)
		}
		if x, has := setup[name]; has {
			if err := fs.Set(key, fmt.Sprintf("%v", x)); err != nil {
				panic(err)
			}
			exp[name] = x
		} else {
			exp[name] = value
		}
		if _, has := input[name]; !has {
			input[name] = value
		}
	}

	return
}

type stringSlice []string

func (s *stringSlice) Set(x string) error {
	*s = append(*s, x)
	return nil
}

func (s stringSlice) String() string {
	return strings.Join(s, ",")
}

func join(a, b string) string {
	if a == "" {
		return b
	}
	return a + "." + b
}
