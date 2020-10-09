package flagutil

import (
	"flag"
	"reflect"
)

// OverrideSet returns a wrapper around v which Set() method is replaced by f.
func OverrideSet(v flag.Value, f func(string) error) flag.Value {
	return value{
		value: v,
		doSet: f,
	}
}

type flagSetPair [2]*flag.FlagSet

func (p flagSetPair) Set(name, s string) error {
	for i := 0; i < len(p); i++ {
		if p[i] == nil {
			continue
		}
		if err := p[i].Set(name, s); err != nil {
			return err
		}
	}
	return nil
}

type value struct {
	value        flag.Value
	doSet        func(string) error
	doGet        func() interface{}
	doString     func() string
	doIsBoolFlag func() bool
}

func (v value) Set(s string) error {
	if fn := v.doSet; fn != nil {
		return fn(s)
	}
	if v := v.value; v != nil {
		return v.Set(s)
	}
	return nil
}
func (v value) Get() interface{} {
	if fn := v.doGet; fn != nil {
		return fn()
	}
	if g, ok := v.value.(flag.Getter); ok {
		return g.Get()
	}
	return nil
}
func (v value) String() string {
	if fn := v.doString; fn != nil {
		return fn()
	}
	if v := v.value; v != nil {
		return v.String()
	}
	return ""
}
func (v value) IsBoolFlag() bool {
	if fn := v.doIsBoolFlag; fn != nil {
		return fn()
	}
	if b, ok := v.value.(interface {
		IsBoolFlag() bool
	}); ok {
		return b.IsBoolFlag()
	}
	return false
}

type valuePair [2]flag.Value

func (p valuePair) Set(val string) error {
	for _, v := range p {
		if err := v.Set(val); err != nil {
			return err
		}
	}
	return nil
}

func (p valuePair) Get() interface{} {
	var (
		v0 interface{}
		v1 interface{}
	)
	if g0, ok := p[0].(flag.Getter); ok {
		v0 = g0.Get()
	}
	if g1, ok := p[1].(flag.Getter); ok {
		v1 = g1.Get()
	}
	if !reflect.DeepEqual(v0, v1) {
		return nil
	}
	return v0
}

func (p valuePair) String() string {
	if p.isZero() {
		return ""
	}
	s0 := p[0].String()
	s1 := p[1].String()
	if s0 != s1 {
		return ""
	}
	return s0
}

func (p valuePair) IsBoolFlag() bool {
	if isBoolValue(p[0]) && isBoolValue(p[1]) {
		return true
	}
	return false
}

func (p valuePair) isZero() bool {
	return p == valuePair{}
}
