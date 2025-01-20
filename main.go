package main

import (
	"os"
	"redis-proxy/src"
	"strings"
)

func main() {
	local_addr := "127.0.0.1:28080"
	if os.Getenv("LOCAL_ADDR") != "" {
		local_addr = os.Getenv("LOCAL_ADDR")
	}
	redis_host_list := []string{"rds.internal.pengbei.net:6379"}
	if os.Getenv("REDIS_HOST_LIST") != "" {
		redis_host_list = strings.Split(os.Getenv("REDIS_HOST_LIST"), ",")
	}
	redis_passwrod := "rds_PWD"
	if os.Getenv("REDIS_PASSWORD") != "" {
		redis_passwrod = os.Getenv("REDIS_PASSWORD")
	}
	src.RunProxy(local_addr, redis_host_list, redis_passwrod)
}
