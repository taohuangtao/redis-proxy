package config

import (
	"os"
	"strings"
)

var DEBUG bool = GetDebug()
var LOCAL_ADDR string = GetLocalAddr()
var REDIS_HOSTS []string = GetRedisHosts()
var REDIS_PWD string = GetRedisPwd()
var ZK_HOSTS []string = GetZkHosts()
var RUN_REDIS_SENTINEL bool = useRunRedisSentinel()

func GetDebug() bool {
	if os.Getenv("DEBUG") == "true" {
		return true
	} else {
		return false
	}
}

func GetLocalAddr() string {
	local_addr := "127.0.0.1:28080"
	if os.Getenv("LOCAL_ADDR") != "" {
		local_addr = os.Getenv("LOCAL_ADDR")
	}
	return local_addr
}
func GetRedisHosts() []string {
	redis_host_list := []string{"rds.internal.pengbei.net:6379"}
	if os.Getenv("REDIS_HOSTS") != "" {
		redis_host_list = strings.Split(os.Getenv("REDIS_HOSTS"), ",")
	}
	return redis_host_list
}
func GetRedisPwd() string {
	redis_passwrod := "rds_PWD"
	if os.Getenv("REDIS_PASSWORD") != "" {
		redis_passwrod = os.Getenv("REDIS_PASSWORD")
	}
	return redis_passwrod
}
func GetZkHosts() []string {
	zk_hosts := []string{"127.0.0.1:2181"}
	if os.Getenv("ZK_HOSTS") != "" {
		zk_hosts = strings.Split(os.Getenv("ZK_HOSTS"), ",")
	}
	return zk_hosts
}

func useRunRedisSentinel() bool {
	return os.Getenv("RUN_REDIS_SENTINEL") == "true"
}
