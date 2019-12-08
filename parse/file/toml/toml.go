package toml

import "github.com/BurntSushi/toml"

type Syntax struct {
}

func (s *Syntax) Unmarshal(p []byte) (m map[string]interface{}, err error) {
	err = toml.Unmarshal(p, &m)
	return
}
