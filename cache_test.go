package wechat

import (
	"fmt"
	"github.com/astaxie/beego/cache"
	_ "github.com/astaxie/beego/cache/redis"
//	_ "github.com/garyburd/redigo/redis"
	"testing"
	"time"
)

func Test_cache(t *testing.T) {
	//	bm, err := cache.NewCache("memory", `{"interval":10}`)
	bm, err := cache.NewCache("redis", `{"conn":"127.0.0.1:6379"}`)

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

//	if v, _ := Int(bm.Get("astaxie"), err); v != 1 {
//		t.Error("get err")
//	}

	if err = bm.Put("astaxie2", "teststring", 10); err != nil {
		t.Error("set Error", err)
	}

	if v, _ := getRedisCacheString(bm.Get("astaxie2")); v != "teststring" {
		t.Error("test get string fail")
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

//func getRedisString(input interface{}) string {
//	if result, err := redis.String(input, nil); err == nil {
//		return result
//	}
//	return ""
//}
