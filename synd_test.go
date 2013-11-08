package synd

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"testing"
	"time"
)

var (
	channels = make([]Channel, 0)
	cam      campaign
)

func Test_synd(t *testing.T) {
	tm := time.Now()
	ts := tm.Format("20060102150405")
	loadDemoData()
	loadUserData()

	acts := make([]*Action, 0)

	for _, ca := range cam.Actions {
		if ca.Active {
			ch := getChannel(ca.ChannelID)
			a := getAction(ca.ID, ch)
			//HACK: concat configs is weird
			for k, v := range ch.Config {
				a.Config[k] = v
			}

			//dump in a timestamp
			for _, v := range ca.Param {
				if v[0] == "testing 123" {
					v[0] = "testing " + ts
				}
			}

			a.Configure(ca.Config, ca.Param)
			acts = append(acts, a)
		}
	}

	syn, _ := NewSyndicator(acts)

	rep, err := syn.Async()

	if err != nil {
		t.Fatal(err)
		log.Println("error: ", err)
	}

	if !rep.Success {
		log.Fatal("syndication failed.")
	}

	log.Println("log: ", rep)

}

func loadUserData() {
	b, err := ioutil.ReadFile("demouser.json")
	if err != nil {
		log.Fatal("cannot read demouser.json")
	}
	err = json.Unmarshal(b, &cam)
	if err != nil {
		log.Fatal("cannot decode demouser.json", err)
	}
}

func loadDemoData() {

	b, err := ioutil.ReadFile("demo.json")
	if err != nil {
		log.Fatal("cannot read demo.json")
	}
	err = json.Unmarshal(b, &channels)
	if err != nil {
		log.Fatal("cannot decode demo.json", err)
	}

}
func getChannel(id int64) *Channel {
	for _, c := range channels {
		if c.ID == id {
			if c.Config == nil {
				c.Config = make(map[string]string)
			}
			//log.Println("return channel")
			return &c

		}
	}
	return new(Channel)
}
func getAction(id int64, ch *Channel) *Action {
	for _, a := range ch.Actions {
		if a.ID == id {
			if a.Config == nil {
				a.Config = make(map[string]string)
			}
			return &a
		}
	}
	return new(Action)
}
