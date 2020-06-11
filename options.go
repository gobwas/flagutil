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

type ParseOptionFunc func(*config)

func (fn ParseOptionFunc) setupParseConfig(c *config) { fn(c) }

type ParserOptionFunc func(*config, *parser)

func (fn ParserOptionFunc) setupParserConfig(p *parser) { fn(nil, p) }
func (fn ParserOptionFunc) setupParseConfig(c *config)  { fn(c, nil) }

func stashFunc(check func(*flag.Flag) bool) (opt ParserOptionFunc) {
	return ParserOptionFunc(func(c *config, p *parser) {
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

func WithStashName(name string) ParserOptionFunc {
	return stashFunc(func(f *flag.Flag) bool {
		return f.Name == name
	})
}

func WithStashPrefix(prefix string) ParserOptionFunc {
	return stashFunc(func(f *flag.Flag) bool {
		return strings.HasPrefix(f.Name, prefix)
	})
}

func WithStashRegexp(re *regexp.Regexp) ParserOptionFunc {
	return stashFunc(func(f *flag.Flag) bool {
		return re.MatchString(f.Name)
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
func WithIgnoreUndefined() ParseOptionFunc {
	return ParseOptionFunc(func(c *config) {
		c.ignoreUndefined = true
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
