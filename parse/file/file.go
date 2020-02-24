package file

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/gobwas/flagutil/parse"
)

// Syntax is an interface capable to parse file syntax.
type Syntax interface {
	Unmarshal([]byte) (map[string]interface{}, error)
}

// Lookup is an interface to search for syntax source.
type Lookup interface {
	Lookup(parse.FlagGetter) (io.ReadCloser, error)
}

// ErrNoFile is an returned by Lookup implementation to report that lookup
// didn't find any file to parse.
var ErrNoFile = fmt.Errorf("flagutil/file: no file")

// LookupFunc is an adapter that allows the use of ordinar functions as Lookup.
type LookupFunc func() (io.ReadCloser, error)

// Lookup implements Lookup interface.
func (f LookupFunc) Lookup() (io.ReadCloser, error) {
	return f()
}

// MultiLookup holds Lookup implementations and their order.
type MultiLookup []Lookup

// Lookup implements Lookup interface.
func (ls MultiLookup) Lookup(fs parse.FlagGetter) (io.ReadCloser, error) {
	for _, l := range ls {
		rc, err := l.Lookup(fs)
		if err == ErrNoFile {
			continue
		}
		if err != nil {
			return nil, err
		}
		return rc, nil
	}
	return nil, ErrNoFile
}

// FlagLookup search for flag with equal name and interprets it as filename to
// open.
type FlagLookup string

// Lookup implements Lookup interface.
func (s FlagLookup) Lookup(fs parse.FlagGetter) (io.ReadCloser, error) {
	f := fs.Lookup(string(s))
	if f == nil {
		return nil, ErrNoFile
	}
	path := f.Value.String()
	if path == "" {
		return nil, ErrNoFile
	}
	return os.Open(path)
}

// PathLookup prepares source search in a given order of paths.
type PathLookup []string

// Lookup implements Lookup interface.
func (ps PathLookup) Lookup(fs parse.FlagGetter) (io.ReadCloser, error) {
	for _, p := range ps {
		info, err := os.Stat(p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return nil, fmt.Errorf(
				"flagutil/file: can't parse %s since its dir",
				p,
			)
		}
		return os.Open(p)
	}
	return nil, ErrNoFile
}

// BytesLookup succeeds source lookup with itself.
type BytesLookup []byte

// Lookup implements Lookup interface.
func (b BytesLookup) Lookup(fs parse.FlagGetter) (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewReader(b)), nil
}

// Parser contains options of parsing source and filling flag values.
type Parser struct {
	// Lookup contains logic of how configuration source must be opened.
	// Lookup must not be nil.
	Lookup Lookup

	// Requires makes Parser to fail if Lookup doesn't return any source.
	Required bool

	// Syntax contains logic of parsing source.
	Syntax Syntax
}

// Parse implements flagutil.Parser interface.
func (p *Parser) Parse(fs parse.FlagSet) error {
	bts, err := p.readSource(fs)
	if err == ErrNoFile {
		if p.Required {
			err = fmt.Errorf("flagutil/file: source not found")
		} else {
			err = nil
		}
	}
	if err != nil {
		return err
	}
	if len(bts) == 0 {
		return nil
	}
	x, err := p.Syntax.Unmarshal(bts)
	if err != nil {
		return err
	}
	return parse.Setup(x, parse.VisitorFunc{
		SetFunc: func(name, value string) error {
			return fs.Set(name, value)
		},
		HasFunc: func(name string) bool {
			return fs.Lookup(name) != nil
		},
	})
}

func (p *Parser) readSource(fs parse.FlagSet) ([]byte, error) {
	src, err := p.Lookup.Lookup(fs)
	if err != nil {
		return nil, err
	}
	defer src.Close()
	return ioutil.ReadAll(src)
}
