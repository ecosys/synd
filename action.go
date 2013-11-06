package synd

import (
	"errors"
)

type Action struct {
	ID            int64               `json:"id"`
	Name          string              `json:"name"`
	Provider      Provider            `json:"provider"`
	Authenticator Authenticator       `json:"authenticator"`
	Config        map[string]string   `json:"config"`
	Param         map[string][]string `json:"param"`
	configured    bool
}

func (act *Action) Configure(config map[string]string, param map[string][]string) {
	if act.Config == nil {
		act.Config = make(map[string]string)
	}
	if act.Param == nil {
		act.Param = make(map[string][]string)
	}
	for k, c := range config {
		act.Config[k] = c
	}
	for k, p := range param {
		act.Param[k] = p
	}
	act.configured = true
}

func (act *Action) Validate() error {
	var err error
	if !act.configured {
		err = errors.New("action not configured")
	}
	return err
}
