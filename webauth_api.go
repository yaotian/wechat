package wechat

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/yaotian/wechat/cache"
	"github.com/yaotian/wechat/entry"
	"io/ioutil"
	"net/http"
)

var (
	//=============网页授权获取用户基本信息 OAuth !!！仅服务号
	//获得access token
	fmt_token_url_from_oauth string = "https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code"
	//获得user信息
	fmt_userinfo_url_from_oauth string = "https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN"

	//=============jsapi !!!仅服务号
	//获得jsapi ticket
	fmt_jsapi_token_url string = "https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=%s&type=jsapi"
)

type WebAuthClient struct {
	appid     string
	appsecret string
	cache     cache.Cache
}

func NewWebAuthClient(fwh_appid, fwh_appsecret string) *WebAuthClient {
	api := &WebAuthClient{appid: fwh_appid, appsecret: fwh_appsecret}
	//	ca, _ := cache.NewCache("memory", `{"interval":10}`) //10秒gc一次
	ca, _ := cache.NewCache("redisx", `{"conn":":6379"}`)
	api.cache = ca
	return api
}

//OAuth 服务号获OAuth code为微信服务器回调过来的code
func (c *WebAuthClient) GetTokenFromOAuth(code string) (string, string, error) {

	cache_key := c.appid + "." + default_oauth_token_from_code_key

	if c.cache != nil {
		if v := c.cache.Get(cache_key); v != nil {
			switch t := v.(type) {
			case TokenResponse:
				return t.Token, t.Openid, nil
			default:
				return "", "", fmt.Errorf("unexpected type v:", t)
			}
		}
	}

	reponse, err := http.Get(fmt.Sprintf(fmt_token_url_from_oauth, c.appid, c.appsecret, code))
	if err != nil {
		return "", "", err
	}

	defer reponse.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(reponse.Body)
	if err != nil {
		return "", "", err
	}

	err = checkJSError(data)
	if err != nil {
		return "", "", err
	}

	var tr TokenResponse
	if err = json.Unmarshal(data, &tr); err != nil {
		return "", "", err
	}

	if c.cache != nil {
		c.cache.Put(cache_key, tr, int64(tr.Expires_in-20))
	}

	return tr.Token, tr.Openid, nil
}

//OAuth 服务号获得个人信息
func (c *WebAuthClient) GetSubscriberFromOAuth(oid string, token string, subscriber *entry.Subscriber) error {
	cache_key := c.appid + "." + default_subscribe_key + "." + oid
	
	if c.cache != nil {
		if v := c.cache.Get(cache_key); v != nil {
			switch t := v.(type) {
			case []byte:
				if err := json.Unmarshal(t, subscriber); err != nil {
					return err
				} else {
					return nil
				}
			}
		}
	}

	var reponse *http.Response
	reponse, err := http.Get(fmt.Sprintf(fmt_userinfo_url_from_oauth, token, oid))
	if err != nil {
		return err
	}

	defer reponse.Body.Close()

	data, _ := ioutil.ReadAll(reponse.Body)
	fmt.Print(string(data))
	err = checkJSError(data)
	if err != nil {
		return err
	}

	if c.cache != nil {
		c.cache.Put(cache_key, data, default_cache_sec)
	}
	if err = json.Unmarshal(data, subscriber); err != nil {
		return err
	}

	return nil

}

//==============================Web Js API 支持===================
//JsAPI
func (c *WebAuthClient) GetJsAPISignature(timestamp, nonceStr, url string) (string, error) {
	//先获得jsapiTicket
	var jsapiTicket string

	//先获得jsapiTicket =================
	if t, err := c.GetJsTicket(); err != nil {
		return "", err
	} else {
		jsapiTicket = t
	}

	//签名
	n := len("jsapi_ticket=") + len(jsapiTicket) +
		len("&noncestr=") + len(nonceStr) +
		len("&timestamp=") + len(timestamp) +
		len("&url=") + len(url)

	buf := make([]byte, 0, n)

	buf = append(buf, "jsapi_ticket="...)
	buf = append(buf, jsapiTicket...)
	buf = append(buf, "&noncestr="...)
	buf = append(buf, nonceStr...)
	buf = append(buf, "&timestamp="...)
	buf = append(buf, timestamp...)
	buf = append(buf, "&url="...)
	buf = append(buf, url...)

	hashsum := sha1.Sum(buf)
	return hex.EncodeToString(hashsum[:]), nil

}

func (c *WebAuthClient) GetJsTicket() (string, error) {
	var cache_key_jsticket = c.appid + "." + default_jsapi_key
	if c.cache != nil {
		if v := c.cache.Get(cache_key_jsticket); v != nil {
			switch t := v.(type) {
			case string:
				return t, nil
			}
		}
	}

	token, err := c.GetToken()
	if err != nil {
		return "", err
	}

	var reponse *http.Response
	reponse, err = http.Get(fmt.Sprintf(fmt_jsapi_token_url, token))
	if err != nil {
		return "", err
	}

	defer reponse.Body.Close()

	data, _ := ioutil.ReadAll(reponse.Body)
	err = checkJSError(data)
	if err != nil {
		return "", err
	}
	var ti JsapiTicket
	if err = json.Unmarshal(data, &ti); err != nil {
		return "", err
	}

	jsapiTicket := ti.Ticket

	if c.cache != nil {
		c.cache.Put(cache_key_jsticket, jsapiTicket, int64(ti.Expires_in-20))
	}

	return jsapiTicket, nil
}

func (c *WebAuthClient) GetToken() (string, error) {
	cache_key := c.appid + "." + default_token_key

	if c.cache != nil {
		if v := c.cache.Get(cache_key); v != nil {
			switch t := v.(type) {
			case string:
				return t, nil
			case []byte:
				return string(t), nil
			default:
				return "", fmt.Errorf("unexpected type v:", t)
			}
		}
	}

	reponse, err := http.Get(fmt.Sprintf(fmt_token_url, c.appid, c.appsecret))
	if err != nil {
		return "", err
	}

	defer reponse.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(reponse.Body)
	if err != nil {
		return "", err
	}

	err = checkJSError(data)
	if err != nil {
		return "", err
	}

	var tr TokenResponse
	if err = json.Unmarshal(data, &tr); err != nil {
		return "", err
	}
	if c.cache != nil {
		c.cache.Put(cache_key, tr.Token, int64(tr.Expires_in-20))
	}

	return tr.Token, nil
}

//=====================End
