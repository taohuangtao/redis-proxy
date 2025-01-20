package main

import (
	"redis-proxy/src"
	"redis-proxy/src/config"
)

func main() {
	src.RunProxy(config.LOCAL_ADDR, config.REDIS_HOSTS, config.REDIS_PWD)
}
