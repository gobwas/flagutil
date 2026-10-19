package parse

import "fmt"

// Error is a flag a parser could not set: the name the parser used, the
// value it was given, and either that no flag is known by that name or the
// flag's own error for the value.
type Error struct {
	// Name is the name the parser used, without dashes: one of the names
	// the parser sees the flag by, the flag's own or a shorthand among
	// them.
	Name string

	// Value is what the parser tried to set the flag to.
	Value string

	// Undefined reports that no flag is known by Name; Err is nil then.
	Undefined bool

	// Err is the flag's own error for Value.
	Err error
}

// Error implements error interface.
func (e *Error) Error() string {
	if e.Undefined {
		return fmt.Sprintf("parse error: flag provided but not defined: %q", e.Name)
	}
	return fmt.Sprintf("parse error: set %q: %v", e.Name, e.Err)
}

// Unwrap returns the flag's own error.
func (e *Error) Unwrap() error {
	return e.Err
}
