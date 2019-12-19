package parse

import (
	"fmt"
	"reflect"
	"strconv"
)

var SetSeparator = "."

type SetupFunc func(name, value string) error

type Visitor interface {
	Set(name, value string) error
	Has(name string) bool
}

type VisitorFunc struct {
	SetFunc func(name, value string) error
	HasFunc func(name string) bool
}

func (v VisitorFunc) Set(name, value string) error {
	return v.SetFunc(name, value)
}

func (v VisitorFunc) Has(name string) bool {
	return v.HasFunc(name)
}

func Setup(x interface{}, v Visitor) error {
	return setup(v, "", x)
}

func setup(v Visitor, key string, value interface{}) error {
	var (
		typ = reflect.TypeOf(value)
		val = reflect.ValueOf(value)
	)
	switch typ.Kind() {
	case reflect.Map:
		iter := val.MapRange()
		if v.Has(key) {
			for iter.Next() {
				ks, err := stringify(iter.Key().Interface())
				if err != nil {
					return err
				}
				vs, err := stringify(iter.Value().Interface())
				if err != nil {
					return err
				}
				if err := setup(v, key, ks+":"+vs); err != nil {
					return err
				}
			}
		} else {
			for iter.Next() {
				ks, err := stringify(iter.Key().Interface())
				if err != nil {
					return err
				}
				if err := setup(v, Join(key, ks), iter.Value().Interface()); err != nil {
					return err
				}
			}
		}

	case reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			vs, err := stringify(val.Index(i).Interface())
			if err != nil {
				return err
			}
			if err := setup(v, key, vs); err != nil {
				return err
			}
		}

	default:
		str, err := stringify(value)
		if err != nil {
			return err
		}
		if str == "" {
			return fmt.Errorf("can't use empty key as flag name")
		}
		if err := v.Set(key, str); err != nil {
			return fmt.Errorf(
				"set %q (%T) as flag %q value error: %v",
				str, value, key, err,
			)
		}
	}
	return nil
}

func Join(a, b string) string {
	if a == "" {
		return b
	}
	return a + "." + b
}

func stringify(x interface{}) (string, error) {
	switch v := x.(type) {
	case
		bool,
		int,
		int8,
		int16,
		int32,
		int64,
		uint,
		uint8,
		uint16,
		uint32,
		uint64,
		uintptr:

		return fmt.Sprintf("%v", v), nil

	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil

	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil

	case string:
		return v, nil

	default:
		return "", fmt.Errorf("can't stringify %[1]v (%[1]T)", v)
	}
}
