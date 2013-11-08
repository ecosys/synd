package synd

import ()

type Authenticator struct {
	ID    int64             `json:"id"`
	Name  string            `json:"name"`
	Param map[string]string `json:"param"`
}
