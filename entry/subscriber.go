package entry

import (
	"strconv"
)

/*
"subscribe": 1,
    "openid": "o6_bmjrPTlm6_2sgVt7hMZOPfL2M",
    "nickname": "Band",
    "sex": 1,
    "language": "zh_CN",
    "city": "广州",
    "province": "广东",
    "country": "中国",
    "headimgurl":    "http://wx.qlogo.cn/mmopen/g3MonUZtNHkdmzicIlibx6iaFqAc56vxLSUfpb6n5WKSYVY0ChQKkiaJSgQ1dZuTOgvLLrhJbERQQ4eMsv84eavHiaiceqxibJxCfHe/0",
   "subscribe_time": 1382694957
*/
type Subscriber struct {
	Subscribe      int    `json:"subscribe"`
	Openid         string `json:"openid"`
	Nickname       string `json:"nickname"`
	Sex            int    `json:"sex"`
	Language       string `json:"language"`
	City           string `json:"city"`
	Province       string `json:"province"`
	Country        string `json:"country"`
	Headimgurl     string `json:"headimgurl"`
	Subscribe_time int64  `json:"subscribe_time"`
	Unionid        string `json:"unionid"`
}

func (this Subscriber) String() string {
	return "Subscribe:" + strconv.Itoa(this.Subscribe) + "|\n" +
		"OpenId:" + this.Openid + "|\n" +
		"Nickname:" + this.Nickname + "|\n" +
		"Sex:" + strconv.Itoa(this.Sex) + "|\n" +
		"Language:" + this.Language + "|\n" +
		"City:" + this.City + "|\n" +
		"Province:" + this.Province + "|\n" +
		"Country:" + this.Country + "|\n" +
		"Headimgurl:" + this.Headimgurl + "|\n" +
		"Unionid:" + this.Unionid + "|\n"
}
