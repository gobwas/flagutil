package parse

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSetup(t *testing.T) {
	for _, test := range []struct {
		name  string
		pairs [][2]string
		input interface{}
		has   map[string]bool
		err   bool
	}{
		{
			name: "basic",
			input: map[string]string{
				"foo": "bar",
			},
			pairs: [][2]string{
				{"foo", "bar"},
			},
		},
		{
			name: "slice",
			input: map[string]interface{}{
				"foo": []string{
					"a", "b", "c",
				},
			},
			pairs: [][2]string{
				{"foo", "a"},
				{"foo", "b"},
				{"foo", "c"},
			},
		},
		{
			name: "nested",
			input: map[string]interface{}{
				"foo": map[string]string{
					"bar": "baz",
				},
			},
			pairs: [][2]string{
				{"foo.bar", "baz"},
			},
		},
		{
			name: "custom mapping",
			input: map[int]int{
				1: 2,
			},
			pairs: [][2]string{
				{"1", "2"},
			},
		},
		{
			name: "mapping",
			input: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": map[string]string{
						"baz": "yes",
					},
				},
			},
			has: map[string]bool{
				"foo.bar": true,
			},
			pairs: [][2]string{
				{"foo.bar", "baz:yes"},
			},
		},
		{
			name: "restrictions",
			input: map[string]interface{}{
				"slice": []interface{}{
					map[string]string{
						"foo": "bar",
					},
				},
			},
			err: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var act [][2]string
			err := Setup(test.input, VisitorFunc{
				SetFunc: func(name, value string) error {
					act = append(act, [2]string{name, value})
					return nil
				},
				HasFunc: func(name string) bool {
					return test.has[name]
				},
			})
			if test.err && err == nil {
				t.Fatalf("want error; got nothing")
			}
			if !test.err && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if exp := test.pairs; !cmp.Equal(act, exp) {
				t.Fatalf("unexpected pairs:\n%s", cmp.Diff(exp, act))
			}
		})
	}
}
