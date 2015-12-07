package wechat

import (
	"golang.org/x/text/transform"
	"golang.org/x/text/encoding/simplifiedchinese"
	"encoding/json"
	"fmt"
	"github.com/yaotian/wechat/entry"
	"testing"
	"bytes"
	"io/ioutil"
	//	"unicode/utf8"
	"strings"
)

func TestConvertToString(t *testing.T) {
	menu := entry.NewMenu()
	button := entry.NewButton("test")
	button.Append(entry.NewViewButton("test2", "http://mp.weixin.qq.com/s?__biz=MzI5MTA2MzQ4MA==&mid=213616969&idx=1&sn=800110519c306cc52b82f2ff1651116e#rd"))
	menu.Add(button)

	data, _ := json.Marshal(menu)
	//	fmt.Print("raw data,",string(data))

	//	test_string := '{"button":[{"name":"test","sub_button":[{"type":"view","name":"test1","url":"http://mp.weixin.qq.com/s?__biz=MzI5MTA2MzQ4MA==\u0026mid=213616969\u0026idx=1\u0026sn=800110519c306cc52b82f2ff1651116e#rd"}]}]}'
	//	test2_string := "http://mp.weixin.qq.com/s?__biz=MzI5MTA2MzQ4MA==\u0026mid=213616969\u0026idx=1\u0026sn=800110519c306cc52b82f2ff1651116e#rd"
	

//	cdata := tagCoder.ConvertString(string(data))
	for_me := transform.NewReader(bytes.NewBuffer(data), simplifiedchinese.GBK.NewEncoder())
	
	cdata, _ := ioutil.ReadAll(for_me)
	fmt.Print(string(cdata))
	fmt.Print(strings.Replace(string(cdata),"\\u0026","&",-1))

//	fmt.Println("after data,", string(cdata))

//	menu2 := entry.NewMenu()
//	json.Unmarshal(data, &menu2)
//	fmt.Print(menu2.Buttons[0].Sub[0].Url)
}


//func ConvertToString(src string, srcCode string, tagCode string) string {
//    srcCoder := mahonia.NewDecoder(srcCode)
//    srcResult := srcCoder.ConvertString(src)
//    tagCoder := mahonia.NewDecoder(tagCode)
//    _, cdata, _ := tagCoder.Translate([]byte(srcResult), true)
//    result := string(cdata)
//    return result
//}