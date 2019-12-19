package file

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/gobwas/flagutil/parse"
)

type Syntax interface {
	Unmarshal([]byte) (map[string]interface{}, error)
}

type Parser struct {
	Source   io.Reader
	PathFlag string
	Required bool
	Syntax   Syntax
}

func (p *Parser) Parse(fs parse.FlagSet) error {
	bts, err := p.readSource(fs)
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
	var path string
	if f := fs.Lookup(p.PathFlag); f != nil {
		path = f.Value.String()
	}
	src := p.Source
	if src == nil && path != "" {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		src = file
	}
	if src == nil {
		if p.Required {
			return nil, fmt.Errorf("no configuration source given")
		}
		return nil, nil
	}
	return ioutil.ReadAll(src)
}
