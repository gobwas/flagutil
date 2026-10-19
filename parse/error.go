package parse

import "fmt"

// Error is a flag a parser could not set: the flag by name, the value it
// was given, and either that no such flag is defined or the flag's own
// error for the value.
type Error struct {
	// Name is the flag's name, without dashes, as the parser read it.
	Name string

	// Value is what the parser tried to set the flag to.
	Value string

	// Undefined reports that no flag of that name is defined; Err is nil
	// then.
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
