package parse

import (
	"errors"
	"flag"
	"testing"
)

func TestError(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	flags.Int("n", 0, "")
	fs := NewFlagSet(flags)

	for _, test := range []struct {
		name      string
		flag      string
		value     string
		undefined bool
		msg       string
	}{
		{
			name:      "undefined flag",
			flag:      "x",
			value:     "1",
			undefined: true,
			msg:       `parse error: flag provided but not defined: "x"`,
		},
		{
			name:  "value refused",
			flag:  "n",
			value: "nope",
			msg:   `parse error: set "n": parse error`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := fs.Set(test.flag, test.value)
			var e *Error
			if !errors.As(err, &e) {
				t.Fatalf("error: %v; want *Error", err)
			}
			if e.Name != test.flag {
				t.Errorf("name: %q; want %q", e.Name, test.flag)
			}
			if e.Value != test.value {
				t.Errorf("value: %q; want %q", e.Value, test.value)
			}
			if e.Undefined != test.undefined {
				t.Errorf("undefined: %t; want %t", e.Undefined, test.undefined)
			}
			if (e.Err == nil) != test.undefined {
				t.Errorf("err: %v; want nil: %t", e.Err, test.undefined)
			}
			if act := err.Error(); act != test.msg {
				t.Errorf("message: %q; want %q", act, test.msg)
			}
		})
	}
}
