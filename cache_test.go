package wechat

import (
	"fmt"
	"github.com/yaotian/wechat/cache"
	"testing"
	"time"
)

func Test_cache(t *testing.T) {
	//	bm, err := cache.NewCache("memory", `{"interval":10}`)
	bm, err := cache.NewCache("redisx", `{"conn":":6379"}`)
	if err != nil {
		fmt.Printf(err.Error())
		t.Error("init err")
	}

	if bm.IsExist("astaxie") {
		t.Error("check err")
	}

	if err = bm.Put("astaxie", 1, 10); err != nil {
		t.Error("set Error", err)
	}

	time.Sleep(5 * time.Second)

	if !bm.IsExist("astaxie") {
		t.Error("check err")
	}

	time.Sleep(10 * time.Second)

	if bm.IsExist("astaxie") {
		t.Error("check err")
	}
}
