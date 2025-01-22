package config

import (
	"os"
	"strings"
)

var DEBUG bool
var LOCAL_ADDR string
var REDIS_HOSTS []string
var REDIS_PWD string
var ZK_HOSTS []string
var RUN_REDIS_SENTINEL bool

func init() {
	DEBUG = getDebug()
	LOCAL_ADDR = getLocalAddr()
	REDIS_HOSTS = getRedisHosts()
	REDIS_PWD = getRedisPwd()
	ZK_HOSTS = getZkHosts()
	RUN_REDIS_SENTINEL = useRunRedisSentinel()
}

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
