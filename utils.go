package wechat

import (
	"code.google.com/p/mahonia"
	"encoding/json"
	"github.com/garyburd/redigo/redis"
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
		return err
	}

	if errmsg.ErrCode != 0 {
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

