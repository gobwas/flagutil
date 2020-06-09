package toml

import (
	"bytes"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/gobwas/flagutil/parse"
	"github.com/gobwas/flagutil/parse/file"
	"github.com/gobwas/flagutil/parse/testutil"
)

func TestTOML(t *testing.T) {
	testutil.TestParser(t, func(values testutil.Values, fs parse.FlagSet) error {
		p := file.Parser{
			Lookup: file.BytesLookup(marshal(values)),
			Syntax: new(Syntax),
		}
		return p.Parse(fs)
	})
}

func marshal(values testutil.Values) []byte {
	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(values)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}
