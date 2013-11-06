package synd

import ()

type campaign struct {
	Actions []campaignAction
}
type campaignAction struct {
	ID        int64               `json:"id"`
	Active    bool                `json:"active"`
	ChannelID int64               `json:"channelID"`
	Config    map[string]string   `json:"config"`
	Param     map[string][]string `json:"param"`
}
