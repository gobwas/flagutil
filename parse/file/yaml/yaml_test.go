package yaml

import (
	"bytes"
	"io"
	"testing"

	"github.com/gobwas/flagutil/parse"
	"github.com/gobwas/flagutil/parse/file"
	"github.com/gobwas/flagutil/parse/testutil"
	yaml "gopkg.in/yaml.v2"
)

func TestYAML(t *testing.T) {
	testutil.TestParser(t, func(values testutil.Values, fs parse.FlagSet) error {
		p := file.Parser{
			Source: marshal(values),
			Syntax: new(Syntax),
		}
		return p.Parse(fs)
	})
}

func marshal(values testutil.Values) io.Reader {
	bts, err := yaml.Marshal(values)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(bts)
}
