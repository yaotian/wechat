package wechat

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/cache"
	_ "github.com/astaxie/beego/cache/redis"

	"github.com/yaotian/wechat/entry"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	//	"unicode/utf8"

	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"strings"
)

var (
	//公众号可以使用AppID和AppSecret调用本接口来获取access_token
	fmt_token_url string = "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s"

	//开发者可通过OpenID来获取用户基本信息
	fmt_userinfo_url string = "https://api.weixin.qq.com/cgi-bin/user/info?access_token=%s&openid=%s&lang=zh_CN"

	//上传下传媒体信息
	fmt_upload_media_url   string = "http://file.api.weixin.qq.com/cgi-bin/media/upload?access_token=%s&type=%s"
	fmt_download_media_url string = "http://file.api.weixin.qq.com/cgi-bin/media/get?access_token=%s&media_id=%s"

	//发消息
	fmt_sendmessage_url string = "https://api.weixin.qq.com/cgi-bin/message/custom/send?access_token=%s"

	//公众号菜单创建与删除
	fmt_create_menu_url string = "https://api.weixin.qq.com/cgi-bin/menu/create?access_token=%s"
	fmt_remove_menu_url string = "https://api.weixin.qq.com/cgi-bin/menu/delete?access_token=%s"

	//jssdk
	//获得jssdk使用授权时需要的 ticket
	fmt_jsapi_token_url string = "https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=%s&type=jsapi"

	//==================服务号Only========================
	//网页授权获取用户基本信息 OAuth获得access token
	fmt_token_url_from_oauth string = "https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code"
	//网页授权获取用户基本信息获得user信息
	fmt_userinfo_url_from_oauth string = "https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN"
	//OAuth调用url获取open_id, 已经确认过头像的用户不会再有任何提示
	fmt_weboauth_snsapi_base_url     string = "https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_base&state=1#wechat_redirect"
	fmt_weboauth_snsapi_userinfo_url string = "https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_userinfo&state=1#wechat_redirect"

	fmt_template_send_url string = "https://api.weixin.qq.com/cgi-bin/message/template/send?access_token=%s"
)

const (
	default_token_key                 = "wechat.api.default.token.key"
	default_subscribe_key             = "wechat.subscribe.key"
	default_weboauth_subscribe_key    = "wechat.weboauth_subscribe.key"
	default_jsapi_key                 = "wechat.jsapi.key"
	default_oauth_token_from_code_key = "wechat.api.oauth.token.from.code.key"
	default_cache_sec                 = 86400
)

type WeixinMpApiClient struct {
	appid     string
	appsecret string
	cache     cache.Cache
}

func NewWeixinMpApiClient(appid string, appsecret string) (*WeixinMpApiClient, error) {
	api := &WeixinMpApiClient{appid: appid, appsecret: appsecret}
	if ca, err := cache.NewCache("redis", `{"conn":"127.0.0.1:6379"}`); err != nil {
		beego.Error("init cache fail", err)
		return nil, err
	} else {
		api.cache = ca
		return api, nil
	}
}

func (c *WeixinMpApiClient) GetToken() (string, error) {
	cache_key := c.appid + "." + default_token_key
	beego.Info("star to get token")
	if c.cache != nil {
		if v := c.cache.Get(cache_key); v != nil {
			if token, err := getRedisCacheString(v); err == nil && token != "" {
				beego.Info("get token from cache success", token)
				return token, nil
			} else {
				beego.Error("get token from cache fail", err)
			}
			beego.Error("This token cache is not valid, delete it")
			c.cache.Delete(cache_key)
		}
	}

	beego.Info("start to get token from weixin")
	reponse, err := http.Get(fmt.Sprintf(fmt_token_url, c.appid, c.appsecret))
	if err != nil {
		beego.Error(err)
		return "", err
	}

	defer reponse.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(reponse.Body)
	if err != nil {
		beego.Error(err)
		return "", err
	}

	err = checkJSError(data)
	if err != nil {
		beego.Error(err)
		return "", err
	}

	var tr TokenResponse
	if err = json.Unmarshal(data, &tr); err != nil {
		beego.Error(err)
		return "", err
	}

	if c.cache != nil {
		beego.Info("set the token to cache ", tr.Token, int64(tr.Expires_in-100))
		c.cache.Put(cache_key, tr.Token, int64(tr.Expires_in-100))
	}

	return tr.Token, nil
}

//有时token无效，清一下cache
func (c *WeixinMpApiClient) CleanTokenCache() {
	cache_key := c.appid + "." + default_token_key
	beego.Info("star to clean cache token")
	if c.cache != nil {
		c.cache.Delete(cache_key)
	}
}

//Jssdk ======================
//token换js ticket
func (c *WeixinMpApiClient) GetJsTicket() (string, error) {
	var cache_key_jsticket = c.appid + "." + default_jsapi_key
	if c.cache != nil {
		if v := c.cache.Get(cache_key_jsticket); v != nil {
			return getRedisCacheString(v) //ticket是string
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
		c.cache.Put(cache_key_jsticket, jsapiTicket, int64(ti.Expires_in-100))
	}

	return jsapiTicket, nil
}

//获取 jssdk 页面需要的签名
func (c *WeixinMpApiClient) GetJsAPISignature(timestamp, nonceStr, url string) (string, error) {
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

//Jssdk===================End

//从微信平台下载语音文件，文件格式是amr
func (c *WeixinMpApiClient) VoiceDownloadFromWeixin(fileSave, mediaId string) error {
	token, err := c.GetToken()
	if err != nil {
		beego.Error(err)
		return err
	}

	reponse, err := http.Get(fmt.Sprintf(fmt_download_media_url, token, mediaId))
	if err != nil {
		beego.Error(err)
		return err
	}
	defer reponse.Body.Close()

	f, err := os.Create(fileSave)
	if err != nil {
		beego.Error(err)
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, reponse.Body)
	if err != nil {
		beego.Error(err)
	}
	return err
}

//将arm文件转化为mp3格式,ubuntu需要sox支持
//sudo apt-get install lame
//sudo apt-get install sox
//sudo apt-get install libsox-fmt-mp3
//sox test.amr test.mp3
func (c *WeixinMpApiClient) VoiceAmrToMp3(amrFile, mp3File string) error {
	cmd := exec.Command("/usr/bin/sox", amrFile, mp3File)
	return cmd.Run()
}

func (c *WeixinMpApiClient) GetMediaDownloadFromWeixinUrl(mediaId string) (string, error) {
	token, err := c.GetToken()
	if err != nil {
		beego.Error(err)
		return "", err
	}
	return fmt.Sprintf(fmt_download_media_url, token, mediaId), nil
}

func (c *WeixinMpApiClient) Upload() error {
	return nil
}

func (c *WeixinMpApiClient) Download() error {
	return nil
}

func (c *WeixinMpApiClient) GetSubscriber(oid string, subscriber *entry.Subscriber) error {
	cache_key := c.appid + "." + default_subscribe_key + "." + oid
	if c.cache != nil {
		if v := c.cache.Get(cache_key); v != nil {
			if t, err := getRedisCacheBytes(v); err == nil {
				if err := json.Unmarshal(t, subscriber); err != nil {
					//do nothing
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

func (c *WeixinMpApiClient) ListSubscribers() error {
	return nil
}

func (c *WeixinMpApiClient) CreateMenu(menu *entry.Menu) error {
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

func (c *WeixinMpApiClient) GetMenu() error {
	return nil
}

func (c *WeixinMpApiClient) RemoveMenu() error {
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

func (c *WeixinMpApiClient) Post(url string, json []byte) error {
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

func (c *WeixinMpApiClient) SendMessage(msg []byte) error {
	token, err := c.GetToken()
	if err != nil {
		return err
	}
	return c.Post(fmt.Sprintf(fmt_sendmessage_url, token), msg)
}

func (c *WeixinMpApiClient) SendTextMessage(text *entry.TextMessage) error {
	msg, err := json.Marshal(text)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *WeixinMpApiClient) SendImageMessage(image *entry.ImageMessage) error {
	msg, err := json.Marshal(image)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *WeixinMpApiClient) SendVoiceMessage(voice *entry.VoiceMessage) error {
	msg, err := json.Marshal(voice)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *WeixinMpApiClient) SendVideoMessage(video *entry.VideoMessage) error {
	msg, err := json.Marshal(video)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *WeixinMpApiClient) SendMusicMessage(music *entry.MusicMessage) error {
	msg, err := json.Marshal(music)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *WeixinMpApiClient) SendNewsMessage(news *entry.NewsMessage) error {
	msg, err := json.Marshal(news)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *WeixinMpApiClient) ListGroups() error {
	return nil
}

func (c *WeixinMpApiClient) CreateGroup() error {
	return nil
}
func (c *WeixinMpApiClient) UpdateGroup() error {
	return nil
}
func (c *WeixinMpApiClient) RemoveGroup() error {
	return nil
}
func (c *WeixinMpApiClient) SearchGroup() error {
	return nil
}
func (c *WeixinMpApiClient) MovetoGroup() error {
	return nil
}

//服务号Only========================================

//网页授权获取用户 OAuth , 用回调的code换取 token,  这里的token和一般的access token不一样! 调用没有限制，不用cache
func (c *WeixinMpApiClient) GetTokenFromOAuth(code string) (string, string, error) {
	reponse, err := http.Get(fmt.Sprintf(fmt_token_url_from_oauth, c.appid, c.appsecret, code))
	if err != nil {
		beego.Error("get token from oauth fail", err)
		return "", "", err
	}

	defer reponse.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(reponse.Body)
	if err != nil {
		beego.Error("get token from oauth read body fail", err)
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

	return tr.Token, tr.Openid, nil
}

//网页授权获取用户 OAuth 服务号获得个人信息, 这个token是上一步的token ,不是通常的token
func (c *WeixinMpApiClient) GetSubscriberFromOAuth(oid string, token string, subscriber *entry.Subscriber) error {
	cache_key := c.appid + "." + default_weboauth_subscribe_key + "." + oid
	if c.cache != nil {
		if v := c.cache.Get(cache_key); v != nil {
			if t, err := getRedisCacheBytes(v); err == nil {
				if err := json.Unmarshal(t, subscriber); err != nil {
					beego.Error("get subscriber unmarshall cache fail", err)
					//do nothing
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

func (c *WeixinMpApiClient) GetOAuth_Snsapi_Base_Url(redirect_to_url string) string {
	return fmt.Sprintf(fmt_weboauth_snsapi_base_url, c.appid, url.QueryEscape(redirect_to_url))
}

func (c *WeixinMpApiClient) GetOAuth_Snsapi_Userinfo_Url(redirect_to_url string) string {
	return fmt.Sprintf(fmt_weboauth_snsapi_userinfo_url, c.appid, url.QueryEscape(redirect_to_url))
}

//发模板消息

func (c *WeixinMpApiClient) SendTemplateMsg(tmsg *entry.TemplateMessage) (err error) {
	token, err := c.GetToken()
	if err != nil {
		return
	}

	msg, err := json.Marshal(tmsg)
	if err != nil {
		return
	}
	

	postUrl := fmt.Sprintf(fmt_template_send_url, token)
	beego.Debug(token)
	beego.Debug(*tmsg)
	beego.Debug(postUrl)
	beego.Debug(string(msg))

	return c.Post(postUrl, msg)
}

//发送模板消息end

//服务号Only==================End==================

//====================工具类，和公众号本身的api无关

//在公众号后台开发者工具，开发者提供的链接在提交后，微信平台会回调过来数据验证
func Signature(token_gzh_set, signature_from_wx, timestamp_from_wx, nonce_from_wx string) bool {

	strs := sort.StringSlice{token_gzh_set, timestamp_from_wx, nonce_from_wx}
	sort.Strings(strs)
	str := ""

	for _, s := range strs {
		str += s
	}

	h := sha1.New()
	h.Write([]byte(str))

	signature_now := fmt.Sprintf("%x", h.Sum(nil))
	if signature_from_wx == signature_now {
		return true
	}
	return false
}

//======================工具类 End
