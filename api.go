package wechat

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/yaotian/wechat/cache"
	"github.com/yaotian/wechat/entry"
	"io/ioutil"
	"net/http"
	"sort"
	//	"unicode/utf8"
	"code.google.com/p/mahonia"
	"encoding/hex"
	"strings"
)

const (
	default_token_key     = "wechat.api.default.token.key"
	default_subscribe_key = "wechat.subscribe.key"
	default_jsapi_key     = "wechat.jsapi.key"
	default_oauth_token_from_code_key = "wechat.api.oauth.token.from.code.key"
	default_cache_sec     = 86400
)

type ApiError struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func NewApiError(code int, msg string) *ApiError {
	return &ApiError{ErrCode: code, ErrMsg: msg}
}

func (e *ApiError) Error() string {
	return e.ErrMsg
}

type TokenResponse struct {
	Token      string `json:"access_token"`
	Openid     string `json:"openid"`
	Expires_in int64  `json:"expires_in"`
}

type JsapiTicket struct {
	Ticket     string `json:"ticket"`
	Expires_in int64  `json:"expires_in"`
}

func checkJSError(js []byte) error {
	var errmsg ApiError
	if err := json.Unmarshal(js, &errmsg); err != nil {
		return err
	}

	if errmsg.ErrCode != 0 {
		return &errmsg
	}

	return nil
}

//This is old
type ApiClient struct {
	apptoken      string
	appid         string
	appsecret     string
	fwh_apptoken  string
	fwh_appid     string
	fwh_appsecret string
	cache         cache.Cache
}

//This is old
func NewApiClient(apptoken, appid, appsecret, fwh_apptoken, fwh_appid, fwh_appsecret string) *ApiClient {
	api := &ApiClient{apptoken: apptoken, appid: appid, appsecret: appsecret, fwh_apptoken: fwh_apptoken, fwh_appid: fwh_appid, fwh_appsecret: fwh_appsecret}
	//	ca, _ := cache.NewCache("memory", `{"interval":30}`) //30秒gc一次
	ca, _ := cache.NewCache("redisx", `{"conn":":6379"}`)
	api.cache = ca
	return api
}

//下面的ApiClient只是兼容，逐渐不会被用，看gzh_api webauth_api
func (c *ApiClient) Signature(signature, timestamp, nonce string) bool {

	strs := sort.StringSlice{c.apptoken, timestamp, nonce}
	sort.Strings(strs)
	str := ""

	for _, s := range strs {
		str += s
	}

	h := sha1.New()
	h.Write([]byte(str))

	signature_now := fmt.Sprintf("%x", h.Sum(nil))
	if signature == signature_now {
		return true
	}
	return false
}

func (c *ApiClient) GetJsTicket() (string, error) {
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
		c.cache.Put(cache_key_jsticket, jsapiTicket, int64(ti.Expires_in-10))
	}

	return jsapiTicket, nil
}

//JsAPI
func (c *ApiClient) GetJsAPISignature(timestamp, nonceStr, url string) (string, error) {
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

//OAuth 服务号获OAuth
func (c *ApiClient) GetTokenFromOAuth(code string) (string, string, error) {
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

	
	reponse, err := http.Get(fmt.Sprintf(fmt_token_url_from_oauth, c.fwh_appid, c.fwh_appsecret, code))
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
func (c *ApiClient) GetSubscriberFromOAuth(oid string, token string, subscriber *entry.Subscriber) error {
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

func (c *ApiClient) GetToken() (string, error) {
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

func (c *ApiClient) Upload() error {
	return nil
}

func (c *ApiClient) Download() error {
	return nil
}

func (c *ApiClient) GetSubscriber(oid string, subscriber *entry.Subscriber) error {
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

	token, err := c.GetToken()
	if err != nil {
		return err
	}

	var reponse *http.Response
	reponse, err = http.Get(fmt.Sprintf(fmt_userinfo_url, token, oid))
	if err != nil {
		return err
	}

	defer reponse.Body.Close()

	data, _ := ioutil.ReadAll(reponse.Body)
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

func (c *ApiClient) ListSubscribers() error {
	return nil
}

func (c *ApiClient) CreateMenu(menu *entry.Menu) error {
	token, err := c.GetToken()
	if err != nil {
		return err
	}

	data, err := json.Marshal(menu)
	if err != nil {
		return err
	}

	re := strings.Replace(string(data), "\\u0026", "&", -1)

	//	fmt.Printf(re)

	reponse, err := http.Post(fmt.Sprintf(fmt_create_menu_url, token), "application/json;charset=utf-8", bytes.NewBufferString(re))

	if err != nil {
		return err
	}

	defer reponse.Body.Close()

	data2, _ := ioutil.ReadAll(reponse.Body)
	err = checkJSError(data2)
	if err != nil {
		return err
	}
	return nil

	//	return c.Post(fmt.Sprintf(fmt_create_menu_url, token), dst.Bytes())
}

func (c *ApiClient) GetMenu() error {
	return nil
}

func (c *ApiClient) RemoveMenu() error {
	token, err := c.GetToken()
	if err != nil {
		return err
	}

	reponse, err := http.Get(fmt.Sprintf(fmt_remove_menu_url, token))
	if err != nil {
		return err
	}
	defer reponse.Body.Close()

	data, _ := ioutil.ReadAll(reponse.Body)
	err = checkJSError(data)
	if err != nil {
		return err
	}

	return nil
}

func (c *ApiClient) Post(url string, json []byte) error {
	reponse, err := http.Post(url, "text/json", bytes.NewBuffer(json))
	if err != nil {
		return err
	}

	defer reponse.Body.Close()

	data, _ := ioutil.ReadAll(reponse.Body)
	err = checkJSError(data)
	if err != nil {
		return err
	}

	return nil
}

func (c *ApiClient) SendMessage(msg []byte) error {
	token, err := c.GetToken()
	if err != nil {
		return err
	}
	return c.Post(fmt.Sprintf(fmt_sendmessage_url, token), msg)
}

func (c *ApiClient) SendTextMessage(text *entry.TextMessage) error {
	msg, err := json.Marshal(text)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *ApiClient) SendImageMessage(image *entry.ImageMessage) error {
	msg, err := json.Marshal(image)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *ApiClient) SendVoiceMessage(voice *entry.VoiceMessage) error {
	msg, err := json.Marshal(voice)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *ApiClient) SendVideoMessage(video *entry.VideoMessage) error {
	msg, err := json.Marshal(video)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *ApiClient) SendMusicMessage(music *entry.MusicMessage) error {
	msg, err := json.Marshal(music)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *ApiClient) SendNewsMessage(news *entry.NewsMessage) error {
	msg, err := json.Marshal(news)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *ApiClient) ListGroups() error {
	return nil
}

func (c *ApiClient) CreateGroup() error {
	return nil
}
func (c *ApiClient) UpdateGroup() error {
	return nil
}
func (c *ApiClient) RemoveGroup() error {
	return nil
}
func (c *ApiClient) SearchGroup() error {
	return nil
}
func (c *ApiClient) MovetoGroup() error {
	return nil
}

func ConvertToString(src string) string {
	fmt.Println("src:", src)
	tagCoder := mahonia.GetCharset("utf-8").NewDecoder()
	_, cdata, _ := tagCoder.Translate([]byte(src), true)
	result := string(cdata)
	fmt.Println("result:", result)
	return result
}
