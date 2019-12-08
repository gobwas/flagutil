package json

import "encoding/json"

type Syntax struct {
}

func (s *Syntax) Unmarshal(p []byte) (m map[string]interface{}, err error) {
	err = json.Unmarshal(p, &m)
	return
}
