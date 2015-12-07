package wechat

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/cache"
	"github.com/yaotian/wechat/entry"
	"io/ioutil"
	"net/http"
	"sort"
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
)

//所有公众号具有的功能
type GzhApiClient struct {
	apptoken  string
	appid     string
	appsecret string
	cache     cache.Cache
}

func NewGzhApiClient(apptoken, appid, appsecret string) *GzhApiClient {
	api := &GzhApiClient{apptoken: apptoken, appid: appid, appsecret: appsecret}
	ca, _ := cache.NewCache("memory", `{"interval":10}`) //10秒gc一次
	api.cache = ca
	return api
}

func (c *GzhApiClient) Signature(signature, timestamp, nonce string) bool {

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

func (c *GzhApiClient) GetToken() (string, error) {
	cache_key := c.appid + ".gzhapi." + default_token_key

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

func (c *GzhApiClient) Upload() error {
	return nil
}

func (c *GzhApiClient) Download() error {
	return nil
}

func (c *GzhApiClient) GetSubscriber(oid string, subscriber *entry.Subscriber) error {
	var cache_key = c.appid + ".gzhapi." + "sub_" + oid
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

func (c *GzhApiClient) ListSubscribers() error {
	return nil
}

func (c *GzhApiClient) CreateMenu(menu *entry.Menu) error {
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

func (c *GzhApiClient) GetMenu() error {
	return nil
}

func (c *GzhApiClient) RemoveMenu() error {
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

func (c *GzhApiClient) Post(url string, json []byte) error {
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

func (c *GzhApiClient) SendMessage(msg []byte) error {
	token, err := c.GetToken()
	if err != nil {
		return err
	}
	return c.Post(fmt.Sprintf(fmt_sendmessage_url, token), msg)
}

func (c *GzhApiClient) SendTextMessage(text *entry.TextMessage) error {
	msg, err := json.Marshal(text)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *GzhApiClient) SendImageMessage(image *entry.ImageMessage) error {
	msg, err := json.Marshal(image)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *GzhApiClient) SendVoiceMessage(voice *entry.VoiceMessage) error {
	msg, err := json.Marshal(voice)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *GzhApiClient) SendVideoMessage(video *entry.VideoMessage) error {
	msg, err := json.Marshal(video)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *GzhApiClient) SendMusicMessage(music *entry.MusicMessage) error {
	msg, err := json.Marshal(music)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *GzhApiClient) SendNewsMessage(news *entry.NewsMessage) error {
	msg, err := json.Marshal(news)
	if err != nil {
		return err
	}

	return c.SendMessage(msg)
}

func (c *GzhApiClient) ListGroups() error {
	return nil
}

func (c *GzhApiClient) CreateGroup() error {
	return nil
}
func (c *GzhApiClient) UpdateGroup() error {
	return nil
}
func (c *GzhApiClient) RemoveGroup() error {
	return nil
}
func (c *GzhApiClient) SearchGroup() error {
	return nil
}
func (c *GzhApiClient) MovetoGroup() error {
	return nil
}
