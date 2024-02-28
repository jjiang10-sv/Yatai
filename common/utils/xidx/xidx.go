package xidx

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/xid"
)

const (
	Xid       = "xid"
	HeaderXid = "X-Xid"
)

// GenXid generates a globally unique ID
func GenXid() string {
	return xid.New().String()
}

func ExtractXidFromRequest(req *http.Request) string {
	// from header
	xidInReq := req.Header.Get(strings.ToUpper(HeaderXid))

	if xidInReq == "" {
		xidInReq = req.Header.Get(HeaderXid)
	}

	if xidInReq != "" {
		return xidInReq
	}

	// from url query
	xidInReq = req.URL.Query().Get(Xid)
	if xidInReq != "" {
		return xidInReq
	}

	if req.Body == nil {
		return ""
	}
	// from body
	// @ref: https://stackoverflow.com/questions/43021058/golang-read-request-body
	body, err := ioutil.ReadAll(req.Body)
	if err == nil {
		// body in json format
		var tmp struct {
			Xid string `json:"xid"`
		}
		err = json.Unmarshal(body, &tmp)
		if err == nil {
			xidInReq = tmp.Xid
		} else {
			// application/x-www-form-urlencoded form
			values, _ := url.ParseQuery(string(body))
			xidInReq = values.Get(Xid)
		}

		// recover request body
		req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	}
	return xidInReq
}
