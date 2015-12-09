package wechat

import (

)

//token， 这个token可以是基础access token ,也可以是网页授权的特殊token
type TokenResponse struct {
	Token      string `json:"access_token"`
	Openid     string `json:"openid"`
	Expires_in int64  `json:"expires_in"`
}

//web jsapi需要的ticket
type JsapiTicket struct {
	Ticket     string `json:"ticket"`
	Expires_in int64  `json:"expires_in"`
}


