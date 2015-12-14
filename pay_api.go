package wechat

import (
	"errors"
	"github.com/astaxie/beego"
	"net/http"
)

var (
	url_unifiedOrder = "https://api.mch.weixin.qq.com/pay/unifiedorder"
	url_orderQuery   = "https://api.mch.weixin.qq.com/pay/orderquery"
	url_closeOrder   = "https://api.mch.weixin.qq.com/pay/closeorder"
)

type WeixinPayApiClient struct {
	appId      string
	mchId      string
	apiKey     string
	httpClient *http.Client
	mpClient   *WeixinMpApiClient
}

func NewWeixinPayApiClient(mchId, apiKey string, httpClient *http.Client, _mpClient *WeixinMpApiClient) *WeixinPayApiClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &WeixinPayApiClient{
		appId:      _mpClient.appid, //应该是一样的
		mchId:      mchId,
		apiKey:     apiKey,
		httpClient: httpClient,
		mpClient:   _mpClient,
	}
}

type Order struct {
	OpenId      string
	Body        string
	Attach      string
	OutTradeNum string
	TotalFee    string
	GoodsTag    string
	Ip          string
	NotifyUrl   string
}

//jsapi call pay 需要的页面配置
//主要是活的prepay_id
func (c *WeixinPayApiClient) GetJsApiPayPrepayId(order Order) (string, error) {
	input := c.CreateUnifiedOrderMap(order)
	if result, err := c.UnifiedOrder(input); err == nil { //有prepay_id
		prepayId := result["prepay_id"]
		if prepayId != "" {
			beego.Info("Get the prepay id,", prepayId)
			return prepayId, nil
		}
	} else {
		beego.Error(err, input)
		return "", err
	}

	/*
			<xml>
		   <return_code><![CDATA[SUCCESS]]></return_code>
		   <return_msg><![CDATA[OK]]></return_msg>
		   <appid><![CDATA[wx2421b1c4370ec43b]]></appid>
		   <mch_id><![CDATA[10000100]]></mch_id>
		   <nonce_str><![CDATA[IITRi8Iabbblz1Jc]]></nonce_str>
		   <sign><![CDATA[7921E432F65EB8ED0CE9755F0E86D72F]]></sign>
		   <result_code><![CDATA[SUCCESS]]></result_code>
		   <prepay_id><![CDATA[wx201411101639507cbf6ffd8b0779950874]]></prepay_id>
		   <trade_type><![CDATA[JSAPI]]></trade_type>
		    </xml>
	*/
	err := errors.New("unknow error to get prepayId")
	beego.Error(err)
	return "", err
}

func (c *WeixinPayApiClient) CreateUnifiedOrderMap(order Order) map[string]string {
	var input = make(map[string]string)
	input["appid"] = c.appId                  //设置微信分配的公众账号ID
	input["mch_id"] = c.mchId                 //设置微信支付分配的商户号
	input["nonce_str"] = GetRandomString(5)   //设置随机字符串，不长于32位。推荐随机数生成算法
	input["body"] = order.Body                //获取商品或支付单简要描述的值

	input["out_trade_no"] = order.OutTradeNum //设置商户系统内部的订单号,32个字符内、可包含字母, 其他说明见商户订单号
	input["total_fee"] = order.TotalFee       //设置订单总金额，只能为整数，详见支付金额
	input["spbill_create_ip"] = order.Ip      //设置APP和网页支付提交用户端ip，Native支付填调用微信支付API的机器IP。
	input["notify_url"] = order.NotifyUrl     //设置接收微信支付异步通知回调地址
	input["trade_type"] = "JSAPI"             //设置取值如下：JSAPI，NATIVE，APP，详细说明见参数规定
	input["openid"] = order.OpenId            //设置trade_type=JSAPI，此参数必传，用户在商户appid下的唯一标识。下单前需要调用【网页授权获取用户信息】接口获取到用户的Openid

//	input["goods_tag"] = order.GoodsTag       //设置商品标记，代金券或立减优惠功能的参数，说明详见代金券或立减优惠
//	input["detail"] = ""                      //设置商品名称明细列表
//	input["attach"] = order.Attach            //设置附加数据，在查询API和支付通知中原样返回，该字段主要用于商户携带订单的自定义数据
//	input["device_info"] = "WEB"                 //设置微信支付分配的终端设备号，商户自定义, PC网页或公众号内支付请传"WEB"
//	input["fee_type"] = "CNY"                 //设置符合ISO 4217标准的三位字母代码，默认人民币：CNY，其他值列表详见货币类型
//	input["time_start"] = GetOrderNow()       //设置订单生成时间，格式为yyyyMMddHHmmss，如2009年12月25日9点10分10秒表示为20091225091010
//	input["time_expire"] = GetOrderExpire()   //获取订单生成时间，格式为yyyyMMddHHmmss，如2009年12月25日9点10分10秒表示为20091225091010
//	input["product_id"] = ""                  //设置trade_type=NATIVE，此参数必传。此id为二维码中包含的商品ID，商户自行定义

	//sign
	sign := Sign(input, c.apiKey, nil)
	input["sign"] = sign
	beego.Info(input)	
	return input
}

//统一下单
func (c *WeixinPayApiClient) UnifiedOrder(req map[string]string) (resp map[string]string, err error) {
	return c.PostXML(url_unifiedOrder, req)
}

//查询订单
func (c *WeixinPayApiClient) OrderQuery(req map[string]string) (resp map[string]string, err error) {
	return c.PostXML(url_orderQuery, req)
}

//关闭订单
func (c *WeixinPayApiClient) CloseOrder(req map[string]string) (resp map[string]string, err error) {
	return c.PostXML(url_closeOrder, req)
}
