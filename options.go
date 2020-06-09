package flagutil

import (
	"flag"
	"strings"
)

type Option interface {
	isOption()
}

// ParseOption is a generic option that can be passed to Parse().
type ParseOption interface {
	isOption()
	setupParseConfig(*config)
}

// PrintOption is a generic option that can be passed to PrintDefaults().
type PrintOption interface {
	isOption()
	setupPrintConfig(*config)
}

type ParsePrintOption interface {
	isOption()
	setupParseConfig(*config)
	setupPrintConfig(*config)
}

type parseOptionFunc func(*config)

func (fn parseOptionFunc) isOption()                  {}
func (fn parseOptionFunc) setupParseConfig(c *config) { fn(c) }

type printOptionFunc func(*config)

func (fn printOptionFunc) isOption()                  {}
func (fn printOptionFunc) setupPrintConfig(c *config) { fn(c) }

type parserOption struct {
	p *parser
}

func (p parserOption) isOption()                  {}
func (p parserOption) setupParseConfig(c *config) { c.parsers = append(c.parsers, p.p) }
func (p parserOption) setupPrintConfig(c *config) { c.parsers = append(c.parsers, p.p) }

type ParserOption interface {
	setupParserConfig(*parser)
}

type parserOptionFunc func(*parser)

func (fn parserOptionFunc) setupParserConfig(p *parser) {
	fn(p)
}

func WithIgnoreByName(names ...string) ParserOption {
	m := make(map[string]bool, len(names))
	for _, name := range names {
		m[name] = true
	}
	return parserOptionFunc(func(p *parser) {
		prev := p.ignore
		p.ignore = func(f *flag.Flag) bool {
			if prev != nil && prev(f) {
				return true
			}
			return m[f.Name]
		}
	})
}

func WithIgnoreByPrefix(prefix string) ParserOption {
	return parserOptionFunc(func(p *parser) {
		prev := p.ignore
		p.ignore = func(f *flag.Flag) bool {
			if prev != nil && prev(f) {
				return true
			}
			return strings.HasPrefix(f.Name, prefix)
		}
	})
}

// WithParser returns a parse option and makes p to be used during Parse().
func WithParser(p Parser, opts ...ParserOption) ParsePrintOption {
	x := &parser{
		Parser: p,
	}
	for _, opt := range opts {
		opt.setupParserConfig(x)
	}
	return parserOption{
		p: x,
	}
}

// WithIgnoreUndefined makes Parse() to not fail on setting undefined flag.
func WithIgnoreUndefined() ParseOption {
	return parseOptionFunc(func(c *config) {
		c.ignoreUndefined = true
	})
}

// WithIgnoreUsage makes Parse() to ignore flag.FlagSet.Usage field when
// receiving flag.ErrHelp error from some parser and print results of
// flagutil.PrintDefaults() instead.
func WithIgnoreUsage() ParseOption {
	return parseOptionFunc(func(c *config) {
		c.ignoreUsage = true
	})
}

func WithUnquoteUsageMode(m UnquoteUsageMode) PrintOption {
	return printOptionFunc(func(c *config) {
		c.unquoteUsageMode = m
	})
}
