package file

import (
	"bytes"
	"context"
	"flag"
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
	Lookup() (io.ReadCloser, error)
}

// ErrNoFile is an returned by Lookup implementation to report that lookup
// didn't find any file to parse.
var ErrNoFile = fmt.Errorf("file: no file")

// LookupFunc is an adapter that allows the use of ordinar functions as Lookup.
type LookupFunc func() (io.ReadCloser, error)

// Lookup implements Lookup interface.
func (f LookupFunc) Lookup() (io.ReadCloser, error) {
	return f()
}

// MultiLookup holds Lookup implementations and their order.
type MultiLookup []Lookup

// Lookup implements Lookup interface.
func (ls MultiLookup) Lookup() (io.ReadCloser, error) {
	for _, l := range ls {
		rc, err := l.Lookup()
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
type FlagLookup struct {
	FlagSet *flag.FlagSet
	Name    string
}

// LookupFlag is a shortcut to build up a FlagLookup structure.
func LookupFlag(fs *flag.FlagSet, name string) *FlagLookup {
	return &FlagLookup{
		FlagSet: fs,
		Name:    name,
	}
}

// Lookup implements Lookup interface.
func (f *FlagLookup) Lookup() (io.ReadCloser, error) {
	flag := f.FlagSet.Lookup(f.Name)
	if flag == nil {
		return nil, ErrNoFile
	}
	path := flag.Value.String()
	if path == "" {
		return nil, ErrNoFile
	}
	return os.Open(path)
}

// PathLookup prepares source search on a path.
// If path is not exits it doesn't fail.
type PathLookup string

// Lookup implements Lookup interface.
func (p PathLookup) Lookup() (io.ReadCloser, error) {
	info, err := os.Stat(string(p))
	if os.IsNotExist(err) {
		return nil, ErrNoFile
	}
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf(
			"file: can't parse %s since its dir",
			p,
		)
	}
	return os.Open(string(p))
}

// BytesLookup succeeds source lookup with itself.
type BytesLookup []byte

// Lookup implements Lookup interface.
func (b BytesLookup) Lookup() (io.ReadCloser, error) {
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
func (p *Parser) Parse(_ context.Context, fs parse.FlagSet) error {
	bts, err := p.readSource()
	if err == ErrNoFile {
		if p.Required {
			err = fmt.Errorf("file: source not found")
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
		return fmt.Errorf("file: syntax error: %v", err)
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

func (p *Parser) readSource() ([]byte, error) {
	src, err := p.Lookup.Lookup()
	if err != nil {
		return nil, err
	}
	defer src.Close()
	return ioutil.ReadAll(src)
}
