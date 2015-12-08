package wechat

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/yaotian/wechat/cache"
	"testing"
	"time"
)

type TestObject struct{
	Id int
	Token string
}

func Test_ObjectCache(t *testing.T){
	var object TestObject
	object.Id = 1
	object.Token = "token"
	
	bm, _ := cache.NewCache("redisx", `{"conn":":6379"}`)

	bm.Put("yaotian", redis.object, 10)
	back,_ := getRedisCacheBytes(bm.Get("yaotian"))
	fmt.Println(back.(TestObject).Id)
	
}

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


	if v, _ := redis.Int(bm.Get("astaxie"), err); v != 1 {
		t.Error("get err")
	}

	if err = bm.Put("astaxie2", "teststring", 10); err != nil {
		t.Error("set Error", err)
	}

	fmt.Println(getRedisCacheString(bm.Get("astaxie2")))
	if v := bm.Get("astaxie2"); v != "string" {
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
