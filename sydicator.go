package synd

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	oauthSvc = oAuthService{}
)

func NewSyndicator(actions []*Action) (syndicator, error) {
	syn := syndicator{actions}

	return syn, nil
}

type syndicator struct {
	actions []*Action
}

func (syn *syndicator) Exec() (Report, error) {
	rep := Report{}
	var err error
	for _, a := range syn.actions {
		ch := make(chan Report)
		syn.execAction(a, ch)
		r := <-ch
		rep.Log = append(rep.Log, r.Log...)
		if !r.Success {
			break
		}
	}

	return rep, err
}
func (syn *syndicator) Async() (Report, error) {
	rep := Report{}
	rep.Success = true
	cnt := len(syn.actions)
	results := make(chan Report, cnt)

	syn.execActions(syn.actions, results)

	log.Printf("waiting for %d results", cnt)

	fin := 0

	for r := range results {
		log.Println("got result report")

		rep.Log = append(rep.Log, r.Log...)

		if !r.Success {
			//if any action fails, overall report shows failure
			rep.Success = false
		}
		fin++
		if fin == cnt {
			break
		}
	}
	log.Println("action execution complete")

	return rep, nil
}
func (syn *syndicator) execActions(acts []*Action, r chan Report) {
	for i, a := range acts {
		log.Printf("executing action: %d\n", i+1)
		go syn.execAction(a, r)
	}
	//close(r)
}
func (syn *syndicator) execAction(act *Action, r chan Report) {
	log.Println("executing action", act.Name)
	rep := Report{}

	rep.Success = true

	err := act.Validate()

	if err != nil {
		rep.Success = false
		rep.Log = append(rep.Log, err.Error())
		log.Println("action error", err)
		r <- rep
	}

	switch act.Provider.Name {
	case "smtp":
		rep, err = syn.execSmtp(act)
	case "json":
		rep, err = syn.execJson(act)
	case "xmlrpc":
		rep, err = syn.execXmlRpc(act)
	}

	log.Println("returning action report", act.Name)
	r <- rep
}

func (syn *syndicator) execSmtp(act *Action) (Report, error) {
	rep := Report{}

	// Set up authentication information.
	auth := smtp.PlainAuth(
		"",
		act.Authenticator.Param["username"],
		act.Authenticator.Param["password"],
		act.Authenticator.Param["server"],
	)

	sender := act.Config["sender"]

	body := "" +
		"To: " + sender + //for email lists, use the sender as the recip for the email clients to hide other recipients
		"\r\nSubject: " + act.Param["subject"][0] +
		"\r\n\r\n" + act.Param["body"][0]

	// Connect to the server, authenticate, set the sender and recipient,
	// and send the email all in one step.
	err := smtp.SendMail(
		act.Config["server"],
		auth,
		sender,
		act.Param["recipients"],
		[]byte(body),
	)

	if err != nil {
		rep.Success = false
		log.Fatal(err)
	}
	rep.Success = true

	return rep, nil
}
func (syn *syndicator) execJson(act *Action) (Report, error) {
	rep := Report{}

	var fullurl string

	fullurl = act.Config["url"] + act.Config["action"]

	for k, v := range act.Config {
		parm := "%" + k + "%"
		fullurl = strings.Replace(fullurl, parm, v, 1)
	}

	var body string
	var tpe string
	var err error
	var req *http.Request
	meth := act.Config["method"]

	switch meth {
	case "POSTURL":
		meth = "POST"
		tpe = "URL"
	case "POSTFORM":
		meth = "POST"
		tpe = "FORM"
	default:
		tpe = meth
	}

	switch tpe {
	case "URL":

		for k, v := range act.Param {
			parm := "%" + k + "%"
			var av string

			for _, itm := range v {
				av += fmt.Sprintf("%s=%v&", k, url.QueryEscape(itm))
			}

			av = strings.TrimRight(av, "&")

			fullurl = strings.Replace(fullurl, k+"="+parm, av, 1)

		}
		//HACK: ugh, this should be configured with the other auth
		if act.Authenticator.Name == "url" {
			for k, v := range act.Authenticator.Param {
				parm := "%" + k + "%"
				log.Println("auth parm: ", parm)
				fullurl = strings.Replace(fullurl, parm, v, 1)
			}
		}
		req, err = http.NewRequest(meth, fullurl, strings.NewReader(body))
	case "FORM":
		body = strings.Replace(url.Values(act.Param).Encode(), "+", "%20", -1)
		req, err = http.NewRequest(meth, fullurl, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	//HACK: ugh, only 'url' handled above
	switch act.Authenticator.Name {
	case "oauth":
		syn.signRequest(fullurl, meth, act, req)
	case "basic":
		log.Printf("setting basic auth with: %v\n", act.Authenticator)
		req.SetBasicAuth(act.Authenticator.Param["username"], act.Authenticator.Param["password"])
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		rep.Success = false
		log.Fatal(err)
	}
	rep.Success = true
	b, err := ioutil.ReadAll(resp.Body)
	rep.Log = append(rep.Log, string(b))

	return rep, nil
}
func (syn *syndicator) signRequest(fullurl string, method string, act *Action, req *http.Request) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := strconv.FormatInt(rand.New(rand.NewSource(time.Now().Unix())).Int63(), 10)

	sig, _ := oauthSvc.Sign(method, fullurl, timestamp, nonce,
		act.Authenticator.Param["oauth_consumer_key"], act.Authenticator.Param["oauth_consumer_secret"],
		act.Authenticator.Param["oauth_token"], act.Authenticator.Param["oauth_token_secret"],
		act.Param)

	header := "OAuth " + url.QueryEscape("oauth_consumer_key") + "=" + "\"" + url.QueryEscape(act.Authenticator.Param["oauth_consumer_key"]) + "\"" + ", "
	header += url.QueryEscape("oauth_nonce") + "=" + "\"" + url.QueryEscape(nonce) + "\"" + ", "
	header += url.QueryEscape("oauth_signature") + "=" + "\"" + url.QueryEscape(sig) + "\"" + ", "
	header += url.QueryEscape("oauth_signature_method") + "=" + "\"" + url.QueryEscape("HMAC-SHA1") + "\"" + ", "
	header += url.QueryEscape("oauth_timestamp") + "=" + "\"" + timestamp + "\"" + ", "
	header += url.QueryEscape("oauth_token") + "=" + "\"" + url.QueryEscape(act.Authenticator.Param["oauth_token"]) + "\"" + ", "
	header += url.QueryEscape("oauth_version") + "=" + "\"" + url.QueryEscape("1.0") + "\""

	req.Header.Set("Authorization", header)
}
func (syn *syndicator) execXmlRpc(act *Action) (Report, error) {
	rep := Report{}

	var fullurl string
	var body string
	//var tpe string
	var err error
	var req *http.Request
	meth := act.Config["method"]

	fullurl = act.Config["url"] + act.Config["action"]

	for k, v := range act.Config {
		parm := "%" + k + "%"
		fullurl = strings.Replace(fullurl, parm, v, 1)
	}

	body = "<methodCall>"
	body += "<methodName>wp.newPost</methodName>"
	body += "<params>"
	///NOTE: XML-RPC for wordpress needs an order
	body += "<blog_id><value><int>0</int></value></blog_id>" ///HACK: hard coded blog id to 0.  This should work for most installations.  Careful with MU when not using subdomains.
	for n, c := range act.Authenticator.Param {
		if n != "url" && n != "action" && n != "method" {
			body += fmt.Sprintf("<%s>", n)
			body += fmt.Sprintf("<value>%s</value>", c)
			body += fmt.Sprintf("</%s>", n)
		}
	}

	body += "<content>"
	body += "<value>"
	body += "<struct>"
	for n, p := range act.Param {
		body += fmt.Sprintf("<member>")
		body += fmt.Sprintf("<name>%s</name>", n)
		body += fmt.Sprintf("<value>%s</value>", p[0])
		body += fmt.Sprintf("</member>")
	}
	body += "</struct>"
	body += "</value>"
	body += "</content>"
	body += "</params>"
	body += "</methodCall>"

	req, err = http.NewRequest(meth, fullurl,
		strings.NewReader(body))

	if err != nil {
		rep.Success = false
		rep.Log = append(rep.Log, err.Error())
	} else {

		client := &http.Client{}
		resp, err := client.Do(req)

		if err != nil {
			rep.Success = false
			log.Fatal(err)
		}
		rep.Success = true
		b, err := ioutil.ReadAll(resp.Body)
		rep.Log = append(rep.Log, string(b))

	}
	return rep, err

}
