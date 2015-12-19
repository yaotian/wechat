package wechat

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"hash"
	"sort"

	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"crypto/tls"
	"github.com/astaxie/beego"
	"net"
	"time"
)

const (
	ReturnCodeSuccess = "SUCCESS"
	ReturnCodeFail    = "FAIL"
)

const (
	ResultCodeSuccess = "SUCCESS"
	ResultCodeFail    = "FAIL"
)

type Error struct {
	XMLName    struct{} `xml:"xml"                  json:"-"`
	ReturnCode string   `xml:"return_code"          json:"return_code"`
	ReturnMsg  string   `xml:"return_msg,omitempty" json:"return_msg,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("return_code: %q, return_msg: %q", e.ReturnCode, e.ReturnMsg)
}

// 微信支付签名.
//  parameters: 待签名的参数集合
//  apiKey:     API密钥
//  fn:         func() hash.Hash, 如果 fn == nil 则默认用 md5.New
func Sign(parameters map[string]string, apiKey string, fn func() hash.Hash) string {
	ks := make([]string, 0, len(parameters))
	for k := range parameters {
		if k == "sign" {
			continue
		}
		ks = append(ks, k)
	}
	sort.Strings(ks)

	if fn == nil {
		fn = md5.New
	}
	h := fn()

	buf := make([]byte, 256)
	for _, k := range ks {
		v := parameters[k]
		if v == "" {
			continue
		}

		buf = buf[:0]
		buf = append(buf, k...)
		buf = append(buf, '=')
		buf = append(buf, v...)
		buf = append(buf, '&')
		h.Write(buf)
	}
	buf = buf[:0]
	buf = append(buf, "key="...)
	buf = append(buf, apiKey...)
	h.Write(buf)

	signature := make([]byte, h.Size()*2)
	hex.Encode(signature, h.Sum(nil))
	return string(bytes.ToUpper(signature))
}

func (c *WeixinPayApiClient) PostXML(url string, req map[string]string, needSSL bool) (resp map[string]string, err error) {
	bodyBuf := textBufferPool.Get().(*bytes.Buffer)
	bodyBuf.Reset()
	defer textBufferPool.Put(bodyBuf)

	if err = FormatMapToXML(bodyBuf, req); err != nil {
		return
	}

	//需要ssl，就需要ssl client
	client := c.httpClient
	if needSSL {
		client = c.shttpClient
	}

	httpResp, err := client.Post(url, "text/xml; charset=utf-8", bodyBuf)
	if err != nil {
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		err = fmt.Errorf("http.Status: %s", httpResp.Status)
		return
	}

	respBody, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return
	}

	if resp, err = ParseXMLToMap(bytes.NewReader(respBody)); err != nil {
		return
	}

	beego.Info(resp)

	// 判断协议状态
	ReturnCode, ok := resp["return_code"]
	if !ok {
		err = errors.New("no return_code parameter")
		return
	}
	if ReturnCode != ReturnCodeSuccess {
		err = &Error{
			ReturnCode: ReturnCode,
			ReturnMsg:  resp["return_msg"],
		}
		return
	}

	// 安全考虑, 做下验证
	appId, ok := resp["appid"]
	if ok && appId != c.appId {
		err = fmt.Errorf("appid mismatch, have: %q, want: %q", appId, c.appId)
		return
	}
	mchId, ok := resp["mch_id"]
	if ok && mchId != c.mchId {
		err = fmt.Errorf("mch_id mismatch, have: %q, want: %q", mchId, c.mchId)
		return
	}

	// 认证签名
	signature1, ok := resp["sign"]
	if !ok {
		err = errors.New("no sign parameter")
		return
	}
	signature2 := Sign(resp, c.apiKey, nil)
	if signature1 != signature2 {
		err = fmt.Errorf("check signature failed, \r\ninput: %q, \r\nlocal: %q", signature1, signature2)
		return
	}
	return
}

// NewTLSHttpClient 创建支持双向证书认证的 http.Client
func NewTLSHttpClient(certFile, keyFile string) (httpClient *http.Client, err error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	httpClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig:     tlsConfig,
		},
		Timeout: 60 * time.Second,
	}
	return
}
