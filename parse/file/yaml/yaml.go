package yaml

import yaml "gopkg.in/yaml.v2"

type Syntax struct {
}

func (s *Syntax) Unmarshal(p []byte) (m map[string]interface{}, err error) {
	err = yaml.Unmarshal(p, &m)
	return
}
