package prompt

import (
	"context"
	"flag"
	"fmt"

	"github.com/gobwas/flagutil/parse"
	"github.com/gobwas/prompt"
)

type FlagInfo struct {
	Message  string
	Options  []string
	Multiple bool
	Boolean  bool
}

type FlagInfoMapper interface {
	FlagInfo(context.Context, *flag.Flag) (FlagInfo, error)
}

type FlagInfoFunc func(context.Context, *flag.Flag) (FlagInfo, error)

func (fn FlagInfoFunc) FlagInfo(ctx context.Context, f *flag.Flag) (FlagInfo, error) {
	return fn(ctx, f)
}

type FlagInfoMap map[string]FlagInfo

func (m FlagInfoMap) FlagInfo(_ context.Context, f *flag.Flag) (FlagInfo, error) {
	return m[f.Name], nil
}

var DefaultMessage = func(f *flag.Flag, c FlagInfo) string {
	m := "Specify " + f.Name + " value"
	if c.Multiple {
		m += "(s)"
	}
	m += " (" + f.Usage + ")"
	return m
}

type Parser struct {
	Retry    bool
	FlagInfo FlagInfoMapper
	Message  func(*flag.Flag, FlagInfo) string
}

func (p *Parser) Parse(ctx context.Context, fs parse.FlagSet) (err error) {
	fs.VisitUnspecified(func(f *flag.Flag) {
		if err != nil {
			return
		}
	repeat:
		var set []string
		set, err = p.values(ctx, f)
		if err != nil {
			return
		}
		for i, s := range set {
			err = fs.Set(f.Name, s)
			if err != nil && p.Retry && i == 0 {
				// i == 0 is required to not leave partially configured flags.
				fmt.Println(err)
				goto repeat
			}
			if err != nil {
				break
			}
		}
	})
	return err
}

func (p *Parser) values(ctx context.Context, f *flag.Flag) ([]string, error) {
	cfg, err := p.info(ctx, f)
	if err != nil {
		return nil, err
	}
	switch {
	case cfg.Options != nil:
		return p.opt(ctx, f, cfg)

	case cfg.Boolean || isBoolFlag(f):
		return p.confirm(ctx, f, cfg)

	default:
		return p.readLine(ctx, f, cfg)
	}
}

func (p *Parser) info(ctx context.Context, f *flag.Flag) (_ FlagInfo, err error) {
	if x := p.FlagInfo; x != nil {
		return x.FlagInfo(ctx, f)
	}
	return
}

func (p *Parser) message(f *flag.Flag, c FlagInfo) string {
	if msg := c.Message; msg != "" {
		return msg
	}
	if fn := p.Message; fn != nil {
		return fn(f, c)
	}
	return DefaultMessage(f, c)
}

func (p *Parser) opt(ctx context.Context, f *flag.Flag, c FlagInfo) (set []string, err error) {
	s := prompt.Select{
		Message: p.message(f, c),
		Options: c.Options,
	}
	var xs []int
	if c.Multiple {
		xs, err = s.Multiple(ctx)
	} else {
		xs = []int{-1}
		xs[0], err = s.Single(ctx)
	}
	if err != nil {
		return nil, err
	}
	set = make([]string, len(xs))
	for i, x := range xs {
		set[i] = c.Options[x]
	}
	return set, nil
}

func (p *Parser) confirm(ctx context.Context, f *flag.Flag, c FlagInfo) (set []string, err error) {
	q := prompt.Question{
		Message: p.message(f, c),
		Strict:  true,
		Mode:    prompt.QuestionSuffix,
	}
	v, err := q.Confirm(ctx)
	if err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf("%t", v)}, nil
}

func (p *Parser) readLine(ctx context.Context, f *flag.Flag, c FlagInfo) (set []string, err error) {
	pt := prompt.Prompt{
		Message: p.message(f, c) + " ",
		Default: f.Value.String(),
	}
	line, err := pt.ReadLine(ctx)
	if err != nil {
		return nil, err
	}
	return []string{line}, nil
}

func isBoolFlag(f *flag.Flag) bool {
	x, ok := f.Value.(interface {
		IsBoolFlag() bool
	})
	return ok && x.IsBoolFlag()
}
