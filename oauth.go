package synd

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	//"fmt"
	//"log"
	"net/url"
	"sort"
	"strings"
)

type oAuthService struct {
}

func (service *oAuthService) Sign(method string, fullurl string,
	timestamp string, nonce string,
	key string, keysecret string, token string, tokensecret string,
	parameters map[string][]string) (string, error) {

	//log.Println("\n")
	//log.Println("attempting to sign")
	//log.Println("method: ", method)
	//log.Println("fullurl: ", fullurl)
	//log.Println("timestamp: ", timestamp)
	//log.Println("nonce: ", nonce)
	//log.Println("key: ", key)
	//log.Println("key secret: ", keysecret)
	//log.Println("token: ", token)
	//log.Println("token secret: ", tokensecret)
	//log.Println("params: ", parameters)
	//log.Println("\n")

	version := "1.0"

	config := make(map[string]string)
	config["oauth_consumer_key"] = url.QueryEscape(key)
	config["oauth_nonce"] = url.QueryEscape(nonce)
	config["oauth_signature_method"] = url.QueryEscape("HMAC-SHA1")
	config["oauth_timestamp"] = url.QueryEscape(timestamp)
	config["oauth_token"] = url.QueryEscape(token)
	config["oauth_version"] = url.QueryEscape(version)

	//log.Printf("params length: %d\n", len(parameters))

	for k, v := range parameters {
		if len(v) > 1 {
			panic("arrays not currently supported when building oauth arguments; they should be")
		}
		//log.Printf("printing parameters: %v\n", parameters)
		val := v[0]
		enc := url.QueryEscape(val)
		config[k] = strings.Replace(enc, "+", "%20", -1)
	}

	//fmt.Printf("\nconfig: \n%v\n", config)

	keys := make([]string, len(config))

	idx := 0
	for k, _ := range config {
		keys[idx] = k
		idx++
	}

	sort.Strings(keys)

	sorted := make(map[string]string)

	for _, k := range keys {
		sorted[k] = config[k]
	}

	encoded := ""
	max := len(sorted)
	idx = 0

	for k, v := range sorted {
		encoded = encoded + k + "=" + v
		if idx != (max - 1) {
			encoded = encoded + "&"
		}
		idx++
	}
	//log.Println("encoded params: ", encoded)

	basestring := method + "&" + url.QueryEscape(fullurl) + "&" + url.QueryEscape(encoded)

	singingkey := url.QueryEscape(keysecret) + "&" + url.QueryEscape(tokensecret)

	//based on https://github.com/mrjones/oauth/blob/master/oauth.go
	hash := hmac.New(sha1.New, []byte(singingkey))

	hash.Write([]byte(basestring))
	raw := hash.Sum(nil)
	base64sig := make([]byte, base64.StdEncoding.EncodedLen(len(raw)))
	base64.StdEncoding.Encode(base64sig, raw)

	return string(base64sig), nil

}
