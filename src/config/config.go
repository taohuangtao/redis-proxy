package config

import (
	"os"
	"strings"
)

var DEBUG bool = getDebug()
var LOCAL_ADDR string = getLocalAddr()
var REDIS_HOSTS []string = getRedisHosts()
var REDIS_PWD string = getRedisPwd()
var ZK_HOSTS []string = getZkHosts()
var RUN_REDIS_SENTINEL bool = useRunRedisSentinel()

func getDebug() bool {
	return os.Getenv("DEBUG") == "true"
}

func getLocalAddr() string {
	local_addr := "127.0.0.1:28080"
	if os.Getenv("LOCAL_ADDR") != "" {
		local_addr = os.Getenv("LOCAL_ADDR")
	}
	return local_addr
}
func getRedisHosts() []string {
	redis_host_list := []string{"127.0.0.1:6379"}
	if os.Getenv("REDIS_HOSTS") != "" {
		redis_host_list = strings.Split(os.Getenv("REDIS_HOSTS"), ",")
	}
	return redis_host_list
}
func getRedisPwd() string {
	redis_passwrod := ""
	if os.Getenv("REDIS_PASSWORD") != "" {
		redis_passwrod = os.Getenv("REDIS_PASSWORD")
	}
	return redis_passwrod
}
func getZkHosts() []string {
	zk_hosts := []string{"127.0.0.1:2181"}
	if os.Getenv("ZK_HOSTS") != "" {
		zk_hosts = strings.Split(os.Getenv("ZK_HOSTS"), ",")
	}
	return zk_hosts
}

func useRunRedisSentinel() bool {
	return os.Getenv("RUN_REDIS_SENTINEL") == "true"
}
