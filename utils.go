package wechat

import (
	"bytes"
	"code.google.com/p/mahonia"
	"crypto/rand"
	"encoding/json"
	"encoding/xml"
	"errors"
	"github.com/astaxie/beego"
	"github.com/garyburd/redigo/redis"
	"io"
	mathRand "math/rand"
	"strconv"
	"sync"
	"time"
)

func getRedisCacheString(redis_input interface{}) (string, error) {
	if result, err := redis.String(redis_input, nil); err == nil {
		return result, nil
	} else {
		return "", err
	}
}

func getRedisCacheBytes(redis_input interface{}) ([]byte, error) {
	if result, err := redis.Bytes(redis_input, nil); err == nil {
		return result, nil
	} else {
		return []byte{}, err
	}
}

// 错误处理===================================

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

func checkJSError(js []byte) error {
	var errmsg ApiError
	if err := json.Unmarshal(js, &errmsg); err != nil {
		beego.Error(err)
		return err
	}

	if errmsg.ErrCode != 0 {
		beego.Error(errmsg)
		return &errmsg
	}

	return nil
}

//================End

func ConvertToString(src string) string {
	tagCoder := mahonia.GetCharset("utf-8").NewDecoder()
	_, cdata, _ := tagCoder.Translate([]byte(src), true)
	result := string(cdata)
	return result
}

//微信支付=======================================
var textBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 16<<10)) // 16KB
	},
}

// ParseXMLToMap parses xml reading from xmlReader and returns the first-level sub-node key-value set,
// if the first-level sub-node contains child nodes, skip it.
func ParseXMLToMap(xmlReader io.Reader) (m map[string]string, err error) {
	if xmlReader == nil {
		err = errors.New("nil xmlReader")
		return
	}

	m = make(map[string]string)
	var (
		d     = xml.NewDecoder(xmlReader)
		tk    xml.Token
		depth = 0 // current xml.Token depth
		key   string
		value bytes.Buffer
	)
	for {
		tk, err = d.Token()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}

		switch v := tk.(type) {
		case xml.StartElement:
			depth++
			switch depth {
			case 2:
				key = v.Name.Local
				value.Reset()
			case 3:
				if err = d.Skip(); err != nil {
					return
				}
				depth--
				key = "" // key == "" indicates that the node with depth==2 has children
			}
		case xml.CharData:
			if depth == 2 && key != "" {
				value.Write(v)
			}
		case xml.EndElement:
			if depth == 2 && key != "" {
				m[key] = value.String()
			}
			depth--
		}
	}
}

// FormatMapToXML marshal map[string]string to xmlWriter with xml format, the root node name is xml.
//  NOTE: This function assumes the key of m map[string]string are legitimate xml name string
//  that does not contain the required escape character!
func FormatMapToXML(xmlWriter io.Writer, m map[string]string) (err error) {
	if xmlWriter == nil {
		return errors.New("nil xmlWriter")
	}

	if _, err = io.WriteString(xmlWriter, "<xml>"); err != nil {
		return
	}

	for k, v := range m {
		if _, err = io.WriteString(xmlWriter, "<"+k+">"); err != nil {
			return
		}
		if err = xml.EscapeText(xmlWriter, []byte(v)); err != nil {
			return
		}
		if _, err = io.WriteString(xmlWriter, "</"+k+">"); err != nil {
			return
		}
	}

	if _, err = io.WriteString(xmlWriter, "</xml>"); err != nil {
		return
	}
	return
}

func GetOrderNow() string {
	//yyyyMMddHHmmss  20091225091010
	//20151212192259
	now := time.Now()
	return beego.Date(now, "YmdHis")
}

func GetOrderExpire() string {
	now := time.Now().Add(600. * time.Second)
	return beego.Date(now, "YmdHis")
}

//============End

// Random generate string
func GetRandomString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func Get10NumString() string {
	num := RandNum(1000000000, 9999999999)
	return strconv.Itoa(num)
}

func RandNum(small, big int) (result int) {
	re := generateRandomNumber(small, big, 1)
	return re[0]
}

//生成count个[start,end)结束的不重复的随机数
func generateRandomNumber(start int, end int, count int) []int {
	//范围检查
	if end < start || (end-start) < count {
		return nil
	}

	//存放结果的slice
	nums := make([]int, 0)
	//随机数生成器，加入时间戳保证每次生成的随机数不一样
	r := mathRand.New(mathRand.NewSource(time.Now().UnixNano()))
	for len(nums) < count {
		//生成随机数
		num := r.Intn((end - start)) + start

		//查重
		exist := false
		for _, v := range nums {
			if v == num {
				exist = true
				break
			}
		}

		if !exist {
			nums = append(nums, num)
		}
	}

	return nums
}
