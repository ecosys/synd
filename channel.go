package synd

import ()

type Channel struct {
	ID      int64               `json:"id"`
	Name    string              `json:"name"`
	Actions []Action            `json:"actions"`
	Config  map[string]string   `json:"config"`
	Param   map[string][]string `json:"param"`
}
