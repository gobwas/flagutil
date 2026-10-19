package flagutil

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

// WithNameFunc makes a parser see each flag under the aliases fn returns for
// the flag's name, and not at all when fn returns none.
//
// The flag's own name is an alias only when fn returns it, so fn hides flags,
// renames them for one parser, gives them more names (a shorthand on the
// command line, a legacy variable in the environment), or all of these.
//
// The parser reads every alias, in order, so a flag set under two of them
// takes the later, as a repeated flag does, and help prints them in order.
//
// Several WithNameFunc chain in option order: each receives, one by one, the
// aliases the previous one returned, and none aliases end the chain. Options
// given to WithParser apply before those given to Parse, so a function given
// to Parse sees the aliases a parser's own functions produced.
//
// Two flags fn gives one alias is a programming error and panics in Parse.
func WithNameFunc(fn func(name string) []string) (opt ParseOrParserOptionFunc) {
	return ParseOrParserOptionFunc(func(c *config, p *parser) {
		if c != nil {
			c.parserOptions = append(c.parserOptions, opt)
			return
		}
		prev := p.aliases
		if prev == nil {
			p.aliases = fn
			return
		}
		p.aliases = func(name string) []string {
			var aliases []string
			for _, alias := range prev(name) {
				aliases = append(aliases, fn(alias)...)
			}
			return aliases
		}
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

// WithUsageOrder sets the comparison function used to order flags in help.
// The function receives registered flag names, without dashes or parser
// aliases.
//
// compare() returning a negative value places n0 before n1, positive value
// places n0 after n1, zero orders the names alphabetically.
//
// It applies to PrintDefaults and to Parse when using WithCustomUsage.
func WithUsageOrder(compare func(n0, n1 string) int) ParseOptionFunc {
	return ParseOptionFunc(func(c *config) {
		c.usageOrder = compare
	})
}

func WithUnquoteUsageMode(m UnquoteUsageMode) ParseOptionFunc {
	return ParseOptionFunc(func(c *config) {
		c.unquoteUsageMode = m
	})
}
