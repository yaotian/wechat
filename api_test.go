package wechat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/yaotian/wechat/entry"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io/ioutil"
	"testing"
	//	"unicode/utf8"
	"strings"
)

func TestConvertToString(t *testing.T) {
	menu := entry.NewMenu()
	button := entry.NewButton("test")
	button.Append(entry.NewViewButton("test2", "http://mp.weixin.qq.com/s?__biz=MzI5MTA2MzQ4MA==&mid=213616969&idx=1&sn=800110519c306cc52b82f2ff1651116e#rd"))
	menu.Add(button)

	data, _ := json.Marshal(menu)

	//	cdata := tagCoder.ConvertString(string(data))
	for_me := transform.NewReader(bytes.NewBuffer(data), simplifiedchinese.GBK.NewEncoder())

	cdata, _ := ioutil.ReadAll(for_me)
	fmt.Print(string(cdata))
	fmt.Print(strings.Replace(string(cdata), "\\u0026", "&", -1))

}


//func ConvertToString(src string, srcCode string, tagCode string) string {
//    srcCoder := mahonia.NewDecoder(srcCode)
//    srcResult := srcCoder.ConvertString(src)
//    tagCoder := mahonia.NewDecoder(tagCode)
//    _, cdata, _ := tagCoder.Translate([]byte(srcResult), true)
//    result := string(cdata)
//    return result
//}
