package flagutil

import (
	"flag"
	"regexp"
	"strings"
)

type ParseOption interface {
	setupParseConfig(*config)
}

type ParserOption interface {
	setupParserConfig(*parser)
}

func WithParseOptions(opts ...ParseOption) ParseOptionFunc {
	return ParseOptionFunc(func(c *config) {
		for _, opt := range opts {
			opt.setupParseConfig(c)
		}
	})
}

type ParseOptionFunc func(*config)

func (fn ParseOptionFunc) setupParseConfig(c *config) { fn(c) }

type ParserOptionFunc func(*parser)

func (fn ParserOptionFunc) setupParserConfig(p *parser) { fn(p) }

type ParseOrParserOptionFunc func(*config, *parser)

func (fn ParseOrParserOptionFunc) setupParseConfig(c *config)  { fn(c, nil) }
func (fn ParseOrParserOptionFunc) setupParserConfig(p *parser) { fn(nil, p) }

func stashFunc(check func(*flag.Flag) bool) (opt ParseOrParserOptionFunc) {
	return ParseOrParserOptionFunc(func(c *config, p *parser) {
		if c != nil {
			c.parserOptions = append(c.parserOptions, opt)
			return
		}
		prev := p.stash
		p.stash = func(f *flag.Flag) bool {
			if prev != nil && prev(f) {
				return true
			}
			return check(f)
		}
	})
}

func WithStashName(name string) ParseOrParserOptionFunc {
	return stashFunc(func(f *flag.Flag) bool {
		return f.Name == name
	})
}

func WithStashPrefix(prefix string) ParseOrParserOptionFunc {
	return stashFunc(func(f *flag.Flag) bool {
		return strings.HasPrefix(f.Name, prefix)
	})
}

func WithStashRegexp(re *regexp.Regexp) ParseOrParserOptionFunc {
	return stashFunc(func(f *flag.Flag) bool {
		return re.MatchString(f.Name)
	})
}

func WithResetSpecified() ParserOptionFunc {
	return ParserOptionFunc(func(p *parser) {
		p.allowResetSpecified = true
	})
}

// WithParser returns a parse option and makes p to be used during Parse().
func WithParser(p Parser, opts ...ParserOption) ParseOptionFunc {
	x := &parser{
		Parser: p,
	}
	for _, opt := range opts {
		opt.setupParserConfig(x)
	}
	return ParseOptionFunc(func(c *config) {
		c.parsers = append(c.parsers, x)
	})
}

// WithIgnoreUndefined makes Parse() to not fail on setting undefined flag.
func WithIgnoreUndefined() (opt ParseOrParserOptionFunc) {
	return ParseOrParserOptionFunc(func(c *config, p *parser) {
		switch {
		case c != nil:
			c.parserOptions = append(c.parserOptions, opt)
		case p != nil:
			p.ignoreUndefined = true
		}
	})
}

func WithAllowResetSpecified() (opt ParseOrParserOptionFunc) {
	return ParseOrParserOptionFunc(func(c *config, p *parser) {
		switch {
		case c != nil:
			c.parserOptions = append(c.parserOptions, opt)
		case p != nil:
			p.allowResetSpecified = true
		}
	})
}

// WithCustomUsage makes Parse() to ignore flag.FlagSet.Usage field when
// receiving flag.ErrHelp error from some parser and print results of
// flagutil.PrintDefaults() instead.
func WithCustomUsage() ParseOptionFunc {
	return ParseOptionFunc(func(c *config) {
		c.customUsage = true
	})
}

func WithUnquoteUsageMode(m UnquoteUsageMode) ParseOptionFunc {
	return ParseOptionFunc(func(c *config) {
		c.unquoteUsageMode = m
	})
}
