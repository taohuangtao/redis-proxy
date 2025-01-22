package main

import (
	"log"
	"redis-proxy/src"
	"redis-proxy/src/config"
	"runtime"
	"time"
)

func main() {
	src.RunProxy(config.LOCAL_ADDR, config.REDIS_HOSTS, config.REDIS_PWD)

	// 定时监控协程数量
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		numGoroutines := runtime.NumGoroutine()
		log.Printf("Current number of goroutines: %d\n", numGoroutines)
	}
}
